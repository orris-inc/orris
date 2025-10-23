package usecases

import "context"

type NodeRepository interface {
	GetBySubscriptionToken(ctx context.Context, token string) ([]*Node, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (NodeData, error)
}

type NodeData struct {
	ID        uint
	Name      string
	TokenHash string
	Status    string
}
