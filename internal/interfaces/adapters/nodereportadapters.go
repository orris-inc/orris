package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	nodeUsecases "github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/interfaces/adapters/systemstatus"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SubscriptionTrafficBufferWriter defines the interface for writing subscription traffic to buffer.
// Defined here to avoid cross-layer dependency on handlers package.
type SubscriptionTrafficBufferWriter interface {
	AddTraffic(nodeID, subscriptionID uint, upload, download int64)
}

// SubscriptionUsageRecorderAdapter adapts to record subscription-based usage
// This adapter records usage by subscription_id for proper usage tracking
//
// Architecture: Agent → Adapter → SubscriptionTrafficBuffer → Redis → MySQL
type SubscriptionUsageRecorderAdapter struct {
	trafficBuffer SubscriptionTrafficBufferWriter
	logger        logger.Interface
}

// NewSubscriptionUsageRecorderAdapter creates a new subscription usage recorder adapter
// Note: Writes to SubscriptionTrafficBuffer for unified traffic aggregation
func NewSubscriptionUsageRecorderAdapter(
	trafficBuffer SubscriptionTrafficBufferWriter,
	logger logger.Interface,
) nodeUsecases.SubscriptionUsageRecorder {
	return &SubscriptionUsageRecorderAdapter{
		trafficBuffer: trafficBuffer,
		logger:        logger,
	}
}

// RecordSubscriptionUsage records subscription usage data to buffer
func (a *SubscriptionUsageRecorderAdapter) RecordSubscriptionUsage(_ context.Context, nodeID uint, subscriptionID uint, upload, download int64) error {
	// Validate subscription ID
	if subscriptionID == 0 {
		a.logger.Warnw("invalid subscription ID", "subscription_id", subscriptionID)
		return nil // Skip invalid subscription IDs
	}

	// Skip zero usage
	if upload == 0 && download == 0 {
		return nil
	}

	// Write to traffic buffer for unified aggregation
	a.trafficBuffer.AddTraffic(nodeID, subscriptionID, upload, download)

	a.logger.Debugw("subscription usage added to buffer",
		"subscription_id", subscriptionID,
		"node_id", nodeID,
		"upload", upload,
		"download", download,
	)

	return nil
}

// BatchRecordSubscriptionUsage records multiple subscriptions' usage data to buffer
func (a *SubscriptionUsageRecorderAdapter) BatchRecordSubscriptionUsage(_ context.Context, nodeID uint, items []nodeUsecases.SubscriptionUsageItem) error {
	if len(items) == 0 {
		return nil
	}

	// Process each item
	validCount := 0

	for _, item := range items {
		// Skip invalid subscription IDs
		if item.SubscriptionID == 0 {
			a.logger.Warnw("skipping invalid subscription ID in batch",
				"subscription_id", item.SubscriptionID,
				"node_id", nodeID,
			)
			continue
		}

		// Skip zero usage
		if item.Upload == 0 && item.Download == 0 {
			continue
		}

		// Write to traffic buffer for unified aggregation
		a.trafficBuffer.AddTraffic(nodeID, item.SubscriptionID, item.Upload, item.Download)
		validCount++
	}

	a.logger.Debugw("subscription usage batch added to buffer",
		"node_id", nodeID,
		"valid_count", validCount,
		"total_count", len(items),
	)

	return nil
}

// OnlineSubscriptionTrackerAdapter adapts to OnlineSubscriptionTracker interface
type OnlineSubscriptionTrackerAdapter struct {
	logger logger.Interface
}

// NewOnlineSubscriptionTrackerAdapter creates a new online subscription tracker adapter
func NewOnlineSubscriptionTrackerAdapter(
	logger logger.Interface,
) nodeUsecases.OnlineSubscriptionTracker {
	return &OnlineSubscriptionTrackerAdapter{
		logger: logger,
	}
}

// UpdateOnlineSubscriptions updates online subscriptions tracking
func (a *OnlineSubscriptionTrackerAdapter) UpdateOnlineSubscriptions(ctx context.Context, nodeID uint, subscriptions []nodeUsecases.OnlineSubscriptionInfo) error {
	// For now, we just log the online subscriptions
	// A full implementation would need a cache (Redis) or database table to track online subscriptions
	a.logger.Infow("online subscriptions updated",
		"node_id", nodeID,
		"count", len(subscriptions),
	)

	// TODO: Implement Redis-based online subscription tracking if needed
	// This would involve:
	// 1. Store subscription IPs and timestamps in Redis with expiry
	// 2. Use sorted sets for efficient querying
	// 3. Clean up expired entries

	return nil
}

// NodeSystemStatusUpdaterAdapter adapts to NodeSystemStatusUpdater interface
type NodeSystemStatusUpdaterAdapter struct {
	redisClient *redis.Client
	logger      logger.Interface
}

// NewNodeSystemStatusUpdaterAdapter creates a new system status updater adapter
func NewNodeSystemStatusUpdaterAdapter(
	redisClient *redis.Client,
	logger logger.Interface,
) nodeUsecases.NodeSystemStatusUpdater {
	return &NodeSystemStatusUpdaterAdapter{
		redisClient: redisClient,
		logger:      logger,
	}
}

