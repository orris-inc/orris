package usecases

import "context"

// NodeOnlineSubscriptionCounter queries online subscription counts per node from Redis.
// Defined in node application layer to keep the interface close to its consumers.
type NodeOnlineSubscriptionCounter interface {
	GetNodeOnlineSubscriptionCount(ctx context.Context, nodeID uint) (int, error)
	GetNodeOnlineSubscriptionCounts(ctx context.Context, nodeIDs []uint) (map[uint]int, error)
}
