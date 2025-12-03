package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type BatchRemoveNodesFromGroupCommand struct {
	GroupID uint
	NodeIDs []uint
}

type BatchRemoveNodesFromGroupResult struct {
	GroupID      uint
	RequestCount int
	RemovedCount int
	NodeCount    int
	Message      string
}

type BatchRemoveNodesFromGroupUseCase struct {
	nodeGroupRepo node.NodeGroupRepository
	logger        logger.Interface
}

func NewBatchRemoveNodesFromGroupUseCase(
	nodeGroupRepo node.NodeGroupRepository,
	logger logger.Interface,
) *BatchRemoveNodesFromGroupUseCase {
	return &BatchRemoveNodesFromGroupUseCase{
		nodeGroupRepo: nodeGroupRepo,
		logger:        logger,
	}
}

func (uc *BatchRemoveNodesFromGroupUseCase) Execute(ctx context.Context, cmd BatchRemoveNodesFromGroupCommand) (*BatchRemoveNodesFromGroupResult, error) {
	uc.logger.Infow("executing batch remove nodes from group use case",
		"group_id", cmd.GroupID,
		"node_count", len(cmd.NodeIDs),
	)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid batch remove nodes from group command", "error", err)
		return nil, err
	}

	// Get node group
	group, err := uc.nodeGroupRepo.GetByID(ctx, cmd.GroupID)
	if err != nil {
		uc.logger.Errorw("failed to get node group", "error", err, "group_id", cmd.GroupID)
		return nil, fmt.Errorf("failed to get node group: %w", err)
	}

	// Batch remove nodes from group
	removedCount, err := group.RemoveNodes(cmd.NodeIDs)
	if err != nil {
		uc.logger.Errorw("failed to remove nodes from group", "error", err)
		return nil, fmt.Errorf("failed to remove nodes from group: %w", err)
	}

	if removedCount == 0 {
		uc.logger.Infow("no nodes removed, none of the nodes were in the group", "group_id", cmd.GroupID)
		return &BatchRemoveNodesFromGroupResult{
			GroupID:      cmd.GroupID,
			RequestCount: len(cmd.NodeIDs),
			RemovedCount: 0,
			NodeCount:    group.NodeCount(),
			Message:      "no nodes removed, none of the nodes were in the group",
		}, nil
	}

	// Update node group in database
	if err := uc.nodeGroupRepo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update node group in database", "error", err)
		return nil, fmt.Errorf("failed to update node group: %w", err)
	}

	uc.logger.Infow("nodes removed from group successfully",
		"group_id", cmd.GroupID,
		"removed_count", removedCount,
		"remaining_node_count", group.NodeCount(),
	)

	return &BatchRemoveNodesFromGroupResult{
		GroupID:      cmd.GroupID,
		RequestCount: len(cmd.NodeIDs),
		RemovedCount: removedCount,
		NodeCount:    group.NodeCount(),
		Message:      fmt.Sprintf("successfully removed %d nodes from group", removedCount),
	}, nil
}

func (uc *BatchRemoveNodesFromGroupUseCase) validateCommand(cmd BatchRemoveNodesFromGroupCommand) error {
	if cmd.GroupID == 0 {
		return errors.NewValidationError("group ID is required")
	}

	if len(cmd.NodeIDs) == 0 {
		return errors.NewValidationError("at least one node ID is required")
	}

	if len(cmd.NodeIDs) > 100 {
		return errors.NewValidationError("cannot remove more than 100 nodes at once")
	}

	return nil
}
