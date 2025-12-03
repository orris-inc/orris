package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type AddNodeToGroupCommand struct {
	GroupID uint
	NodeID  uint
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
		"node_id", cmd.NodeID,
	)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid add node to group command", "error", err)
		return nil, err
	}

	_, err := uc.nodeRepo.GetByID(ctx, cmd.NodeID)
	if err != nil {
		uc.logger.Errorw("failed to get node", "error", err, "node_id", cmd.NodeID)
		return nil, fmt.Errorf("node not found: %w", err)
	}

	group, err := uc.nodeGroupRepo.GetByID(ctx, cmd.GroupID)
	if err != nil {
		uc.logger.Errorw("failed to get node group", "error", err, "group_id", cmd.GroupID)
		return nil, fmt.Errorf("failed to get node group: %w", err)
	}

	if group.ContainsNode(cmd.NodeID) {
		return nil, errors.NewValidationError("node already exists in this group")
	}

	if err := group.AddNode(cmd.NodeID); err != nil {
		uc.logger.Errorw("failed to add node to group", "error", err)
		return nil, fmt.Errorf("failed to add node to group: %w", err)
	}

	if err := uc.nodeGroupRepo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update node group in database", "error", err)
		return nil, fmt.Errorf("failed to update node group: %w", err)
	}

	uc.logger.Infow("node added to group successfully",
		"group_id", cmd.GroupID,
		"node_id", cmd.NodeID,
		"node_count", group.NodeCount(),
	)

	return &AddNodeToGroupResult{
		GroupID:   cmd.GroupID,
		NodeID:    cmd.NodeID,
		NodeCount: group.NodeCount(),
		Message:   "node added to group successfully",
	}, nil
}

func (uc *AddNodeToGroupUseCase) validateCommand(cmd AddNodeToGroupCommand) error {
	if cmd.GroupID == 0 {
		return errors.NewValidationError("group ID is required")
	}

	if cmd.NodeID == 0 {
		return errors.NewValidationError("node ID is required")
	}

	return nil
}
