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

	// Convert to NodeStatusUpdate for persistence.
	// Both types embed commondto.SystemStatus, so direct assignment works.
	statusUpdate := &nodeUsecases.NodeStatusUpdate{
		SystemStatus: status.SystemStatus,
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

	// Note: Status is now broadcast via aggregated SSE push in AdminHub.nodeBroadcastLoop()
	// instead of individual pushes. This reduces push frequency and allows clients to
	// receive all node statuses in a single batch event.
}

// Ensure NodeStatusHandler implements StatusHandler interface.
var _ services.StatusHandler = (*NodeStatusHandler)(nil)
