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
		CPUCores:       status.CPUCores,
		CPUModelName:   status.CPUModelName,
		CPUMHz:         status.CPUMHz,

		// Swap memory
		SwapTotal:   status.SwapTotal,
		SwapUsed:    status.SwapUsed,
		SwapPercent: status.SwapPercent,

		// Disk I/O
		DiskReadBytes:  status.DiskReadBytes,
		DiskWriteBytes: status.DiskWriteBytes,
		DiskReadRate:   status.DiskReadRate,
		DiskWriteRate:  status.DiskWriteRate,
		DiskIOPS:       status.DiskIOPS,

		// Pressure Stall Information (PSI)
		PSICPUSome:    status.PSICPUSome,
		PSICPUFull:    status.PSICPUFull,
		PSIMemorySome: status.PSIMemorySome,
		PSIMemoryFull: status.PSIMemoryFull,
		PSIIOSome:     status.PSIIOSome,
		PSIIOFull:     status.PSIIOFull,

		// Network extended stats
		NetworkRxPackets: status.NetworkRxPackets,
		NetworkTxPackets: status.NetworkTxPackets,
		NetworkRxErrors:  status.NetworkRxErrors,
		NetworkTxErrors:  status.NetworkTxErrors,
		NetworkRxDropped: status.NetworkRxDropped,
		NetworkTxDropped: status.NetworkTxDropped,

		// Socket statistics
		SocketsUsed:      status.SocketsUsed,
		SocketsTCPInUse:  status.SocketsTCPInUse,
		SocketsUDPInUse:  status.SocketsUDPInUse,
		SocketsTCPOrphan: status.SocketsTCPOrphan,
		SocketsTCPTW:     status.SocketsTCPTW,

		// Process statistics
		ProcessesTotal:   status.ProcessesTotal,
		ProcessesRunning: status.ProcessesRunning,
		ProcessesBlocked: status.ProcessesBlocked,

		// File descriptors
		FileNrAllocated: status.FileNrAllocated,
		FileNrMax:       status.FileNrMax,

		// Context switches and interrupts
		ContextSwitches: status.ContextSwitches,
		Interrupts:      status.Interrupts,

		// Kernel info
		KernelVersion: status.KernelVersion,
		Hostname:      status.Hostname,

		// Virtual memory statistics
		VMPageIn:  status.VMPageIn,
		VMPageOut: status.VMPageOut,
		VMSwapIn:  status.VMSwapIn,
		VMSwapOut: status.VMSwapOut,
		VMOOMKill: status.VMOOMKill,

		// Entropy pool
		EntropyAvailable: status.EntropyAvailable,
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
		CPUCores:       sysStatus.CPUCores,
		CPUModelName:   sysStatus.CPUModelName,
		CPUMHz:         sysStatus.CPUMHz,

		// Swap memory
		SwapTotal:   sysStatus.SwapTotal,
		SwapUsed:    sysStatus.SwapUsed,
		SwapPercent: sysStatus.SwapPercent,

		// Disk I/O
		DiskReadBytes:  sysStatus.DiskReadBytes,
		DiskWriteBytes: sysStatus.DiskWriteBytes,
		DiskReadRate:   sysStatus.DiskReadRate,
		DiskWriteRate:  sysStatus.DiskWriteRate,
		DiskIOPS:       sysStatus.DiskIOPS,

		// Pressure Stall Information (PSI)
		PSICPUSome:    sysStatus.PSICPUSome,
		PSICPUFull:    sysStatus.PSICPUFull,
		PSIMemorySome: sysStatus.PSIMemorySome,
		PSIMemoryFull: sysStatus.PSIMemoryFull,
		PSIIOSome:     sysStatus.PSIIOSome,
		PSIIOFull:     sysStatus.PSIIOFull,

		// Network extended stats
		NetworkRxPackets: sysStatus.NetworkRxPackets,
		NetworkTxPackets: sysStatus.NetworkTxPackets,
		NetworkRxErrors:  sysStatus.NetworkRxErrors,
		NetworkTxErrors:  sysStatus.NetworkTxErrors,
		NetworkRxDropped: sysStatus.NetworkRxDropped,
		NetworkTxDropped: sysStatus.NetworkTxDropped,

		// Socket statistics
		SocketsUsed:      sysStatus.SocketsUsed,
		SocketsTCPInUse:  sysStatus.SocketsTCPInUse,
		SocketsUDPInUse:  sysStatus.SocketsUDPInUse,
		SocketsTCPOrphan: sysStatus.SocketsTCPOrphan,
		SocketsTCPTW:     sysStatus.SocketsTCPTW,

		// Process statistics
		ProcessesTotal:   sysStatus.ProcessesTotal,
		ProcessesRunning: sysStatus.ProcessesRunning,
		ProcessesBlocked: sysStatus.ProcessesBlocked,

		// File descriptors
		FileNrAllocated: sysStatus.FileNrAllocated,
		FileNrMax:       sysStatus.FileNrMax,

		// Context switches and interrupts
		ContextSwitches: sysStatus.ContextSwitches,
		Interrupts:      sysStatus.Interrupts,

		// Kernel info
		KernelVersion: sysStatus.KernelVersion,
		Hostname:      sysStatus.Hostname,

		// Virtual memory statistics
		VMPageIn:  sysStatus.VMPageIn,
		VMPageOut: sysStatus.VMPageOut,
		VMSwapIn:  sysStatus.VMSwapIn,
		VMSwapOut: sysStatus.VMSwapOut,
		VMOOMKill: sysStatus.VMOOMKill,

		// Entropy pool
		EntropyAvailable: sysStatus.EntropyAvailable,

		// Metadata
		UpdatedAt: sysStatus.UpdatedAt,
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
