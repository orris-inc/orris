package adapters

import (
	"context"
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

// UserTrafficRecorderAdapter adapts UserTrafficRepository to UserTrafficRecorder interface
type UserTrafficRecorderAdapter struct {
	userTrafficRepo node.UserTrafficRepository
	logger          logger.Interface
}

// NewUserTrafficRecorderAdapter creates a new user traffic recorder adapter
func NewUserTrafficRecorderAdapter(
	userTrafficRepo node.UserTrafficRepository,
	logger logger.Interface,
) usecases.UserTrafficRecorder {
	return &UserTrafficRecorderAdapter{
		userTrafficRepo: userTrafficRepo,
		logger:          logger,
	}
}

// RecordUserTraffic records user traffic data
func (a *UserTrafficRecorderAdapter) RecordUserTraffic(ctx context.Context, nodeID uint, userID int, upload, download int64) error {
	// Convert int to uint for userID
	if userID <= 0 {
		a.logger.Warnw("invalid user ID", "user_id", userID)
		return nil // Skip invalid user IDs
	}

	// Use increment traffic for atomic updates
	err := a.userTrafficRepo.IncrementTraffic(ctx, uint(userID), nodeID, uint64(upload), uint64(download))
	if err != nil {
		a.logger.Errorw("failed to increment user traffic",
			"error", err,
			"user_id", userID,
			"node_id", nodeID,
		)
		return err
	}

	a.logger.Debugw("user traffic recorded via adapter",
		"user_id", userID,
		"node_id", nodeID,
		"upload", upload,
		"download", download,
	)

	return nil
}

// BatchRecordUserTraffic records multiple users' traffic data in a single batch operation
func (a *UserTrafficRecorderAdapter) BatchRecordUserTraffic(ctx context.Context, nodeID uint, items []usecases.UserTrafficItem) error {
	if len(items) == 0 {
		return nil
	}

	// Use current hour as period for consistent aggregation
	period := time.Now().Truncate(time.Hour)

	// Convert items to domain entities
	traffics := make([]*node.UserTraffic, 0, len(items))
	for _, item := range items {
		// Skip invalid user IDs
		if item.UserID <= 0 {
			a.logger.Warnw("skipping invalid user ID in batch",
				"user_id", item.UserID,
				"node_id", nodeID,
			)
			continue
		}

		// Skip zero traffic
		if item.Upload == 0 && item.Download == 0 {
			continue
		}

		// Create user traffic entity
		traffic, err := node.NewUserTraffic(uint(item.UserID), nodeID, nil, period)
		if err != nil {
			a.logger.Errorw("failed to create user traffic entity",
				"error", err,
				"user_id", item.UserID,
				"node_id", nodeID,
			)
			continue
		}

		// Accumulate traffic
		if err := traffic.Accumulate(uint64(item.Upload), uint64(item.Download)); err != nil {
			a.logger.Errorw("failed to accumulate traffic",
				"error", err,
				"user_id", item.UserID,
				"node_id", nodeID,
			)
			continue
		}

		traffics = append(traffics, traffic)
	}

	// If no valid traffic data, return early
	if len(traffics) == 0 {
		a.logger.Debugw("no valid traffic data to record in batch",
			"node_id", nodeID,
			"original_count", len(items),
		)
		return nil
	}

	// Batch upsert to database
	err := a.userTrafficRepo.BatchUpsert(ctx, traffics)
	if err != nil {
		a.logger.Errorw("failed to batch upsert user traffic",
			"error", err,
			"node_id", nodeID,
			"count", len(traffics),
		)
		return err
	}

	a.logger.Infow("user traffic batch recorded via adapter",
		"node_id", nodeID,
		"users_recorded", len(traffics),
		"original_count", len(items),
	)

	return nil
}

// OnlineUserTrackerAdapter adapts to OnlineUserTracker interface
type OnlineUserTrackerAdapter struct {
	logger logger.Interface
}

// NewOnlineUserTrackerAdapter creates a new online user tracker adapter
func NewOnlineUserTrackerAdapter(
	logger logger.Interface,
) usecases.OnlineUserTracker {
	return &OnlineUserTrackerAdapter{
		logger: logger,
	}
}

// UpdateOnlineUsers updates online users tracking
func (a *OnlineUserTrackerAdapter) UpdateOnlineUsers(ctx context.Context, nodeID uint, users []usecases.OnlineUserInfo) error {
	// For now, we just log the online users
	// A full implementation would need a cache (Redis) or database table to track online users
	a.logger.Infow("online users updated",
		"node_id", nodeID,
		"count", len(users),
	)

	// TODO: Implement Redis-based online user tracking if needed
	// This would involve:
	// 1. Store user IPs and timestamps in Redis with expiry
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
func (a *NodeSystemStatusUpdaterAdapter) UpdateSystemStatus(ctx context.Context, nodeID uint, cpu, memory, disk float64, networkUsage string, uptime int) error {
	// Log system status metrics
	// A full implementation would store these in a time-series database or monitoring system
	a.logger.Infow("node system status updated",
		"node_id", nodeID,
		"cpu", cpu,
		"memory", memory,
		"disk", disk,
		"network", networkUsage,
		"uptime", uptime,
	)

	// TODO: Implement system metrics storage if needed
	// This could involve:
	// 1. Store in time-series database (InfluxDB, Prometheus)
	// 2. Store in separate metrics table
	// 3. Send to monitoring service (Grafana, Datadog)

	return nil
}
