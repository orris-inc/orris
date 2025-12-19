package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ManageResourceGroupForwardAgentsUseCase handles adding/removing forward agents from resource groups
type ManageResourceGroupForwardAgentsUseCase struct {
	resourceGroupRepo resource.Repository
	agentRepo         forward.AgentRepository
	logger            logger.Interface
}

// NewManageResourceGroupForwardAgentsUseCase creates a new ManageResourceGroupForwardAgentsUseCase
func NewManageResourceGroupForwardAgentsUseCase(
	resourceGroupRepo resource.Repository,
	agentRepo forward.AgentRepository,
	logger logger.Interface,
) *ManageResourceGroupForwardAgentsUseCase {
	return &ManageResourceGroupForwardAgentsUseCase{
		resourceGroupRepo: resourceGroupRepo,
		agentRepo:         agentRepo,
		logger:            logger,
	}
}

// AddAgents adds forward agents to a resource group
func (uc *ManageResourceGroupForwardAgentsUseCase) AddAgents(ctx context.Context, groupID uint, agentSIDs []string) (*dto.BatchOperationResult, error) {
	// Verify the resource group exists
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	result := &dto.BatchOperationResult{
		Succeeded: make([]string, 0),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	for _, agentSID := range agentSIDs {
		// Parse the short ID from the prefixed SID (fa_xxx -> xxx)
		shortID, err := id.ParseForwardAgentID(agentSID)
		if err != nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "invalid agent ID format",
			})
			continue
		}

		// Get agent by short ID
		agent, err := uc.agentRepo.GetByShortID(ctx, shortID)
		if err != nil {
			uc.logger.Warnw("failed to get forward agent", "error", err, "agent_sid", agentSID)
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "failed to get forward agent",
			})
			continue
		}
		if agent == nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "forward agent not found",
			})
			continue
		}

		// Check if already in this group
		if agent.GroupID() != nil && *agent.GroupID() == groupID {
			// Already in this group, count as success
			result.Succeeded = append(result.Succeeded, agentSID)
			continue
		}

		// Set group ID and update
		gid := groupID
		agent.SetGroupID(&gid)
		if err := uc.agentRepo.Update(ctx, agent); err != nil {
			uc.logger.Errorw("failed to update forward agent", "error", err, "agent_sid", agentSID)
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "failed to update forward agent",
			})
			continue
		}

		result.Succeeded = append(result.Succeeded, agentSID)
	}

	uc.logger.Infow("added forward agents to resource group",
		"group_id", groupID,
		"succeeded_count", len(result.Succeeded),
		"failed_count", len(result.Failed))

	return result, nil
}

// RemoveAgents removes forward agents from a resource group
func (uc *ManageResourceGroupForwardAgentsUseCase) RemoveAgents(ctx context.Context, groupID uint, agentSIDs []string) (*dto.BatchOperationResult, error) {
	// Verify the resource group exists
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	result := &dto.BatchOperationResult{
		Succeeded: make([]string, 0),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	for _, agentSID := range agentSIDs {
		// Parse the short ID from the prefixed SID (fa_xxx -> xxx)
		shortID, err := id.ParseForwardAgentID(agentSID)
		if err != nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "invalid agent ID format",
			})
			continue
		}

		// Get agent by short ID
		agent, err := uc.agentRepo.GetByShortID(ctx, shortID)
		if err != nil {
			uc.logger.Warnw("failed to get forward agent", "error", err, "agent_sid", agentSID)
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "failed to get forward agent",
			})
			continue
		}
		if agent == nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "forward agent not found",
			})
			continue
		}

		// Check if the agent belongs to this group
		if agent.GroupID() == nil || *agent.GroupID() != groupID {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "forward agent does not belong to this group",
			})
			continue
		}

		// Remove group ID
		agent.SetGroupID(nil)
		if err := uc.agentRepo.Update(ctx, agent); err != nil {
			uc.logger.Errorw("failed to update forward agent", "error", err, "agent_sid", agentSID)
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     agentSID,
				Reason: "failed to update forward agent",
			})
			continue
		}

		result.Succeeded = append(result.Succeeded, agentSID)
	}

	uc.logger.Infow("removed forward agents from resource group",
		"group_id", groupID,
		"succeeded_count", len(result.Succeeded),
		"failed_count", len(result.Failed))

	return result, nil
}

// ListAgents lists all forward agents in a resource group with pagination
func (uc *ManageResourceGroupForwardAgentsUseCase) ListAgents(ctx context.Context, groupID uint, page, pageSize int) (*dto.ListGroupForwardAgentsResponse, error) {
	// Verify the resource group exists
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	// List agents with group filter
	filter := forward.AgentListFilter{
		Page:     page,
		PageSize: pageSize,
		GroupIDs: []uint{groupID},
	}

	agents, total, err := uc.agentRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list forward agents", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to list forward agents: %w", err)
	}

	// Convert to response DTOs
	items := make([]dto.ForwardAgentSummaryResponse, 0, len(agents))
	groupSID := group.SID()
	for _, agent := range agents {
		items = append(items, dto.ForwardAgentSummaryResponse{
			ID:        id.FormatForwardAgentID(agent.ShortID()),
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
