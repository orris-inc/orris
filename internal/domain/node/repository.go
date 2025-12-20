package node

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/shared/query"
)

type NodeRepository interface {
	Create(ctx context.Context, node *Node) error
	GetByID(ctx context.Context, id uint) (*Node, error)
	GetBySID(ctx context.Context, sid string) (*Node, error)
	GetByIDs(ctx context.Context, ids []uint) ([]*Node, error)
	GetByToken(ctx context.Context, tokenHash string) (*Node, error)
	Update(ctx context.Context, node *Node) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filter NodeFilter) ([]*Node, int64, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
	ExistsByNameExcluding(ctx context.Context, name string, excludeID uint) (bool, error)
	ExistsByAddress(ctx context.Context, address string, port int) (bool, error)
	ExistsByAddressExcluding(ctx context.Context, address string, port int, excludeID uint) (bool, error)
	IncrementTraffic(ctx context.Context, nodeID uint, amount uint64) error
	// UpdateLastSeenAt updates the last_seen_at timestamp and public IPs for a node
	// Public IPs are optional - pass empty strings to skip updating them
	UpdateLastSeenAt(ctx context.Context, nodeID uint, publicIPv4, publicIPv6 string) error
	// GetLastSeenAt retrieves the last_seen_at timestamp for a node
	GetLastSeenAt(ctx context.Context, nodeID uint) (*time.Time, error)
}

type NodeFilter struct {
	query.BaseFilter
	Name     *string
	Status   *string
	Tag      *string
	GroupIDs []uint // Filter by resource group IDs
}
