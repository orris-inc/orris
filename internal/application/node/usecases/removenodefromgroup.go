package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type RemoveNodeFromGroupCommand struct {
	GroupID uint
	NodeID  uint
}

type RemoveNodeFromGroupResult struct {
	GroupID   uint
	NodeID    uint
	NodeCount int
	Message   string
}

type RemoveNodeFromGroupUseCase struct {
	nodeGroupRepo node.NodeGroupRepository
	logger        logger.Interface
}

func NewRemoveNodeFromGroupUseCase(
	nodeGroupRepo node.NodeGroupRepository,
	logger logger.Interface,
) *RemoveNodeFromGroupUseCase {
	return &RemoveNodeFromGroupUseCase{
		nodeGroupRepo: nodeGroupRepo,
		logger:        logger,
	}
}

func (uc *RemoveNodeFromGroupUseCase) Execute(ctx context.Context, cmd RemoveNodeFromGroupCommand) (*RemoveNodeFromGroupResult, error) {
	uc.logger.Infow("executing remove node from group use case",
		"group_id", cmd.GroupID,
		"node_id", cmd.NodeID,
	)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid remove node from group command", "error", err)
		return nil, err
	}

	group, err := uc.nodeGroupRepo.GetByID(ctx, cmd.GroupID)
	if err != nil {
		uc.logger.Errorw("failed to get node group", "error", err, "group_id", cmd.GroupID)
		return nil, fmt.Errorf("failed to get node group: %w", err)
	}

	if !group.ContainsNode(cmd.NodeID) {
		return nil, errors.NewValidationError("node does not exist in this group")
	}

	if err := group.RemoveNode(cmd.NodeID); err != nil {
		uc.logger.Errorw("failed to remove node from group", "error", err)
		return nil, fmt.Errorf("failed to remove node from group: %w", err)
	}

	if err := uc.nodeGroupRepo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update node group in database", "error", err)
		return nil, fmt.Errorf("failed to update node group: %w", err)
	}

	uc.logger.Infow("node removed from group successfully",
		"group_id", cmd.GroupID,
		"node_id", cmd.NodeID,
		"node_count", group.NodeCount(),
	)

	return &RemoveNodeFromGroupResult{
		GroupID:   cmd.GroupID,
		NodeID:    cmd.NodeID,
		NodeCount: group.NodeCount(),
		Message:   "node removed from group successfully",
	}, nil
}

func (uc *RemoveNodeFromGroupUseCase) validateCommand(cmd RemoveNodeFromGroupCommand) error {
	if cmd.GroupID == 0 {
		return errors.NewValidationError("group ID is required")
	}

	if cmd.NodeID == 0 {
		return errors.NewValidationError("node ID is required")
	}

	return nil
}
