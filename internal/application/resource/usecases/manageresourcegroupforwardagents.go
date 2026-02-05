package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ManageResourceGroupForwardAgentsUseCase handles adding/removing forward agents from resource groups
type ManageResourceGroupForwardAgentsUseCase struct {
	resourceGroupRepo resource.Repository
	agentRepo         forward.AgentRepository
	planRepo          subscription.PlanRepository
	logger            logger.Interface
}

// NewManageResourceGroupForwardAgentsUseCase creates a new ManageResourceGroupForwardAgentsUseCase
func NewManageResourceGroupForwardAgentsUseCase(
	resourceGroupRepo resource.Repository,
	agentRepo forward.AgentRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *ManageResourceGroupForwardAgentsUseCase {
	return &ManageResourceGroupForwardAgentsUseCase{
		resourceGroupRepo: resourceGroupRepo,
		agentRepo:         agentRepo,
		planRepo:          planRepo,
		logger:            logger,
	}
}

// AddAgents adds forward agents to a resource group by internal ID
func (uc *ManageResourceGroupForwardAgentsUseCase) AddAgents(ctx context.Context, groupID uint, agentSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeAddAgents(ctx, group, agentSIDs)
}

// AddAgentsBySID adds forward agents to a resource group by Stripe-style SID
func (uc *ManageResourceGroupForwardAgentsUseCase) AddAgentsBySID(ctx context.Context, groupSID string, agentSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "group_sid", groupSID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeAddAgents(ctx, group, agentSIDs)
}

// executeAddAgents performs the actual add agents logic
func (uc *ManageResourceGroupForwardAgentsUseCase) executeAddAgents(ctx context.Context, group *resource.ResourceGroup, agentSIDs []string) (*dto.BatchOperationResult, error) {
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	// Verify the plan type is forward
	plan, err := uc.planRepo.GetByID(ctx, group.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get plan", "error", err, "plan_id", group.PlanID())
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found for resource group")
	}
	if !plan.PlanType().IsForward() {
		uc.logger.Warnw("attempted to add forward agents to non-forward plan resource group",
			"group_id", group.ID(),
			"plan_id", group.PlanID(),
			"plan_type", plan.PlanType().String())
		return nil, resource.ErrGroupPlanTypeMismatchForward
	}

	result := &dto.BatchOperationResult{
		Succeeded: make([]string, 0),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	// Validate SID formats and collect valid SIDs for batch fetch
	validSIDs := make([]string, 0, len(agentSIDs))
	for _, agentSID := range agentSIDs {
		if err := id.ValidatePrefix(agentSID, id.PrefixForwardAgent); err != nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "invalid agent ID format",
			})
			continue
		}
		validSIDs = append(validSIDs, agentSID)
	}

	// Batch fetch all valid agents
	agentMap := make(map[string]*forward.ForwardAgent)
	if len(validSIDs) > 0 {
		agents, err := uc.agentRepo.GetBySIDs(ctx, validSIDs)
		if err != nil {
			uc.logger.Errorw("failed to batch get forward agents", "error", err)
			// Fall back to marking all as failed
			for _, sid := range validSIDs {
				result.Failed = append(result.Failed, dto.BatchOperationErr{
					ID:     sid,
					Reason: "failed to get forward agent",
				})
			}
			return result, nil
		}
		for _, agent := range agents {
			agentMap[agent.SID()] = agent
		}
	}

	// Process each valid SID and collect agents that need to be updated
	groupID := group.ID()
	agentsToUpdate := make(map[uint][]uint) // agentID -> new groupIDs
	sidToID := make(map[uint]string)        // For mapping back to SIDs
	succeededSIDs := make([]string, 0)

	for _, agentSID := range validSIDs {
		agent, ok := agentMap[agentSID]
		if !ok || agent == nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "forward agent not found",
			})
			continue
		}

		// Check if already in this group
		currentGroupIDs := agent.GroupIDs()
		alreadyInGroup := false
		for _, gid := range currentGroupIDs {
			if gid == groupID {
				alreadyInGroup = true
				break
			}
		}
		if alreadyInGroup {
			// Already in this group, count as success
			succeededSIDs = append(succeededSIDs, agentSID)
			continue
		}

		// Add group ID to the list and collect for batch update
		newGroupIDs := append(currentGroupIDs, groupID)
		agentsToUpdate[agent.ID()] = newGroupIDs
		sidToID[agent.ID()] = agentSID
		succeededSIDs = append(succeededSIDs, agentSID)
	}

	// Batch update all agents that need changes
	if len(agentsToUpdate) > 0 {
		_, err := uc.agentRepo.BatchUpdateGroupIDs(ctx, agentsToUpdate)
		if err != nil {
			uc.logger.Errorw("failed to batch update forward agents", "error", err)
			// Mark all pending updates as failed
			for _, agentSID := range succeededSIDs {
				if agent, ok := agentMap[agentSID]; ok && agentsToUpdate[agent.ID()] != nil {
					result.Failed = append(result.Failed, dto.BatchOperationErr{
						ID:     agentSID,
						Reason: "failed to update forward agent",
					})
				} else {
					// Agent was already in group, still succeeded
					result.Succeeded = append(result.Succeeded, agentSID)
				}
			}
		} else {
			result.Succeeded = append(result.Succeeded, succeededSIDs...)
		}
	} else {
		// All agents were already in the group
		result.Succeeded = append(result.Succeeded, succeededSIDs...)
	}

	uc.logger.Infow("added forward agents to resource group",
		"group_id", groupID,
		"group_sid", group.SID(),
		"succeeded_count", len(result.Succeeded),
		"failed_count", len(result.Failed))

	return result, nil
}

