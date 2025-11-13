package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/node"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type DeleteNodeGroupCommand struct {
	GroupID uint
}

type DeleteNodeGroupResult struct {
	Success bool
	Message string
}

type DeleteNodeGroupUseCase struct {
	nodeGroupRepo node.NodeGroupRepository
	logger        logger.Interface
}

func NewDeleteNodeGroupUseCase(
	nodeGroupRepo node.NodeGroupRepository,
	logger logger.Interface,
) *DeleteNodeGroupUseCase {
	return &DeleteNodeGroupUseCase{
		nodeGroupRepo: nodeGroupRepo,
		logger:        logger,
	}
}

func (uc *DeleteNodeGroupUseCase) Execute(ctx context.Context, cmd DeleteNodeGroupCommand) (*DeleteNodeGroupResult, error) {
	uc.logger.Infow("executing delete node group use case", "group_id", cmd.GroupID)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid delete node group command", "error", err)
		return nil, err
	}

	group, err := uc.nodeGroupRepo.GetByID(ctx, cmd.GroupID)
	if err != nil {
		uc.logger.Errorw("failed to get node group", "error", err, "group_id", cmd.GroupID)
		return nil, fmt.Errorf("failed to get node group: %w", err)
	}

	// Log warning if deleting a group with active subscription plan associations
	// Note: Foreign key constraints have been removed for flexibility
	if len(group.SubscriptionPlanIDs()) > 0 {
		uc.logger.Warnw("deleting node group with active subscription plan associations",
			"group_id", cmd.GroupID,
			"group_name", group.Name(),
			"plan_count", len(group.SubscriptionPlanIDs()),
		)
	}

	// Soft delete the node group
	// Note: Associated records in node_group_nodes and node_group_plans will remain
	// but queries should filter by deleted_at
	if err := uc.nodeGroupRepo.Delete(ctx, cmd.GroupID); err != nil {
		uc.logger.Errorw("failed to delete node group from database", "error", err)
		return nil, fmt.Errorf("failed to delete node group: %w", err)
	}

	uc.logger.Infow("node group deleted successfully",
		"group_id", cmd.GroupID,
		"name", group.Name(),
	)

	return &DeleteNodeGroupResult{
		Success: true,
		Message: "node group deleted successfully",
	}, nil
}

func (uc *DeleteNodeGroupUseCase) validateCommand(cmd DeleteNodeGroupCommand) error {
	if cmd.GroupID == 0 {
		return errors.NewValidationError("group ID is required")
	}

	return nil
}
