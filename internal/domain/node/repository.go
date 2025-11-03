package node

import (
	"context"

	"orris/internal/shared/query"
)

type NodeRepository interface {
	Create(ctx context.Context, node *Node) error
	GetByID(ctx context.Context, id uint) (*Node, error)
	GetByToken(ctx context.Context, tokenHash string) (*Node, error)
	Update(ctx context.Context, node *Node) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filter NodeFilter) ([]*Node, int64, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
	ExistsByAddress(ctx context.Context, address string, port int) (bool, error)
}

type NodeFilter struct {
	query.BaseFilter
	Name    *string
	Status  *string
	Country *string
	Tag     *string
}
