package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/node"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type BatchAddNodesToGroupCommand struct {
	GroupID uint
	NodeIDs []uint
}

type BatchAddNodesToGroupResult struct {
	GroupID      uint
	RequestCount int
	AddedCount   int
	SkippedCount int
	NodeCount    int
	Message      string
}

type BatchAddNodesToGroupUseCase struct {
	nodeRepo      node.NodeRepository
	nodeGroupRepo node.NodeGroupRepository
	logger        logger.Interface
}

func NewBatchAddNodesToGroupUseCase(
	nodeRepo node.NodeRepository,
	nodeGroupRepo node.NodeGroupRepository,
	logger logger.Interface,
) *BatchAddNodesToGroupUseCase {
	return &BatchAddNodesToGroupUseCase{
		nodeRepo:      nodeRepo,
		nodeGroupRepo: nodeGroupRepo,
		logger:        logger,
	}
}

func (uc *BatchAddNodesToGroupUseCase) Execute(ctx context.Context, cmd BatchAddNodesToGroupCommand) (*BatchAddNodesToGroupResult, error) {
	uc.logger.Infow("executing batch add nodes to group use case",
		"group_id", cmd.GroupID,
		"node_count", len(cmd.NodeIDs),
	)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid batch add nodes to group command", "error", err)
		return nil, err
	}

	// Get node group
	group, err := uc.nodeGroupRepo.GetByID(ctx, cmd.GroupID)
	if err != nil {
		uc.logger.Errorw("failed to get node group", "error", err, "group_id", cmd.GroupID)
		return nil, fmt.Errorf("failed to get node group: %w", err)
	}

	// Validate that all nodes exist
	validNodeIDs := make([]uint, 0, len(cmd.NodeIDs))
	for _, nodeID := range cmd.NodeIDs {
		if nodeID == 0 {
			continue
		}

		_, err := uc.nodeRepo.GetByID(ctx, nodeID)
		if err != nil {
			uc.logger.Warnw("node not found, skipping",
				"node_id", nodeID,
				"error", err,
			)
			continue
		}

		validNodeIDs = append(validNodeIDs, nodeID)
	}

	if len(validNodeIDs) == 0 {
		return nil, errors.NewValidationError("no valid nodes to add")
	}

	// Batch add nodes to group
	addedCount, err := group.AddNodes(validNodeIDs)
	if err != nil {
		uc.logger.Errorw("failed to add nodes to group", "error", err)
		return nil, fmt.Errorf("failed to add nodes to group: %w", err)
	}

	if addedCount == 0 {
		uc.logger.Infow("no new nodes added, all nodes already in group", "group_id", cmd.GroupID)
		return &BatchAddNodesToGroupResult{
			GroupID:      cmd.GroupID,
			RequestCount: len(cmd.NodeIDs),
			AddedCount:   0,
			SkippedCount: len(cmd.NodeIDs),
			NodeCount:    group.NodeCount(),
			Message:      "no new nodes added, all nodes already in group",
		}, nil
	}

	// Update node group in database
	if err := uc.nodeGroupRepo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update node group in database", "error", err)
		return nil, fmt.Errorf("failed to update node group: %w", err)
	}

	skippedCount := len(validNodeIDs) - addedCount
	uc.logger.Infow("nodes added to group successfully",
		"group_id", cmd.GroupID,
		"added_count", addedCount,
		"skipped_count", skippedCount,
		"total_node_count", group.NodeCount(),
	)

	return &BatchAddNodesToGroupResult{
		GroupID:      cmd.GroupID,
		RequestCount: len(cmd.NodeIDs),
		AddedCount:   addedCount,
		SkippedCount: skippedCount,
		NodeCount:    group.NodeCount(),
		Message:      fmt.Sprintf("successfully added %d nodes to group", addedCount),
	}, nil
}

func (uc *BatchAddNodesToGroupUseCase) validateCommand(cmd BatchAddNodesToGroupCommand) error {
	if cmd.GroupID == 0 {
		return errors.NewValidationError("group ID is required")
	}

	if len(cmd.NodeIDs) == 0 {
		return errors.NewValidationError("at least one node ID is required")
	}

	if len(cmd.NodeIDs) > 100 {
		return errors.NewValidationError("cannot add more than 100 nodes at once")
	}

	return nil
}
