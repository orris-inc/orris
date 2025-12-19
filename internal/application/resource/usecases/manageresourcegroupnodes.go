package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ManageResourceGroupNodesUseCase handles adding/removing nodes from resource groups
type ManageResourceGroupNodesUseCase struct {
	resourceGroupRepo resource.Repository
	nodeRepo          node.NodeRepository
	logger            logger.Interface
}

// NewManageResourceGroupNodesUseCase creates a new ManageResourceGroupNodesUseCase
func NewManageResourceGroupNodesUseCase(
	resourceGroupRepo resource.Repository,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *ManageResourceGroupNodesUseCase {
	return &ManageResourceGroupNodesUseCase{
		resourceGroupRepo: resourceGroupRepo,
		nodeRepo:          nodeRepo,
		logger:            logger,
	}
}

// AddNodes adds nodes to a resource group
func (uc *ManageResourceGroupNodesUseCase) AddNodes(ctx context.Context, groupID uint, nodeSIDs []string) (*dto.BatchOperationResult, error) {
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
		"succeeded_count", len(result.Succeeded),
		"failed_count", len(result.Failed))

	return result, nil
}

// RemoveNodes removes nodes from a resource group
func (uc *ManageResourceGroupNodesUseCase) RemoveNodes(ctx context.Context, groupID uint, nodeSIDs []string) (*dto.BatchOperationResult, error) {
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
		"succeeded_count", len(result.Succeeded),
		"failed_count", len(result.Failed))

	return result, nil
}

// ListNodes lists all nodes in a resource group with pagination
func (uc *ManageResourceGroupNodesUseCase) ListNodes(ctx context.Context, groupID uint, page, pageSize int) (*dto.ListGroupNodesResponse, error) {
	// Verify the resource group exists
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

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