// RemoveAgents removes forward agents from a resource group by internal ID
func (uc *ManageResourceGroupForwardAgentsUseCase) RemoveAgents(ctx context.Context, groupID uint, agentSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeRemoveAgents(ctx, group, agentSIDs)
}

// RemoveAgentsBySID removes forward agents from a resource group by Stripe-style SID
func (uc *ManageResourceGroupForwardAgentsUseCase) RemoveAgentsBySID(ctx context.Context, groupSID string, agentSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "group_sid", groupSID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeRemoveAgents(ctx, group, agentSIDs)
}

// executeRemoveAgents performs the actual remove agents logic
func (uc *ManageResourceGroupForwardAgentsUseCase) executeRemoveAgents(ctx context.Context, group *resource.ResourceGroup, agentSIDs []string) (*dto.BatchOperationResult, error) {
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	result := &dto.BatchOperationResult{
		Succeeded: make([]string, 0),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	// Validate SID formats and collect valid SIDs for batch fetch
	validSIDs := make([]string, 0, len(agentSIDs))
	for _, agentSID := range agentSIDs {
		if err := id.ValidatePrefix(agentSID, id.PrefixForwardAgent); err != nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "invalid agent ID format",
			})
			continue
		}
		validSIDs = append(validSIDs, agentSID)
	}

	// Batch fetch all valid agents
	agentMap := make(map[string]*forward.ForwardAgent)
	if len(validSIDs) > 0 {
		agents, err := uc.agentRepo.GetBySIDs(ctx, validSIDs)
		if err != nil {
			uc.logger.Errorw("failed to batch get forward agents", "error", err)
			// Fall back to marking all as failed
			for _, sid := range validSIDs {
				result.Failed = append(result.Failed, dto.BatchOperationErr{
					ID:     sid,
					Reason: "failed to get forward agent",
				})
			}
			return result, nil
		}
		for _, agent := range agents {
			agentMap[agent.SID()] = agent
		}
	}

	// Process each valid SID and collect agents that need to be updated
	groupID := group.ID()
	agentsToUpdate := make(map[uint][]uint) // agentID -> new groupIDs
	sidToID := make(map[uint]string)        // For mapping back to SIDs
	succeededSIDs := make([]string, 0)

	for _, agentSID := range validSIDs {
		agent, ok := agentMap[agentSID]
		if !ok || agent == nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "forward agent not found",
			})
			continue
		}

		// Check if the agent belongs to this group
		currentGroupIDs := agent.GroupIDs()
		foundIndex := -1
		for i, gid := range currentGroupIDs {
			if gid == groupID {
				foundIndex = i
				break
			}
		}
		if foundIndex == -1 {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "forward agent does not belong to this group",
			})
			continue
		}

		// Remove group ID from the list and collect for batch update
		newGroupIDs := make([]uint, 0, len(currentGroupIDs)-1)
		for i, gid := range currentGroupIDs {
			if i != foundIndex {
				newGroupIDs = append(newGroupIDs, gid)
			}
		}
		agentsToUpdate[agent.ID()] = newGroupIDs
		sidToID[agent.ID()] = agentSID
		succeededSIDs = append(succeededSIDs, agentSID)
	}

	// Batch update all agents that need changes
	if len(agentsToUpdate) > 0 {
		_, err := uc.agentRepo.BatchUpdateGroupIDs(ctx, agentsToUpdate)
		if err != nil {
			uc.logger.Errorw("failed to batch update forward agents", "error", err)
			// Mark all pending updates as failed
			for _, agentSID := range succeededSIDs {
				result.Failed = append(result.Failed, dto.BatchOperationErr{
					ID:     agentSID,
					Reason: "failed to update forward agent",
				})
			}
		} else {
			result.Succeeded = append(result.Succeeded, succeededSIDs...)
		}
	}

	uc.logger.Infow("removed forward agents from resource group",
		"group_id", groupID,
		"group_sid", group.SID(),
		"succeeded_count", len(result.Succeeded),
		"failed_count", len(result.Failed))

	return result, nil
}

// ListAgents lists all forward agents in a resource group with pagination by internal ID
func (uc *ManageResourceGroupForwardAgentsUseCase) ListAgents(ctx context.Context, groupID uint, page, pageSize int) (*dto.ListGroupForwardAgentsResponse, error) {
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeListAgents(ctx, group, page, pageSize)
}

// ListAgentsBySID lists all forward agents in a resource group with pagination by Stripe-style SID
func (uc *ManageResourceGroupForwardAgentsUseCase) ListAgentsBySID(ctx context.Context, groupSID string, page, pageSize int) (*dto.ListGroupForwardAgentsResponse, error) {
	group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "group_sid", groupSID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeListAgents(ctx, group, page, pageSize)
}

// executeListAgents performs the actual list agents logic
func (uc *ManageResourceGroupForwardAgentsUseCase) executeListAgents(ctx context.Context, group *resource.ResourceGroup, page, pageSize int) (*dto.ListGroupForwardAgentsResponse, error) {
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	// List agents with group filter
	filter := forward.AgentListFilter{
		Page:     page,
		PageSize: pageSize,
		GroupIDs: []uint{group.ID()},
	}

	agents, total, err := uc.agentRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list forward agents", "error", err, "group_id", group.ID())
		return nil, fmt.Errorf("failed to list forward agents: %w", err)
	}

	// Convert to response DTOs
	items := make([]dto.ForwardAgentSummaryResponse, 0, len(agents))
	groupSID := group.SID()
	for _, agent := range agents {
		items = append(items, dto.ForwardAgentSummaryResponse{
			ID:        agent.SID(),
			Name:      agent.Name(),
			Status:    string(agent.Status()),
			GroupSID:  &groupSID,
			CreatedAt: agent.CreatedAt(),
		})
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &dto.ListGroupForwardAgentsResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}
