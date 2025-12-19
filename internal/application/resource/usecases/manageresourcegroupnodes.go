package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ManageResourceGroupNodesUseCase handles adding/removing nodes from resource groups
type ManageResourceGroupNodesUseCase struct {
	resourceGroupRepo resource.Repository
	nodeRepo          node.NodeRepository
	planRepo          subscription.PlanRepository
	logger            logger.Interface
}

// NewManageResourceGroupNodesUseCase creates a new ManageResourceGroupNodesUseCase
func NewManageResourceGroupNodesUseCase(
	resourceGroupRepo resource.Repository,
	nodeRepo node.NodeRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *ManageResourceGroupNodesUseCase {
	return &ManageResourceGroupNodesUseCase{
		resourceGroupRepo: resourceGroupRepo,
		nodeRepo:          nodeRepo,
		planRepo:          planRepo,
		logger:            logger,
	}
}

// AddNodes adds nodes to a resource group by its internal ID
func (uc *ManageResourceGroupNodesUseCase) AddNodes(ctx context.Context, groupID uint, nodeSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeAddNodes(ctx, group, nodeSIDs)
}

// AddNodesBySID adds nodes to a resource group by its Stripe-style SID
func (uc *ManageResourceGroupNodesUseCase) AddNodesBySID(ctx context.Context, groupSID string, nodeSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "group_sid", groupSID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeAddNodes(ctx, group, nodeSIDs)
}

// executeAddNodes performs the actual add nodes logic
func (uc *ManageResourceGroupNodesUseCase) executeAddNodes(ctx context.Context, group *resource.ResourceGroup, nodeSIDs []string) (*dto.BatchOperationResult, error) {
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	groupID := group.ID()

	// Verify the plan type is node
	plan, err := uc.planRepo.GetByID(ctx, group.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get plan", "error", err, "plan_id", group.PlanID())
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found for resource group")
	}
	if !plan.PlanType().IsNode() {
		uc.logger.Warnw("attempted to add nodes to non-node plan resource group",
			"group_id", groupID,
			"plan_id", group.PlanID(),
			"plan_type", plan.PlanType().String())
		return nil, resource.ErrGroupPlanTypeMismatchNode
	}

	result := &dto.BatchOperationResult{
		Succeeded: make([]string, 0),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	for _, nodeSID := range nodeSIDs {
		// Get node by SID
		n, err := uc.nodeRepo.GetBySID(ctx, nodeSID)
		if err != nil {
			uc.logger.Warnw("failed to get node", "error", err, "node_sid", nodeSID)
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     nodeSID,
				Reason: "failed to get node",
			})
			continue
		}
		if n == nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     nodeSID,
				Reason: "node not found",
			})
			continue
		}

		// Check if already in this group
		if n.HasGroupID(groupID) {
			// Already in this group, count as success
			result.Succeeded = append(result.Succeeded, nodeSID)
			continue
		}

		// Add group ID and update
		n.AddGroupID(groupID)
		if err := uc.nodeRepo.Update(ctx, n); err != nil {
			uc.logger.Errorw("failed to update node", "error", err, "node_sid", nodeSID)
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     nodeSID,
				Reason: "failed to update node",
			})
			continue
		}

		result.Succeeded = append(result.Succeeded, nodeSID)
	}

	uc.logger.Infow("added nodes to resource group",
		"group_id", groupID,
		"group_sid", group.SID(),
		"succeeded_count", len(result.Succeeded),
		"failed_count", len(result.Failed))

	return result, nil
}

// RemoveNodes removes nodes from a resource group by its internal ID
func (uc *ManageResourceGroupNodesUseCase) RemoveNodes(ctx context.Context, groupID uint, nodeSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeRemoveNodes(ctx, group, nodeSIDs)
}

// RemoveNodesBySID removes nodes from a resource group by its Stripe-style SID
func (uc *ManageResourceGroupNodesUseCase) RemoveNodesBySID(ctx context.Context, groupSID string, nodeSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "group_sid", groupSID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeRemoveNodes(ctx, group, nodeSIDs)
}

