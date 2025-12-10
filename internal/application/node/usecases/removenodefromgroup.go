package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type RemoveNodeFromGroupCommand struct {
	GroupID     uint
	NodeShortID string
}

type RemoveNodeFromGroupResult struct {
	GroupID   uint
	NodeID    uint
	NodeCount int
	Message   string
}

type RemoveNodeFromGroupUseCase struct {
	nodeRepo      node.NodeRepository
	nodeGroupRepo node.NodeGroupRepository
	logger        logger.Interface
}

func NewRemoveNodeFromGroupUseCase(
	nodeRepo node.NodeRepository,
	nodeGroupRepo node.NodeGroupRepository,
	logger logger.Interface,
) *RemoveNodeFromGroupUseCase {
	return &RemoveNodeFromGroupUseCase{
		nodeRepo:      nodeRepo,
		nodeGroupRepo: nodeGroupRepo,
		logger:        logger,
	}
}

func (uc *RemoveNodeFromGroupUseCase) Execute(ctx context.Context, cmd RemoveNodeFromGroupCommand) (*RemoveNodeFromGroupResult, error) {
	uc.logger.Infow("executing remove node from group use case",
		"group_id", cmd.GroupID,
		"node_short_id", cmd.NodeShortID,
	)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid remove node from group command", "error", err)
		return nil, err
	}

	node, err := uc.nodeRepo.GetByShortID(ctx, cmd.NodeShortID)
	if err != nil {
		uc.logger.Errorw("failed to get node by short ID", "error", err, "node_short_id", cmd.NodeShortID)
		return nil, fmt.Errorf("node not found: %w", err)
	}

	group, err := uc.nodeGroupRepo.GetByID(ctx, cmd.GroupID)
	if err != nil {
		uc.logger.Errorw("failed to get node group", "error", err, "group_id", cmd.GroupID)
		return nil, fmt.Errorf("failed to get node group: %w", err)
	}

	if !group.ContainsNode(node.ID()) {
		return nil, errors.NewValidationError("node does not exist in this group")
	}

	if err := group.RemoveNode(node.ID()); err != nil {
		uc.logger.Errorw("failed to remove node from group", "error", err)
		return nil, fmt.Errorf("failed to remove node from group: %w", err)
	}

	if err := uc.nodeGroupRepo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update node group in database", "error", err)
		return nil, fmt.Errorf("failed to update node group: %w", err)
	}

	uc.logger.Infow("node removed from group successfully",
		"group_id", cmd.GroupID,
		"node_short_id", cmd.NodeShortID,
		"node_id", node.ID(),
		"node_count", group.NodeCount(),
	)

	return &RemoveNodeFromGroupResult{
		GroupID:   cmd.GroupID,
		NodeID:    node.ID(),
		NodeCount: group.NodeCount(),
		Message:   "node removed from group successfully",
	}, nil
}

func (uc *RemoveNodeFromGroupUseCase) validateCommand(cmd RemoveNodeFromGroupCommand) error {
	if cmd.GroupID == 0 {
		return errors.NewValidationError("group ID is required")
	}

	if cmd.NodeShortID == "" {
		return errors.NewValidationError("node short ID is required")
	}

	return nil
}
