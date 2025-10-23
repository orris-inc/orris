package node

import "context"

type NodeRepository interface {
	Create(ctx context.Context, node *Node) error
	GetByID(ctx context.Context, id uint) (*Node, error)
	GetByToken(ctx context.Context, tokenHash string) (*Node, error)
	Update(ctx context.Context, node *Node) error
	Delete(ctx context.Context, id uint) error

	List(ctx context.Context, filter NodeFilter) ([]*Node, int64, error)
	GetByGroupID(ctx context.Context, groupID uint) ([]*Node, error)
	GetByStatus(ctx context.Context, status string) ([]*Node, error)
	GetAvailableNodes(ctx context.Context) ([]*Node, error)

	ExistsByName(ctx context.Context, name string) (bool, error)
	ExistsByAddress(ctx context.Context, address string, port int) (bool, error)
}

type NodeFilter struct {
	Name     *string
	Status   *string
	Country  *string
	Tag      *string
	Page     int
	PageSize int
	SortBy   string
	SortDesc bool
}
