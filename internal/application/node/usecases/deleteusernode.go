package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type DeleteUserNodeCommand struct {
	UserID  uint
	NodeSID string
}

type DeleteUserNodeExecutor interface {
	Execute(ctx context.Context, cmd DeleteUserNodeCommand) error
}

type DeleteUserNodeUseCase struct {
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

func NewDeleteUserNodeUseCase(
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *DeleteUserNodeUseCase {
	return &DeleteUserNodeUseCase{
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

func (uc *DeleteUserNodeUseCase) Execute(ctx context.Context, cmd DeleteUserNodeCommand) error {
	uc.logger.Infow("executing delete user node use case", "user_id", cmd.UserID, "node_sid", cmd.NodeSID)

	nodeEntity, err := uc.nodeRepo.GetBySID(ctx, cmd.NodeSID)
	if err != nil {
		return err
	}

	// Verify ownership
	if !nodeEntity.IsOwnedBy(cmd.UserID) {
		return errors.NewForbiddenError("access denied to this node")
	}

	// Delete the node
	if err := uc.nodeRepo.Delete(ctx, nodeEntity.ID()); err != nil {
		uc.logger.Errorw("failed to delete user node", "node_sid", cmd.NodeSID, "error", err)
		return err
	}

	uc.logger.Infow("user node deleted successfully", "user_id", cmd.UserID, "node_sid", cmd.NodeSID)
	return nil
}
