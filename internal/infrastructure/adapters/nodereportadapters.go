package adapters

import (
	"context"
	"fmt"
	"time"

	"orris/internal/application/node/usecases"
	"orris/internal/domain/node"
	"orris/internal/infrastructure/cache"
	"orris/internal/shared/logger"
)

// NodeTrafficRecorderAdapter adapts TrafficCache and NodeTrafficRepository to NodeTrafficRecorder interface
type NodeTrafficRecorderAdapter struct {
	trafficCache    cache.TrafficCache
	nodeTrafficRepo node.NodeTrafficRepository
	logger          logger.Interface
}

// NewNodeTrafficRecorderAdapter creates a new traffic recorder adapter
func NewNodeTrafficRecorderAdapter(
	trafficCache cache.TrafficCache,
	nodeTrafficRepo node.NodeTrafficRepository,
	logger logger.Interface,
) usecases.NodeTrafficRecorder {
	return &NodeTrafficRecorderAdapter{
		trafficCache:    trafficCache,
		nodeTrafficRepo: nodeTrafficRepo,
		logger:          logger,
	}
}

// RecordTraffic records node traffic data
func (a *NodeTrafficRecorderAdapter) RecordTraffic(ctx context.Context, nodeID uint, upload, download uint64) error {
	// 1. Record to Redis cache for high performance (priority)
	err := a.trafficCache.IncrementTraffic(ctx, nodeID, upload, download)
	if err != nil {
		a.logger.Warnw("failed to record traffic to redis, fallback to direct recording",
			"error", err,
			"node_id", nodeID,
		)
		// Don't fail here - Redis is cache layer, continue with historical recording
	}

	// 2. Record to NodeTraffic table for historical tracking
	period := time.Now().Truncate(time.Hour)
	traffic, err := node.NewNodeTraffic(nodeID, nil, nil, period)
	if err != nil {
		a.logger.Errorw("failed to create node traffic entity",
			"error", err,
			"node_id", nodeID,
		)
		return err
	}

	// Accumulate traffic
	if err := traffic.Accumulate(upload, download); err != nil {
		a.logger.Errorw("failed to accumulate traffic",
			"error", err,
			"node_id", nodeID,
		)
		return err
	}

	// Record in repository
	if err := a.nodeTrafficRepo.RecordTraffic(ctx, traffic); err != nil {
		return err
	}

	a.logger.Debugw("node traffic recorded via adapter",
		"node_id", nodeID,
		"upload", upload,
		"download", download,
	)

	return nil
}

