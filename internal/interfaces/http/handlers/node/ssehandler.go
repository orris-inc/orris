// Package node provides HTTP handlers for node management.
package node

import (
	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/interfaces/http/handlers/common"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// NodeSSEHandler handles SSE connections for node events.
type NodeSSEHandler struct {
	*common.SSEHandlerBase
}

// NewNodeSSEHandler creates a new NodeSSEHandler.
func NewNodeSSEHandler(adminHub *services.AdminHub, log logger.Interface) *NodeSSEHandler {
	return &NodeSSEHandler{
		SSEHandlerBase: common.NewSSEHandlerBase(adminHub, log),
	}
}

// Events handles GET /nodes/events
// Establishes an SSE connection for real-time node status updates.
// Supports Last-Event-ID header for reconnection replay.
func (h *NodeSSEHandler) Events(c *gin.Context) {
	// Get user ID from context
	userID, ok := h.GetUserID(c)
	if !ok {
		h.HandleUnauthorized(c, "unauthorized")
		return
	}

	// Parse node_ids filter
	nodeFilters := h.ParseFilterIDs(c, "node_ids")

	// Get Last-Event-ID for replay support
	lastEventID := h.GetLastEventID(c)

	// Generate connection ID
	connID := h.GenerateConnID()

	// Register SSE connection
	conn := h.GetAdminHub().RegisterConn(connID, userID, nodeFilters)
	if conn == nil {
		h.HandleTooManyRequests(c)
		return
	}

	// Set SSE headers
	h.SetupSSEResponse(c)

	// Send initial connection event
	if !h.SendInitialConnection(c) {
		h.HandleInitialWriteError(connID, nil)
		return
	}

	h.GetLogger().Debugw("node SSE connection established",
		"conn_id", connID,
		"user_id", userID,
		"node_filters", nodeFilters,
		"last_event_id", lastEventID,
	)

	// Replay missed events if reconnecting with Last-Event-ID
	if lastEventID != "" {
		if !h.ReplayMissedEvents(c, userID, lastEventID, "node", nodeFilters, connID, "node SSE") {
			h.HandleInitialWriteError(connID, nil)
			return
		}
	} else {
		// Send initial node status immediately after connection
		// This ensures the client gets current status without waiting for the next broadcast cycle
		h.GetAdminHub().BroadcastNodeStatusToConn(conn)
	}

	// Run event loop
	h.RunEventLoop(c, conn, connID, userID, "node SSE")
}
