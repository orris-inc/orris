package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	commondto "github.com/orris-inc/orris/internal/application/common/dto"
	nodeUsecases "github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/adapters/systemstatus"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SubscriptionUsageRecorderAdapter adapts to record subscription-based usage
// This adapter records usage by subscription_id for proper usage tracking
//
// Architecture: Agent → Adapter → MySQL
type SubscriptionUsageRecorderAdapter struct {
	subscriptionUsageRepo subscription.SubscriptionUsageRepository
	logger                logger.Interface
}

// NewSubscriptionUsageRecorderAdapter creates a new subscription usage recorder adapter
// Note: Directly writes to database for simplicity and reliability
func NewSubscriptionUsageRecorderAdapter(
	subscriptionUsageRepo subscription.SubscriptionUsageRepository,
	logger logger.Interface,
) nodeUsecases.SubscriptionUsageRecorder {
	return &SubscriptionUsageRecorderAdapter{
		subscriptionUsageRepo: subscriptionUsageRepo,
		logger:                logger,
	}
}

// RecordSubscriptionUsage records subscription usage data directly to database
func (a *SubscriptionUsageRecorderAdapter) RecordSubscriptionUsage(ctx context.Context, nodeID uint, subscriptionID uint, upload, download int64) error {
	// Validate subscription ID
	if subscriptionID == 0 {
		a.logger.Warnw("invalid subscription ID", "subscription_id", subscriptionID)
		return nil // Skip invalid subscription IDs
	}

	// Skip zero usage
	if upload == 0 && download == 0 {
		return nil
	}

	// Create period for current hour aggregation
	// Truncate to hour in business timezone, then convert to UTC for storage
	period := biztime.TruncateToHourInBiz(biztime.NowUTC())

	// Create domain entity
	usage, err := subscription.NewSubscriptionUsage(subscription.ResourceTypeNode.String(), nodeID, &subscriptionID, period)
	if err != nil {
		a.logger.Errorw("failed to create subscription usage entity",
			"error", err,
			"node_id", nodeID,
			"subscription_id", subscriptionID,
		)
		return err
	}

	// Accumulate usage
	if err := usage.Accumulate(uint64(upload), uint64(download)); err != nil {
		a.logger.Errorw("failed to accumulate usage",
			"error", err,
			"node_id", nodeID,
			"subscription_id", subscriptionID,
		)
		return err
	}

	// Record in repository
	if err := a.subscriptionUsageRepo.RecordUsage(ctx, usage); err != nil {
		a.logger.Errorw("failed to record subscription usage",
			"error", err,
			"subscription_id", subscriptionID,
			"node_id", nodeID,
		)
		return err
	}

	a.logger.Debugw("subscription usage recorded",
		"subscription_id", subscriptionID,
		"node_id", nodeID,
		"upload", upload,
		"download", download,
	)

	return nil
}