// NodeStatusUpdaterAdapter adapts NodeRepository to NodeStatusUpdater interface
type NodeStatusUpdaterAdapter struct {
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

// NewNodeStatusUpdaterAdapter creates a new status updater adapter
func NewNodeStatusUpdaterAdapter(
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) usecases.NodeStatusUpdater {
	return &NodeStatusUpdaterAdapter{
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

// UpdateStatus updates node status and system information
func (a *NodeStatusUpdaterAdapter) UpdateStatus(ctx context.Context, nodeID uint, status string, onlineUsers int, systemInfo *usecases.SystemInfo) error {
	// Get node entity
	nodeEntity, err := a.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		a.logger.Errorw("failed to get node by ID",
			"error", err,
			"node_id", nodeID,
		)
		return err
	}

	// Update status if provided and valid
	if status != "" {
		// Status transitions are handled by the aggregate
		switch status {
		case "active", "online":
			if err := nodeEntity.Activate(); err != nil {
				a.logger.Warnw("failed to activate node",
					"error", err,
					"node_id", nodeID,
				)
			}
		case "inactive", "offline":
			if err := nodeEntity.Deactivate(); err != nil {
				a.logger.Warnw("failed to deactivate node",
					"error", err,
					"node_id", nodeID,
				)
			}
		}
	}

	// Update metadata with system information
	// Note: The current domain model doesn't directly support system info fields
	// This would need to be extended in the domain model if detailed tracking is required
	if systemInfo != nil {
		a.logger.Debugw("received system info",
			"node_id", nodeID,
			"load", systemInfo.Load,
			"memory_usage", systemInfo.MemoryUsage,
			"disk_usage", systemInfo.DiskUsage,
		)
		// For now, we log the system info
		// TODO: Extend domain model to support system metrics if needed
	}

	// Persist updated node
	if err := a.nodeRepo.Update(ctx, nodeEntity); err != nil {
		a.logger.Errorw("failed to update node",
			"error", err,
			"node_id", nodeID,
		)
		return err
	}

	a.logger.Infow("node status updated via adapter",
		"node_id", nodeID,
		"status", status,
		"online_users", onlineUsers,
	)

	return nil
}

// NodeLimitCheckerAdapter adapts to NodeLimitChecker interface
// Note: Traffic limits are now managed at subscription level, not node level
type NodeLimitCheckerAdapter struct {
	logger logger.Interface
}

// NewNodeLimitCheckerAdapter creates a new limit checker adapter
func NewNodeLimitCheckerAdapter(
	logger logger.Interface,
) usecases.NodeLimitChecker {
	return &NodeLimitCheckerAdapter{
		logger: logger,
	}
}

// CheckLimits checks if node has exceeded traffic limits
// Note: Traffic limits have been moved from node level to subscription level
// This method now returns unlimited for all nodes
func (a *NodeLimitCheckerAdapter) CheckLimits(ctx context.Context, nodeID uint) (exceeded bool, remaining uint64, err error) {
	// Traffic limits are now managed at subscription level, not node level
	// Following migration 007_remove_node_traffic_fields.sql
	// Nodes no longer have traffic_limit or traffic_used fields

	a.logger.Debugw("checked node limits - no limits at node level",
		"node_id", nodeID,
		"exceeded", false,
		"note", "traffic limits are managed at subscription level",
	)

	// Return unlimited - no limits at node level
	return false, 0, nil
}

// SubscriptionTrafficRecorderAdapter adapts to record subscription-based traffic
// This adapter records traffic by subscription_id for proper traffic tracking
//
// Architecture: XrayR → Adapter → MySQL
type SubscriptionTrafficRecorderAdapter struct {
	nodeTrafficRepo node.NodeTrafficRepository
	logger          logger.Interface
}

// NewSubscriptionTrafficRecorderAdapter creates a new subscription traffic recorder adapter
// Note: Directly writes to database for simplicity and reliability
func NewSubscriptionTrafficRecorderAdapter(
	nodeTrafficRepo node.NodeTrafficRepository,
	logger logger.Interface,
) usecases.SubscriptionTrafficRecorder {
	return &SubscriptionTrafficRecorderAdapter{
		nodeTrafficRepo: nodeTrafficRepo,
		logger:          logger,
	}
}

// RecordSubscriptionTraffic records subscription traffic data directly to database
func (a *SubscriptionTrafficRecorderAdapter) RecordSubscriptionTraffic(ctx context.Context, nodeID uint, subscriptionID int, upload, download int64) error {
	// Validate subscription ID
	if subscriptionID <= 0 {
		a.logger.Warnw("invalid subscription ID", "subscription_id", subscriptionID)
		return nil // Skip invalid subscription IDs
	}

	// Skip zero traffic
	if upload == 0 && download == 0 {
		return nil
	}

	// Create period for current hour aggregation
	period := time.Now().Truncate(time.Hour)

	// Create domain entity
	subIDUint := uint(subscriptionID)
	traffic, err := node.NewNodeTraffic(nodeID, &subIDUint, nil, period)
	if err != nil {
		a.logger.Errorw("failed to create node traffic entity",
			"error", err,
			"node_id", nodeID,
			"subscription_id", subscriptionID,
		)
		return err
	}

	// Accumulate traffic
	if err := traffic.Accumulate(uint64(upload), uint64(download)); err != nil {
		a.logger.Errorw("failed to accumulate traffic",
			"error", err,
			"node_id", nodeID,
			"subscription_id", subscriptionID,
		)
		return err
	}

	// Record in repository
	if err := a.nodeTrafficRepo.RecordTraffic(ctx, traffic); err != nil {
		a.logger.Errorw("failed to record subscription traffic",
			"error", err,
			"subscription_id", subscriptionID,
			"node_id", nodeID,
		)
		return err
	}

	a.logger.Debugw("subscription traffic recorded",
		"subscription_id", subscriptionID,
		"node_id", nodeID,
		"upload", upload,
		"download", download,
	)

	return nil
}

// BatchRecordSubscriptionTraffic records multiple subscriptions' traffic data directly to database
func (a *SubscriptionTrafficRecorderAdapter) BatchRecordSubscriptionTraffic(ctx context.Context, nodeID uint, items []usecases.SubscriptionTrafficItem) error {
	if len(items) == 0 {
		return nil
	}

	// Use current hour as period for consistent aggregation
	period := time.Now().Truncate(time.Hour)

	// Process each item
	validCount := 0
	errorCount := 0

	for _, item := range items {
		// Skip invalid subscription IDs
		if item.SubscriptionID <= 0 {
			a.logger.Warnw("skipping invalid subscription ID in batch",
				"subscription_id", item.SubscriptionID,
				"node_id", nodeID,
			)
			continue
		}

		// Skip zero traffic
		if item.Upload == 0 && item.Download == 0 {
			continue
		}

		// Create domain entity
		subIDUint := uint(item.SubscriptionID)
		traffic, err := node.NewNodeTraffic(nodeID, &subIDUint, nil, period)
		if err != nil {
			a.logger.Errorw("failed to create node traffic entity in batch",
				"error", err,
				"node_id", nodeID,
				"subscription_id", item.SubscriptionID,
			)
			errorCount++
			continue
		}

		// Accumulate traffic
		if err := traffic.Accumulate(uint64(item.Upload), uint64(item.Download)); err != nil {
			a.logger.Errorw("failed to accumulate traffic in batch",
				"error", err,
				"node_id", nodeID,
				"subscription_id", item.SubscriptionID,
			)
			errorCount++
			continue
		}

		// Record in repository
		if err := a.nodeTrafficRepo.RecordTraffic(ctx, traffic); err != nil {
			a.logger.Errorw("failed to record subscription traffic in batch",
				"error", err,
				"node_id", nodeID,
				"subscription_id", item.SubscriptionID,
			)
			errorCount++
			continue
		}

		validCount++
	}

	a.logger.Infow("subscription traffic batch processed",
		"node_id", nodeID,
		"success_count", validCount,
		"error_count", errorCount,
		"total_count", len(items),
	)

	// Return error if all items failed
	if validCount == 0 && errorCount > 0 {
		return fmt.Errorf("failed to record any traffic in batch")
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
) usecases.OnlineSubscriptionTracker {
	return &OnlineSubscriptionTrackerAdapter{
		logger: logger,
	}
}

// UpdateOnlineSubscriptions updates online subscriptions tracking
func (a *OnlineSubscriptionTrackerAdapter) UpdateOnlineSubscriptions(ctx context.Context, nodeID uint, subscriptions []usecases.OnlineSubscriptionInfo) error {
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
	logger logger.Interface
}

// NewNodeSystemStatusUpdaterAdapter creates a new system status updater adapter
func NewNodeSystemStatusUpdaterAdapter(
	logger logger.Interface,
) usecases.NodeSystemStatusUpdater {
	return &NodeSystemStatusUpdaterAdapter{
		logger: logger,
	}
}

// UpdateSystemStatus updates node system status metrics
func (a *NodeSystemStatusUpdaterAdapter) UpdateSystemStatus(ctx context.Context, nodeID uint, cpu, memory, disk float64, uptime int) error {
	// Log system status metrics
	// A full implementation would store these in a time-series database or monitoring system
	a.logger.Infow("node system status updated",
		"node_id", nodeID,
		"cpu", cpu,
		"memory", memory,
		"disk", disk,
		"uptime", uptime,
	)

	// TODO: Implement system metrics storage if needed
	// This could involve:
	// 1. Store in time-series database (InfluxDB, Prometheus)
	// 2. Store in separate metrics table
	// 3. Send to monitoring service (Grafana, Datadog)

	return nil
}
