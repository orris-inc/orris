package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type AddNodeToGroupCommand struct {
	GroupID     uint
	NodeShortID string
}

type AddNodeToGroupResult struct {
	GroupID   uint
	NodeID    uint
	NodeCount int
	Message   string
}

type AddNodeToGroupUseCase struct {
	nodeRepo      node.NodeRepository
	nodeGroupRepo node.NodeGroupRepository
	logger        logger.Interface
}

func NewAddNodeToGroupUseCase(
	nodeRepo node.NodeRepository,
	nodeGroupRepo node.NodeGroupRepository,
	logger logger.Interface,
) *AddNodeToGroupUseCase {
	return &AddNodeToGroupUseCase{
		nodeRepo:      nodeRepo,
		nodeGroupRepo: nodeGroupRepo,
		logger:        logger,
	}
}

func (uc *AddNodeToGroupUseCase) Execute(ctx context.Context, cmd AddNodeToGroupCommand) (*AddNodeToGroupResult, error) {
	uc.logger.Infow("executing add node to group use case",
		"group_id", cmd.GroupID,
		"node_short_id", cmd.NodeShortID,
	)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid add node to group command", "error", err)
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

	if group.ContainsNode(node.ID()) {
		return nil, errors.NewValidationError("node already exists in this group")
	}

	if err := group.AddNode(node.ID()); err != nil {
		uc.logger.Errorw("failed to add node to group", "error", err)
		return nil, fmt.Errorf("failed to add node to group: %w", err)
	}

	if err := uc.nodeGroupRepo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update node group in database", "error", err)
		return nil, fmt.Errorf("failed to update node group: %w", err)
	}

	uc.logger.Infow("node added to group successfully",
		"group_id", cmd.GroupID,
		"node_short_id", cmd.NodeShortID,
		"node_id", node.ID(),
		"node_count", group.NodeCount(),
	)

	return &AddNodeToGroupResult{
		GroupID:   cmd.GroupID,
		NodeID:    node.ID(),
		NodeCount: group.NodeCount(),
		Message:   "node added to group successfully",
	}, nil
}

func (uc *AddNodeToGroupUseCase) validateCommand(cmd AddNodeToGroupCommand) error {
	if cmd.GroupID == 0 {
		return errors.NewValidationError("group ID is required")
	}

	if cmd.NodeShortID == "" {
		return errors.NewValidationError("node short ID is required")
	}

	return nil
}
