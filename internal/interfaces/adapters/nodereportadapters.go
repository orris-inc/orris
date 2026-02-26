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

// OnlineSubscriptionTrackerAdapter adapts to OnlineSubscriptionTracker interface.
// Uses Redis sorted sets to track online device IPs per subscription.
type OnlineSubscriptionTrackerAdapter struct {
	redisClient *redis.Client
	logger      logger.Interface
}

// NewOnlineSubscriptionTrackerAdapter creates a new online subscription tracker adapter.
// Returns concrete type so it can satisfy both OnlineSubscriptionTracker and OnlineDeviceCounter interfaces.
func NewOnlineSubscriptionTrackerAdapter(
	redisClient *redis.Client,
	logger logger.Interface,
) *OnlineSubscriptionTrackerAdapter {
	return &OnlineSubscriptionTrackerAdapter{
		redisClient: redisClient,
		logger:      logger,
	}
}

const (
	// deviceOnlineKeyPrefix is the Redis key prefix for online device tracking.
	// Key format: device_online:{subscriptionID}
	deviceOnlineKeyPrefix = "device_online:"
	// deviceOnlineTTL is the expiry for each subscription's device set.
	deviceOnlineTTL = 5 * time.Minute
	// deviceOnlineStaleThreshold is the max age before a device entry is considered stale.
	deviceOnlineStaleThreshold = 5 * time.Minute

	// nodeOnlineSubsKeyPrefix is the Redis key prefix for per-node online subscription tracking.
	// Key format: node_online_subs:{nodeID}
	nodeOnlineSubsKeyPrefix = "node_online_subs:"
	// nodeOnlineSubsTTL is the expiry for each node's online subscription set.
	nodeOnlineSubsTTL = 5 * time.Minute
	// nodeOnlineSubsStaleThreshold is the max age before a subscription entry is considered stale.
	nodeOnlineSubsStaleThreshold = 5 * time.Minute
)

// UpdateOnlineSubscriptions updates online subscriptions tracking in Redis.
// For each subscription, it stores connected IPs in a sorted set with timestamps,
// removes stale entries, and sets a TTL on the key.
func (a *OnlineSubscriptionTrackerAdapter) UpdateOnlineSubscriptions(ctx context.Context, nodeID uint, subscriptions []nodeUsecases.OnlineSubscriptionInfo) error {
	if len(subscriptions) == 0 {
		return nil
	}

	// Group IPs by subscription ID
	subIPs := make(map[uint][]string)
	for _, s := range subscriptions {
		subIPs[s.SubscriptionID] = append(subIPs[s.SubscriptionID], s.IP)
	}

	now := float64(biztime.NowUTC().Unix())
	staleThreshold := now - deviceOnlineStaleThreshold.Seconds()

	pipe := a.redisClient.Pipeline()
	for subID, ips := range subIPs {
		key := fmt.Sprintf("%s%d", deviceOnlineKeyPrefix, subID)

		// Add each IP with current timestamp as score
		members := make([]redis.Z, 0, len(ips))
		for _, ip := range ips {
			members = append(members, redis.Z{Score: now, Member: ip})
		}
		pipe.ZAdd(ctx, key, members...)

		// Remove stale entries (score < staleThreshold)
		pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%f", staleThreshold))

		// Set TTL
		pipe.Expire(ctx, key, deviceOnlineTTL)
	}

	// Maintain per-node online subscription set
	nodeKey := fmt.Sprintf("%s%d", nodeOnlineSubsKeyPrefix, nodeID)
	nodeMembers := make([]redis.Z, 0, len(subIPs))
	for subID := range subIPs {
		nodeMembers = append(nodeMembers, redis.Z{Score: now, Member: fmt.Sprintf("%d", subID)})
	}
	nodeStaleThreshold := now - nodeOnlineSubsStaleThreshold.Seconds()
	pipe.ZAdd(ctx, nodeKey, nodeMembers...)
	pipe.ZRemRangeByScore(ctx, nodeKey, "-inf", fmt.Sprintf("%f", nodeStaleThreshold))
	pipe.Expire(ctx, nodeKey, nodeOnlineSubsTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		a.logger.Errorw("failed to update online subscriptions in redis",
			"error", err,
			"node_id", nodeID,
			"subscription_count", len(subIPs),
		)
		return fmt.Errorf("failed to update online subscriptions: %w", err)
	}

	a.logger.Debugw("online subscriptions updated in redis",
		"node_id", nodeID,
		"subscription_count", len(subIPs),
	)

	return nil
}

// GetOnlineDeviceCount returns the number of online devices for a single subscription.
// Removes stale entries before counting.
func (a *OnlineSubscriptionTrackerAdapter) GetOnlineDeviceCount(ctx context.Context, subscriptionID uint) (int, error) {
	key := fmt.Sprintf("%s%d", deviceOnlineKeyPrefix, subscriptionID)
	staleThreshold := fmt.Sprintf("%f", float64(biztime.NowUTC().Unix())-deviceOnlineStaleThreshold.Seconds())

	// Remove stale entries first
	a.redisClient.ZRemRangeByScore(ctx, key, "-inf", staleThreshold)

	count, err := a.redisClient.ZCard(ctx, key).Result()
	if err != nil {
		a.logger.Errorw("failed to get online device count",
			"error", err,
			"subscription_id", subscriptionID,
		)
		return 0, fmt.Errorf("failed to get online device count: %w", err)
	}

	return int(count), nil
}

