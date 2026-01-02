// Package adapters provides infrastructure adapters.
package adapters

import (
	"context"
	"encoding/json"

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
// This matches the ReportNodeStatusRequest format for consistency.
type NodeStatusData struct {
	// System resources
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryUsed    uint64  `json:"memory_used"`
	MemoryTotal   uint64  `json:"memory_total"`
	MemoryAvail   uint64  `json:"memory_avail"`
	DiskPercent   float64 `json:"disk_percent"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
	UptimeSeconds int64   `json:"uptime_seconds"`

	// System load
	LoadAvg1  float64 `json:"load_avg_1"`
	LoadAvg5  float64 `json:"load_avg_5"`
	LoadAvg15 float64 `json:"load_avg_15"`

	// Network statistics
	NetworkRxBytes uint64 `json:"network_rx_bytes"`
	NetworkTxBytes uint64 `json:"network_tx_bytes"`
	NetworkRxRate  uint64 `json:"network_rx_rate"`
	NetworkTxRate  uint64 `json:"network_tx_rate"`

	// Connection statistics
	TCPConnections int `json:"tcp_connections"`
	UDPConnections int `json:"udp_connections"`

	// Network info
	PublicIPv4 string `json:"public_ipv4,omitempty"`
	PublicIPv6 string `json:"public_ipv6,omitempty"`

	// Agent info
	AgentVersion string `json:"agent_version,omitempty"`
	Platform     string `json:"platform,omitempty"`
	Arch         string `json:"arch,omitempty"`
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
