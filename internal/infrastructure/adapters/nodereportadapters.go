package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	nodeUsecases "github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/subscription"
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

	// Store status in Redis hash with 5 minutes TTL
	data := map[string]interface{}{
		// System resources
		"cpu_percent":    fmt.Sprintf("%.2f", status.CPUPercent),
		"memory_percent": fmt.Sprintf("%.2f", status.MemoryPercent),
		"memory_used":    fmt.Sprintf("%d", status.MemoryUsed),
		"memory_total":   fmt.Sprintf("%d", status.MemoryTotal),
		"memory_avail":   fmt.Sprintf("%d", status.MemoryAvail),
		"disk_percent":   fmt.Sprintf("%.2f", status.DiskPercent),
		"disk_used":      fmt.Sprintf("%d", status.DiskUsed),
		"disk_total":     fmt.Sprintf("%d", status.DiskTotal),
		"uptime_seconds": fmt.Sprintf("%d", status.UptimeSeconds),

		// System load
		"load_avg_1":  fmt.Sprintf("%.2f", status.LoadAvg1),
		"load_avg_5":  fmt.Sprintf("%.2f", status.LoadAvg5),
		"load_avg_15": fmt.Sprintf("%.2f", status.LoadAvg15),

		// Network statistics
		"network_rx_bytes": fmt.Sprintf("%d", status.NetworkRxBytes),
		"network_tx_bytes": fmt.Sprintf("%d", status.NetworkTxBytes),
		"network_rx_rate":  fmt.Sprintf("%d", status.NetworkRxRate),
		"network_tx_rate":  fmt.Sprintf("%d", status.NetworkTxRate),

		// Connection statistics
		"tcp_connections": fmt.Sprintf("%d", status.TCPConnections),
		"udp_connections": fmt.Sprintf("%d", status.UDPConnections),

		// Agent info
		"agent_version": status.AgentVersion,
		"platform":      status.Platform,
		"arch":          status.Arch,

		// Metadata
		"updated_at": biztime.NowUTC().Unix(),
	}

	// Only store public IPs if provided
	if status.PublicIPv4 != "" {
		data["public_ipv4"] = status.PublicIPv4
	}
	if status.PublicIPv6 != "" {
		data["public_ipv6"] = status.PublicIPv6
	}

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
func parseNodeSystemStatus(values map[string]string) *nodeUsecases.NodeSystemStatus {
	status := &nodeUsecases.NodeSystemStatus{
		PublicIPv4:   values["public_ipv4"],
		PublicIPv6:   values["public_ipv6"],
		AgentVersion: values["agent_version"],
		Platform:     values["platform"],
		Arch:         values["arch"],
	}

	// Parse float64 fields
	if v, ok := values["cpu_percent"]; ok {
		fmt.Sscanf(v, "%f", &status.CPUPercent)
	}
	if v, ok := values["memory_percent"]; ok {
		fmt.Sscanf(v, "%f", &status.MemoryPercent)
	}
	if v, ok := values["disk_percent"]; ok {
		fmt.Sscanf(v, "%f", &status.DiskPercent)
	}
	if v, ok := values["load_avg_1"]; ok {
		fmt.Sscanf(v, "%f", &status.LoadAvg1)
	}
	if v, ok := values["load_avg_5"]; ok {
		fmt.Sscanf(v, "%f", &status.LoadAvg5)
	}
	if v, ok := values["load_avg_15"]; ok {
		fmt.Sscanf(v, "%f", &status.LoadAvg15)
	}

	// Parse uint64 fields
	if v, ok := values["memory_used"]; ok {
		fmt.Sscanf(v, "%d", &status.MemoryUsed)
	}
	if v, ok := values["memory_total"]; ok {
		fmt.Sscanf(v, "%d", &status.MemoryTotal)
	}
	if v, ok := values["memory_avail"]; ok {
		fmt.Sscanf(v, "%d", &status.MemoryAvail)
	}
	if v, ok := values["disk_used"]; ok {
		fmt.Sscanf(v, "%d", &status.DiskUsed)
	}
	if v, ok := values["disk_total"]; ok {
		fmt.Sscanf(v, "%d", &status.DiskTotal)
	}
	if v, ok := values["network_rx_bytes"]; ok {
		fmt.Sscanf(v, "%d", &status.NetworkRxBytes)
	}
	if v, ok := values["network_tx_bytes"]; ok {
		fmt.Sscanf(v, "%d", &status.NetworkTxBytes)
	}
	if v, ok := values["network_rx_rate"]; ok {
		fmt.Sscanf(v, "%d", &status.NetworkRxRate)
	}
	if v, ok := values["network_tx_rate"]; ok {
		fmt.Sscanf(v, "%d", &status.NetworkTxRate)
	}

	// Parse int64 fields
	if v, ok := values["uptime_seconds"]; ok {
		fmt.Sscanf(v, "%d", &status.UptimeSeconds)
	}
	if v, ok := values["updated_at"]; ok {
		fmt.Sscanf(v, "%d", &status.UpdatedAt)
	}

	// Parse int fields
	if v, ok := values["tcp_connections"]; ok {
		fmt.Sscanf(v, "%d", &status.TCPConnections)
	}
	if v, ok := values["udp_connections"]; ok {
		fmt.Sscanf(v, "%d", &status.UDPConnections)
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