// GetOnlineDeviceCounts returns online device counts for multiple subscriptions in batch.
// Uses Redis pipeline for efficiency. Removes stale entries before counting.
func (a *OnlineSubscriptionTrackerAdapter) GetOnlineDeviceCounts(ctx context.Context, subscriptionIDs []uint) (map[uint]int, error) {
	result := make(map[uint]int, len(subscriptionIDs))
	if len(subscriptionIDs) == 0 {
		return result, nil
	}

	staleThreshold := fmt.Sprintf("%f", float64(biztime.NowUTC().Unix())-deviceOnlineStaleThreshold.Seconds())

	// First pipeline: remove stale entries
	cleanPipe := a.redisClient.Pipeline()
	for _, subID := range subscriptionIDs {
		key := fmt.Sprintf("%s%d", deviceOnlineKeyPrefix, subID)
		cleanPipe.ZRemRangeByScore(ctx, key, "-inf", staleThreshold)
	}
	_, _ = cleanPipe.Exec(ctx) // Best-effort cleanup

	// Second pipeline: count entries
	countPipe := a.redisClient.Pipeline()
	cmds := make(map[uint]*redis.IntCmd, len(subscriptionIDs))
	for _, subID := range subscriptionIDs {
		key := fmt.Sprintf("%s%d", deviceOnlineKeyPrefix, subID)
		cmds[subID] = countPipe.ZCard(ctx, key)
	}

	_, err := countPipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		a.logger.Errorw("failed to batch get online device counts",
			"error", err,
			"subscription_count", len(subscriptionIDs),
		)
		return result, fmt.Errorf("failed to batch get online device counts: %w", err)
	}

	for subID, cmd := range cmds {
		count, err := cmd.Result()
		if err != nil {
			continue
		}
		result[subID] = int(count)
	}

	return result, nil
}

// GetNodeOnlineSubscriptionCount returns the number of online subscriptions for a single node.
// Removes stale entries before counting.
func (a *OnlineSubscriptionTrackerAdapter) GetNodeOnlineSubscriptionCount(ctx context.Context, nodeID uint) (int, error) {
	key := fmt.Sprintf("%s%d", nodeOnlineSubsKeyPrefix, nodeID)
	staleThreshold := fmt.Sprintf("%f", float64(biztime.NowUTC().Unix())-nodeOnlineSubsStaleThreshold.Seconds())

	// Remove stale entries first
	a.redisClient.ZRemRangeByScore(ctx, key, "-inf", staleThreshold)

	count, err := a.redisClient.ZCard(ctx, key).Result()
	if err != nil {
		a.logger.Errorw("failed to get node online subscription count",
			"error", err,
			"node_id", nodeID,
		)
		return 0, fmt.Errorf("failed to get node online subscription count: %w", err)
	}

	return int(count), nil
}

// GetNodeOnlineSubscriptionCounts returns online subscription counts for multiple nodes in batch.
// Uses Redis pipeline for efficiency. Removes stale entries before counting.
func (a *OnlineSubscriptionTrackerAdapter) GetNodeOnlineSubscriptionCounts(ctx context.Context, nodeIDs []uint) (map[uint]int, error) {
	result := make(map[uint]int, len(nodeIDs))
	if len(nodeIDs) == 0 {
		return result, nil
	}

	staleThreshold := fmt.Sprintf("%f", float64(biztime.NowUTC().Unix())-nodeOnlineSubsStaleThreshold.Seconds())

	// First pipeline: remove stale entries
	cleanPipe := a.redisClient.Pipeline()
	for _, nodeID := range nodeIDs {
		key := fmt.Sprintf("%s%d", nodeOnlineSubsKeyPrefix, nodeID)
		cleanPipe.ZRemRangeByScore(ctx, key, "-inf", staleThreshold)
	}
	_, _ = cleanPipe.Exec(ctx) // Best-effort cleanup

	// Second pipeline: count entries
	countPipe := a.redisClient.Pipeline()
	cmds := make(map[uint]*redis.IntCmd, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		key := fmt.Sprintf("%s%d", nodeOnlineSubsKeyPrefix, nodeID)
		cmds[nodeID] = countPipe.ZCard(ctx, key)
	}

	_, err := countPipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		a.logger.Errorw("failed to batch get node online subscription counts",
			"error", err,
			"node_count", len(nodeIDs),
		)
		return result, fmt.Errorf("failed to batch get node online subscription counts: %w", err)
	}

	for nodeID, cmd := range cmds {
		count, err := cmd.Result()
		if err != nil {
			continue
		}
		result[nodeID] = int(count)
	}

	return result, nil
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
