// Package adapters provides infrastructure adapters.
package adapters

import (
	"context"
	"encoding/json"

	commondto "github.com/orris-inc/orris/internal/application/common/dto"
	nodeUsecases "github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// NodeSIDResolver resolves node internal ID to Stripe-style SID.
type NodeSIDResolver interface {
	GetSIDByID(nodeID uint) (string, bool)
}

// NodeStatusHandler handles status updates from node agents via WebSocket.
type NodeStatusHandler struct {
	statusUpdater   nodeUsecases.NodeSystemStatusUpdater
	lastSeenUpdater nodeUsecases.NodeLastSeenUpdater
	sidResolver     NodeSIDResolver
	adminHub        *services.AdminHub
	logger          logger.Interface
}

// NewNodeStatusHandler creates a new NodeStatusHandler.
func NewNodeStatusHandler(
	statusUpdater nodeUsecases.NodeSystemStatusUpdater,
	lastSeenUpdater nodeUsecases.NodeLastSeenUpdater,
	log logger.Interface,
) *NodeStatusHandler {
	return &NodeStatusHandler{
		statusUpdater:   statusUpdater,
		lastSeenUpdater: lastSeenUpdater,
		logger:          log,
	}
}

// SetAdminHub sets the AdminHub for SSE broadcasting.
func (h *NodeStatusHandler) SetAdminHub(hub *services.AdminHub, resolver NodeSIDResolver) {
	h.adminHub = hub
	h.sidResolver = resolver
}

// NodeStatusData represents the status data format from node agent WebSocket.
// This matches the ReportNodeStatusRequest format for consistency by embedding SystemStatus.
type NodeStatusData struct {
	commondto.SystemStatus
}

// HandleStatus processes status update from a node agent.
func (h *NodeStatusHandler) HandleStatus(nodeID uint, data any) {
	// Parse data to NodeStatusData
	dataBytes, err := json.Marshal(data)
	if err != nil {
		h.logger.Warnw("failed to marshal node status data",
			"error", err,
			"node_id", nodeID,
		)
		return
	}

	var status NodeStatusData
	if err := json.Unmarshal(dataBytes, &status); err != nil {
		h.logger.Warnw("failed to parse node status data",
			"error", err,
			"node_id", nodeID,
		)
		return
	}

	// Convert to NodeStatusUpdate for persistence
	statusUpdate := &nodeUsecases.NodeStatusUpdate{
		// System resources
		CPUPercent:    status.CPUPercent,
		MemoryPercent: status.MemoryPercent,
		MemoryUsed:    status.MemoryUsed,
		MemoryTotal:   status.MemoryTotal,
		MemoryAvail:   status.MemoryAvail,
		DiskPercent:   status.DiskPercent,
		DiskUsed:      status.DiskUsed,
		DiskTotal:     status.DiskTotal,
		UptimeSeconds: status.UptimeSeconds,

		// System load
		LoadAvg1:  status.LoadAvg1,
		LoadAvg5:  status.LoadAvg5,
		LoadAvg15: status.LoadAvg15,

		// Network statistics
		NetworkRxBytes: status.NetworkRxBytes,
		NetworkTxBytes: status.NetworkTxBytes,
		NetworkRxRate:  status.NetworkRxRate,
		NetworkTxRate:  status.NetworkTxRate,

		// Connection statistics
		TCPConnections: status.TCPConnections,
		UDPConnections: status.UDPConnections,

		// Network info
		PublicIPv4: status.PublicIPv4,
		PublicIPv6: status.PublicIPv6,

		// Agent info
		AgentVersion: status.AgentVersion,
		Platform:     status.Platform,
		Arch:         status.Arch,
	}

	// Persist status to Redis
	ctx := context.Background()
	if err := h.statusUpdater.UpdateSystemStatus(ctx, nodeID, statusUpdate); err != nil {
		h.logger.Errorw("failed to update node agent status via websocket",
			"error", err,
			"node_id", nodeID,
		)
		return
	}

	// Update last_seen_at, public IPs, and agent info (throttled at database layer)
	if h.lastSeenUpdater != nil {
		if err := h.lastSeenUpdater.UpdateLastSeenAt(ctx, nodeID, status.PublicIPv4, status.PublicIPv6, status.AgentVersion, status.Platform, status.Arch); err != nil {
			h.logger.Warnw("failed to update last_seen_at via websocket",
				"error", err,
				"node_id", nodeID,
			)
		}
	}

	h.logger.Debugw("node agent status updated via websocket",
		"node_id", nodeID,
		"cpu", status.CPUPercent,
		"memory", status.MemoryPercent,
		"uptime", status.UptimeSeconds,
	)

	// Broadcast status update via SSE (throttled by AdminHub)
	if h.adminHub != nil && h.sidResolver != nil {
		if nodeSID, ok := h.sidResolver.GetSIDByID(nodeID); ok {
			h.adminHub.BroadcastNodeStatus(nodeSID, statusUpdate)
		}
	}
}

// Ensure NodeStatusHandler implements StatusHandler interface.
var _ services.StatusHandler = (*NodeStatusHandler)(nil)
