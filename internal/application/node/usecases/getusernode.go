package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type GetUserNodeQuery struct {
	UserID  uint
	NodeSID string
}

type GetUserNodeExecutor interface {
	Execute(ctx context.Context, q GetUserNodeQuery) (*dto.UserNodeDTO, error)
}

type GetUserNodeUseCase struct {
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

func NewGetUserNodeUseCase(
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *GetUserNodeUseCase {
	return &GetUserNodeUseCase{
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

func (uc *GetUserNodeUseCase) Execute(ctx context.Context, q GetUserNodeQuery) (*dto.UserNodeDTO, error) {
	uc.logger.Debugw("executing get user node use case", "user_id", q.UserID, "node_sid", q.NodeSID)

	nodeEntity, err := uc.nodeRepo.GetBySID(ctx, q.NodeSID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if !nodeEntity.IsOwnedBy(q.UserID) {
		return nil, errors.NewForbiddenError("access denied to this node")
	}

	return dto.ToUserNodeDTO(nodeEntity), nil
}