// BatchRecordSubscriptionUsage records multiple subscriptions' usage data directly to database
func (a *SubscriptionUsageRecorderAdapter) BatchRecordSubscriptionUsage(ctx context.Context, nodeID uint, items []nodeUsecases.SubscriptionUsageItem) error {
	if len(items) == 0 {
		return nil
	}

	// Use current hour as period for consistent aggregation
	// Truncate to hour in business timezone, then convert to UTC for storage
	period := biztime.TruncateToHourInBiz(biztime.NowUTC())

	// Process each item
	validCount := 0
	errorCount := 0

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

		// Create domain entity
		subID := item.SubscriptionID
		usage, err := subscription.NewSubscriptionUsage(subscription.ResourceTypeNode.String(), nodeID, &subID, period)
		if err != nil {
			a.logger.Errorw("failed to create subscription usage entity in batch",
				"error", err,
				"node_id", nodeID,
				"subscription_id", item.SubscriptionID,
			)
			errorCount++
			continue
		}

		// Accumulate usage
		if err := usage.Accumulate(uint64(item.Upload), uint64(item.Download)); err != nil {
			a.logger.Errorw("failed to accumulate usage in batch",
				"error", err,
				"node_id", nodeID,
				"subscription_id", item.SubscriptionID,
			)
			errorCount++
			continue
		}

		// Record in repository
		if err := a.subscriptionUsageRepo.RecordUsage(ctx, usage); err != nil {
			a.logger.Errorw("failed to record subscription usage in batch",
				"error", err,
				"node_id", nodeID,
				"subscription_id", item.SubscriptionID,
			)
			errorCount++
			continue
		}

		validCount++
	}

	a.logger.Infow("subscription usage batch processed",
		"node_id", nodeID,
		"success_count", validCount,
		"error_count", errorCount,
		"total_count", len(items),
	)

	// Return error if all items failed
	if validCount == 0 && errorCount > 0 {
		return fmt.Errorf("failed to record any usage in batch")
	}

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

	// Convert NodeStatusUpdate to SystemStatus for shared serialization
	sysStatus := &commondto.SystemStatus{
		CPUPercent:     status.CPUPercent,
		MemoryPercent:  status.MemoryPercent,
		MemoryUsed:     status.MemoryUsed,
		MemoryTotal:    status.MemoryTotal,
		MemoryAvail:    status.MemoryAvail,
		DiskPercent:    status.DiskPercent,
		DiskUsed:       status.DiskUsed,
		DiskTotal:      status.DiskTotal,
		UptimeSeconds:  status.UptimeSeconds,
		LoadAvg1:       status.LoadAvg1,
		LoadAvg5:       status.LoadAvg5,
		LoadAvg15:      status.LoadAvg15,
		NetworkRxBytes: status.NetworkRxBytes,
		NetworkTxBytes: status.NetworkTxBytes,
		NetworkRxRate:  status.NetworkRxRate,
		NetworkTxRate:  status.NetworkTxRate,
		TCPConnections: status.TCPConnections,
		UDPConnections: status.UDPConnections,
		PublicIPv4:     status.PublicIPv4,
		PublicIPv6:     status.PublicIPv6,
		AgentVersion:   status.AgentVersion,
		Platform:       status.Platform,
		Arch:           status.Arch,
	}

	// Use shared helper to convert to Redis fields
	data := systemstatus.ToRedisFields(sysStatus)

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
		"disk_percent", status.DiskPercent,
		"uptime_seconds", status.UptimeSeconds,
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
func parseNodeSystemStatus(values map[string]string) *nodeUsecases.NodeSystemStatus {
	// Use shared parser for common system status fields
	sysStatus := systemstatus.ParseSystemStatus(values)

	status := &nodeUsecases.NodeSystemStatus{
		CPUPercent:     sysStatus.CPUPercent,
		MemoryPercent:  sysStatus.MemoryPercent,
		MemoryUsed:     sysStatus.MemoryUsed,
		MemoryTotal:    sysStatus.MemoryTotal,
		MemoryAvail:    sysStatus.MemoryAvail,
		DiskPercent:    sysStatus.DiskPercent,
		DiskUsed:       sysStatus.DiskUsed,
		DiskTotal:      sysStatus.DiskTotal,
		UptimeSeconds:  sysStatus.UptimeSeconds,
		LoadAvg1:       sysStatus.LoadAvg1,
		LoadAvg5:       sysStatus.LoadAvg5,
		LoadAvg15:      sysStatus.LoadAvg15,
		NetworkRxBytes: sysStatus.NetworkRxBytes,
		NetworkTxBytes: sysStatus.NetworkTxBytes,
		NetworkRxRate:  sysStatus.NetworkRxRate,
		NetworkTxRate:  sysStatus.NetworkTxRate,
		TCPConnections: sysStatus.TCPConnections,
		UDPConnections: sysStatus.UDPConnections,
		PublicIPv4:     sysStatus.PublicIPv4,
		PublicIPv6:     sysStatus.PublicIPv6,
		AgentVersion:   sysStatus.AgentVersion,
		Platform:       sysStatus.Platform,
		Arch:           sysStatus.Arch,
	}

	// Parse UpdatedAt which is specific to NodeSystemStatus
	if v, ok := values[systemstatus.FieldUpdatedAt]; ok {
		fmt.Sscanf(v, "%d", &status.UpdatedAt)
	}

	return status
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
