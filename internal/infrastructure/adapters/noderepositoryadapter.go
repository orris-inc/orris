package adapters

import (
	"context"

	"orris/internal/application/node/usecases"
	"orris/internal/shared/logger"
)

type NodeRepositoryAdapter struct {
	logger logger.Interface
}

func NewNodeRepositoryAdapter(logger logger.Interface) *NodeRepositoryAdapter {
	return &NodeRepositoryAdapter{
		logger: logger,
	}
}

func (r *NodeRepositoryAdapter) GetBySubscriptionToken(ctx context.Context, token string) ([]*usecases.Node, error) {
	r.logger.Warnw("GetBySubscriptionToken not implemented, returning empty list")
	return []*usecases.Node{}, nil
}

func (r *NodeRepositoryAdapter) GetByTokenHash(ctx context.Context, tokenHash string) (usecases.NodeData, error) {
	r.logger.Warnw("GetByTokenHash not implemented, returning empty data")
	return usecases.NodeData{}, nil
}