// UpdateSystemStatus updates node system status metrics in Redis
func (a *NodeSystemStatusUpdaterAdapter) UpdateSystemStatus(ctx context.Context, nodeID uint, status *nodeUsecases.NodeStatusUpdate) error {
	key := fmt.Sprintf("node:%d:status", nodeID)

	// NodeStatusUpdate embeds commondto.SystemStatus, use it directly
	data := systemstatus.ToRedisFields(&status.SystemStatus)

	// Add metadata
	data[systemstatus.FieldUpdatedAt] = fmt.Sprintf("%d", biztime.NowUTC().Unix())

	pipe := a.redisClient.Pipeline()
	pipe.HSet(ctx, key, data)
	pipe.Expire(ctx, key, 5*time.Minute)

	_, err := pipe.Exec(ctx)
	if err != nil {
		a.logger.Errorw("failed to store node status in redis",
			"error", err,
			"node_id", nodeID,
		)
		return fmt.Errorf("failed to store node status: %w", err)
	}

	a.logger.Debugw("node system status updated in redis",
		"node_id", nodeID,
		"cpu_percent", status.CPUPercent,
		"memory_percent", status.MemoryPercent,
	)

	return nil
}

// NodeSystemStatusQuerierAdapter queries node system status from Redis
type NodeSystemStatusQuerierAdapter struct {
	redisClient *redis.Client
	logger      logger.Interface
}

// NewNodeSystemStatusQuerierAdapter creates a new system status querier adapter
func NewNodeSystemStatusQuerierAdapter(
	redisClient *redis.Client,
	logger logger.Interface,
) *NodeSystemStatusQuerierAdapter {
	return &NodeSystemStatusQuerierAdapter{
		redisClient: redisClient,
		logger:      logger,
	}
}

// GetNodeSystemStatus retrieves node system status from Redis
func (a *NodeSystemStatusQuerierAdapter) GetNodeSystemStatus(ctx context.Context, nodeID uint) (*nodeUsecases.NodeSystemStatus, error) {
	key := fmt.Sprintf("node:%d:status", nodeID)

	// Get all fields from Redis hash
	values, err := a.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		a.logger.Errorw("failed to get node status from redis",
			"error", err,
			"node_id", nodeID,
		)
		return nil, fmt.Errorf("failed to get node status: %w", err)
	}

	// If no data found, return nil (node status not available)
	if len(values) == 0 {
		return nil, nil
	}

	// Parse values into NodeSystemStatus
	status := parseNodeSystemStatus(values)

	return status, nil
}

// parseNodeSystemStatus parses Redis hash values into NodeSystemStatus
// using the shared systemstatus parser for common fields.
// NodeSystemStatus embeds commondto.SystemStatus, so direct assignment works.
func parseNodeSystemStatus(values map[string]string) *nodeUsecases.NodeSystemStatus {
	return &nodeUsecases.NodeSystemStatus{
		SystemStatus: systemstatus.ParseSystemStatus(values),
	}
}

// GetMultipleNodeSystemStatus retrieves system status for multiple nodes in batch
func (a *NodeSystemStatusQuerierAdapter) GetMultipleNodeSystemStatus(ctx context.Context, nodeIDs []uint) (map[uint]*nodeUsecases.NodeSystemStatus, error) {
	result := make(map[uint]*nodeUsecases.NodeSystemStatus)

	// Use pipeline for efficient batch querying
	pipe := a.redisClient.Pipeline()
	cmds := make(map[uint]*redis.MapStringStringCmd)

	for _, nodeID := range nodeIDs {
		key := fmt.Sprintf("node:%d:status", nodeID)
		cmds[nodeID] = pipe.HGetAll(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		a.logger.Errorw("failed to get multiple node statuses from redis",
			"error", err,
			"node_count", len(nodeIDs),
		)
		return result, fmt.Errorf("failed to get node statuses: %w", err)
	}

	// Parse results using shared helper function
	for nodeID, cmd := range cmds {
		values, err := cmd.Result()
		if err != nil || len(values) == 0 {
			continue
		}

		result[nodeID] = parseNodeSystemStatus(values)
	}

	return result, nil
}

// SubscriptionIDResolverAdapter implements SubscriptionIDResolver interface
type SubscriptionIDResolverAdapter struct {
	subscriptionRepo subscription.SubscriptionRepository
	logger           logger.Interface
}

// NewSubscriptionIDResolverAdapter creates a new subscription ID resolver adapter
func NewSubscriptionIDResolverAdapter(
	subscriptionRepo subscription.SubscriptionRepository,
	logger logger.Interface,
) nodeUsecases.SubscriptionIDResolver {
	return &SubscriptionIDResolverAdapter{
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

// GetIDBySID resolves a single subscription SID to internal ID
func (a *SubscriptionIDResolverAdapter) GetIDBySID(ctx context.Context, sid string) (uint, error) {
	sub, err := a.subscriptionRepo.GetBySID(ctx, sid)
	if err != nil {
		a.logger.Warnw("failed to resolve subscription SID",
			"sid", sid,
			"error", err,
		)
		return 0, err
	}
	return sub.ID(), nil
}

// GetIDsBySIDs resolves multiple subscription SIDs to internal IDs in batch
func (a *SubscriptionIDResolverAdapter) GetIDsBySIDs(ctx context.Context, sids []string) (map[string]uint, error) {
	result := make(map[string]uint)

	// Query each SID individually (could be optimized with batch query if needed)
	for _, sid := range sids {
		sub, err := a.subscriptionRepo.GetBySID(ctx, sid)
		if err != nil {
			a.logger.Warnw("failed to resolve subscription SID in batch",
				"sid", sid,
				"error", err,
			)
			continue // Skip failed lookups, don't fail the entire batch
		}
		result[sid] = sub.ID()
	}

	return result, nil
}
