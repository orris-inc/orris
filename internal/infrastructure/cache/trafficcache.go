package cache

import (
	"context"
)

// TrafficCache manages traffic statistics in Redis
type TrafficCache interface {
	// IncrementTraffic atomically increments node traffic in Redis
	IncrementTraffic(ctx context.Context, nodeID uint, upload, download uint64) error

	// GetNodeTraffic returns total traffic (MySQL base + Redis delta)
	GetNodeTraffic(ctx context.Context, nodeID uint) (uint64, error)

	// FlushToDatabase synchronizes Redis traffic to MySQL in batch
	FlushToDatabase(ctx context.Context) error

	// GetAllPendingNodeIDs returns all node IDs with pending traffic in Redis
	GetAllPendingNodeIDs(ctx context.Context) ([]uint, error)
}
