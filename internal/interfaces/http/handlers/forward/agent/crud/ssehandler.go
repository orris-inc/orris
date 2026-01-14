// Package crud provides HTTP handlers for forward agent CRUD operations.
package crud

import (
	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/interfaces/http/handlers/common"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ForwardAgentSSEHandler handles SSE connections for forward agent events.
type ForwardAgentSSEHandler struct {
	*common.SSEHandlerBase
}

// NewForwardAgentSSEHandler creates a new ForwardAgentSSEHandler.
func NewForwardAgentSSEHandler(adminHub *services.AdminHub, log logger.Interface) *ForwardAgentSSEHandler {
	return &ForwardAgentSSEHandler{
		SSEHandlerBase: common.NewSSEHandlerBase(adminHub, log),
	}
}

// Events handles GET /forward-agents/events
// Establishes an SSE connection for real-time forward agent status updates.
// Supports Last-Event-ID header for reconnection replay.
func (h *ForwardAgentSSEHandler) Events(c *gin.Context) {
	// Get user ID from context
	userID, ok := h.GetUserID(c)
	if !ok {
		h.HandleUnauthorized(c, "unauthorized")
		return
	}

	// Parse agent_ids filter
	agentFilters := h.ParseFilterIDs(c, "agent_ids")

	// Get Last-Event-ID for replay support
	lastEventID := h.GetLastEventID(c)

	// Generate connection ID
	connID := h.GenerateConnID()

	// Register SSE connection with agent filters only (no node filters)
	conn := h.GetAdminHub().RegisterConnWithFilters(connID, userID, nil, agentFilters)
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

	h.GetLogger().Debugw("forward agent SSE connection established",
		"conn_id", connID,
		"user_id", userID,
		"agent_filters", agentFilters,
		"last_event_id", lastEventID,
	)

	// Replay missed events if reconnecting with Last-Event-ID
	if lastEventID != "" {
		if !h.ReplayMissedEvents(c, userID, lastEventID, "agent", agentFilters, connID, "forward agent SSE") {
			h.HandleInitialWriteError(connID, nil)
			return
		}
	} else {
		// Send initial agent status immediately after connection
		// This ensures the client gets current status without waiting for the next broadcast cycle
		h.GetAdminHub().BroadcastAgentStatusToConn(conn)
	}

	// Run event loop
	h.RunEventLoop(c, conn, connID, userID, "forward agent SSE")
}
