// Package adapters provides infrastructure adapters.
package adapters

import (
	"context"
	"sync"
	"time"

	commondto "github.com/orris-inc/orris/internal/application/common/dto"
	nodeUsecases "github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// batchNodeStatusQueryTimeout is the maximum time allowed for batch node status queries.
	batchNodeStatusQueryTimeout = 10 * time.Second

	// nodeMetadataCacheTTL is the TTL for node metadata cache.
	nodeMetadataCacheTTL = 1 * time.Minute
)

// nodeMetadataCache holds cached node metadata.
type nodeMetadataCache struct {
	// allNodes maps nodeID -> metadata for all nodes
	allNodes map[uint]*node.NodeMetadata
	// sidToID maps SID -> nodeID for quick lookup
	sidToID map[string]uint
	// lastUpdated is the time when cache was last refreshed
	lastUpdated time.Time
	mu          sync.RWMutex
}

// NodeStatusQuerierAdapter implements services.NodeStatusQuerier.
// It fetches node status from Redis and resolves node metadata from database.
// Metadata is cached in memory to reduce database queries.
type NodeStatusQuerierAdapter struct {
	nodeRepo      node.NodeRepository
	statusQuerier *NodeSystemStatusQuerierAdapter
	cache         *nodeMetadataCache
	logger        logger.Interface
}

// NewNodeStatusQuerierAdapter creates a new NodeStatusQuerierAdapter.
func NewNodeStatusQuerierAdapter(
	nodeRepo node.NodeRepository,
	statusQuerier *NodeSystemStatusQuerierAdapter,
	log logger.Interface,
) *NodeStatusQuerierAdapter {
	return &NodeStatusQuerierAdapter{
		nodeRepo:      nodeRepo,
		statusQuerier: statusQuerier,
		cache: &nodeMetadataCache{
			allNodes: make(map[uint]*node.NodeMetadata),
			sidToID:  make(map[string]uint),
		},
		logger: log,
	}
}

// refreshCacheIfNeeded refreshes the metadata cache if it's expired.
func (a *NodeStatusQuerierAdapter) refreshCacheIfNeeded(ctx context.Context) error {
	a.cache.mu.RLock()
	needsRefresh := biztime.NowUTC().Sub(a.cache.lastUpdated) > nodeMetadataCacheTTL
	a.cache.mu.RUnlock()

	if !needsRefresh {
		return nil
	}

	// Acquire write lock and check again
	a.cache.mu.Lock()
	defer a.cache.mu.Unlock()

	// Double-check after acquiring write lock
	if biztime.NowUTC().Sub(a.cache.lastUpdated) <= nodeMetadataCacheTTL {
		return nil
	}

	// Refresh cache from database using lightweight query
	metadata, err := a.nodeRepo.GetAllMetadata(ctx)
	if err != nil {
		return err
	}

	// Rebuild cache
	a.cache.allNodes = make(map[uint]*node.NodeMetadata, len(metadata))
	a.cache.sidToID = make(map[string]uint, len(metadata))
	for _, m := range metadata {
		a.cache.allNodes[m.ID] = m
		a.cache.sidToID[m.SID] = m.ID
	}
	a.cache.lastUpdated = biztime.NowUTC()

	a.logger.Debugw("node metadata cache refreshed", "node_count", len(metadata))
	return nil
}

// GetBatchStatus returns status for multiple nodes by their SIDs.
// If nodeSIDs is nil, returns status for all active nodes.
// Returns a map of nodeSID -> (name, status).
func (a *NodeStatusQuerierAdapter) GetBatchStatus(nodeSIDs []string) (map[string]*services.AgentStatusData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), batchNodeStatusQueryTimeout)
	defer cancel()

	result := make(map[string]*services.AgentStatusData)

	// Refresh cache if needed
	if err := a.refreshCacheIfNeeded(ctx); err != nil {
		a.logger.Errorw("failed to refresh node metadata cache", "error", err)
		return nil, err
	}

	// Get metadata from cache
	a.cache.mu.RLock()
	var nodeIDs []uint
	var nodeMetadata []*node.NodeMetadata

	if nodeSIDs == nil {
		// Get all nodes from cache
		nodeIDs = make([]uint, 0, len(a.cache.allNodes))
		nodeMetadata = make([]*node.NodeMetadata, 0, len(a.cache.allNodes))
		for id, m := range a.cache.allNodes {
			nodeIDs = append(nodeIDs, id)
			nodeMetadata = append(nodeMetadata, m)
		}
	} else {
		// Get specific nodes from cache
		nodeIDs = make([]uint, 0, len(nodeSIDs))
		nodeMetadata = make([]*node.NodeMetadata, 0, len(nodeSIDs))
		for _, sid := range nodeSIDs {
			if id, ok := a.cache.sidToID[sid]; ok {
				nodeIDs = append(nodeIDs, id)
				if m, ok := a.cache.allNodes[id]; ok {
					nodeMetadata = append(nodeMetadata, m)
				}
			}
		}
	}
	a.cache.mu.RUnlock()

	if len(nodeIDs) == 0 {
		return result, nil
	}

	// Build ID to metadata mapping
	idToMetadata := make(map[uint]*node.NodeMetadata, len(nodeMetadata))
	for _, m := range nodeMetadata {
		idToMetadata[m.ID] = m
	}

	// Batch get status from Redis
	statusMap, err := a.statusQuerier.GetMultipleNodeSystemStatus(ctx, nodeIDs)
	if err != nil {
		a.logger.Errorw("failed to get batch node status from redis",
			"error", err,
			"node_count", len(nodeIDs),
		)
		return nil, err
	}

	// Build result map
	for nodeID, status := range statusMap {
		m, ok := idToMetadata[nodeID]
		if !ok {
			continue
		}

		result[m.SID] = &services.AgentStatusData{
			Name:   m.Name,
			Status: a.toStatusResponse(status),
		}
	}

	return result, nil
}

// toStatusResponse converts internal NodeSystemStatus to commondto.SystemStatus for consistent JSON output.
// This ensures the SSE response uses snake_case field names matching the forward agent events.
func (a *NodeStatusQuerierAdapter) toStatusResponse(status *nodeUsecases.NodeSystemStatus) *commondto.SystemStatus {
	if status == nil {
		return nil
	}
	return &commondto.SystemStatus{
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

		// Metadata
		UpdatedAt: status.UpdatedAt,
	}
}

// Ensure NodeStatusQuerierAdapter implements NodeStatusQuerier interface.
var _ services.NodeStatusQuerier = (*NodeStatusQuerierAdapter)(nil)