// executeRemoveNodes performs the actual remove nodes logic
func (uc *ManageResourceGroupNodesUseCase) executeRemoveNodes(ctx context.Context, group *resource.ResourceGroup, nodeSIDs []string) (*dto.BatchOperationResult, error) {
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	groupID := group.ID()

	result := &dto.BatchOperationResult{
		Succeeded: make([]string, 0),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	for _, nodeSID := range nodeSIDs {
		// Get node by SID
		n, err := uc.nodeRepo.GetBySID(ctx, nodeSID)
		if err != nil {
			uc.logger.Warnw("failed to get node", "error", err, "node_sid", nodeSID)
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     nodeSID,
				Reason: "failed to get node",
			})
			continue
		}
		if n == nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     nodeSID,
				Reason: "node not found",
			})
			continue
		}

		// Check if the node belongs to this group
		if !n.HasGroupID(groupID) {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     nodeSID,
				Reason: "node does not belong to this group",
			})
			continue
		}

		// Remove group ID
		n.RemoveGroupID(groupID)
		if err := uc.nodeRepo.Update(ctx, n); err != nil {
			uc.logger.Errorw("failed to update node", "error", err, "node_sid", nodeSID)
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     nodeSID,
				Reason: "failed to update node",
			})
			continue
		}

		result.Succeeded = append(result.Succeeded, nodeSID)
	}

	uc.logger.Infow("removed nodes from resource group",
		"group_id", groupID,
		"group_sid", group.SID(),
		"succeeded_count", len(result.Succeeded),
		"failed_count", len(result.Failed))

	return result, nil
}

// ListNodes lists all nodes in a resource group with pagination by its internal ID
func (uc *ManageResourceGroupNodesUseCase) ListNodes(ctx context.Context, groupID uint, page, pageSize int) (*dto.ListGroupNodesResponse, error) {
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeListNodes(ctx, group, page, pageSize)
}

// ListNodesBySID lists all nodes in a resource group with pagination by its Stripe-style SID
func (uc *ManageResourceGroupNodesUseCase) ListNodesBySID(ctx context.Context, groupSID string, page, pageSize int) (*dto.ListGroupNodesResponse, error) {
	group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "group_sid", groupSID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeListNodes(ctx, group, page, pageSize)
}

// executeListNodes performs the actual list nodes logic
func (uc *ManageResourceGroupNodesUseCase) executeListNodes(ctx context.Context, group *resource.ResourceGroup, page, pageSize int) (*dto.ListGroupNodesResponse, error) {
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	groupID := group.ID()

	// List nodes with group filter
	filter := node.NodeFilter{
		GroupIDs: []uint{groupID},
	}
	filter.Page = page
	filter.PageSize = pageSize

	nodes, total, err := uc.nodeRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list nodes", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Convert to response DTOs
	// Build a map of groupID -> groupSID for all groups the nodes belong to
	groupIDSet := make(map[uint]bool)
	for _, n := range nodes {
		for _, gid := range n.GroupIDs() {
			groupIDSet[gid] = true
		}
	}
	groupIDToSID := make(map[uint]string)
	groupIDToSID[groupID] = group.SID() // Current group is already loaded
	for gid := range groupIDSet {
		if gid != groupID {
			g, err := uc.resourceGroupRepo.GetByID(ctx, gid)
			if err == nil && g != nil {
				groupIDToSID[gid] = g.SID()
			}
		}
	}

	items := make([]dto.NodeSummaryResponse, 0, len(nodes))
	for _, n := range nodes {
		groupSIDs := make([]string, 0, len(n.GroupIDs()))
		for _, gid := range n.GroupIDs() {
			if sid, ok := groupIDToSID[gid]; ok {
				groupSIDs = append(groupSIDs, sid)
			}
		}
		items = append(items, dto.NodeSummaryResponse{
			ID:        n.SID(),
			Name:      n.Name(),
			Status:    n.Status().String(),
			GroupSIDs: groupSIDs,
			CreatedAt: n.CreatedAt(),
		})
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &dto.ListGroupNodesResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}
