package usecases

import "context"

// OnlineDeviceCounter queries online device counts from Redis.
// Defined in subscription application layer to avoid cross-domain dependency.
type OnlineDeviceCounter interface {
	GetOnlineDeviceCount(ctx context.Context, subscriptionID uint) (int, error)
	GetOnlineDeviceCounts(ctx context.Context, subscriptionIDs []uint) (map[uint]int, error)
}
