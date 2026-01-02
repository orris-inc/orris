// Package crud provides HTTP handlers for forward agent CRUD operations.
package crud

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// SSE keepalive interval
	sseKeepAliveInterval = 30 * time.Second
)

// ForwardAgentSSEHandler handles SSE connections for forward agent events.
type ForwardAgentSSEHandler struct {
	adminHub *services.AdminHub
	logger   logger.Interface
}

// NewForwardAgentSSEHandler creates a new ForwardAgentSSEHandler.
func NewForwardAgentSSEHandler(adminHub *services.AdminHub, log logger.Interface) *ForwardAgentSSEHandler {
	return &ForwardAgentSSEHandler{
		adminHub: adminHub,
		logger:   log,
	}
}

// Events handles GET /forward-agents/events
// Establishes an SSE connection for real-time forward agent status updates.
func (h *ForwardAgentSSEHandler) Events(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userIDVal, exists := c.Get(constants.ContextKeyUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	// Parse agent_ids filter
	var agentFilters []string
	if agentIDs := c.Query("agent_ids"); agentIDs != "" {
		filters := strings.Split(agentIDs, ",")
		for _, f := range filters {
			f = strings.TrimSpace(f)
			if f != "" {
				agentFilters = append(agentFilters, f)
			}
		}
	}

	// Generate connection ID
	connID := uuid.New().String()

	// Register SSE connection with agent filters only (no node filters)
	conn := h.adminHub.RegisterConnWithFilters(connID, userID, nil, agentFilters)
	if conn == nil {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many connections"})
		return
	}

	// Set SSE headers
	// Note: CORS headers are handled by global CORS middleware
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable Nginx buffering

	h.logger.Infow("SSE connection established for forward agents",
		"conn_id", connID,
		"user_id", userID,
		"filters", agentFilters,
	)

	// Send initial connection event
	if _, err := c.Writer.WriteString(": connected\n\n"); err != nil {
		h.adminHub.UnregisterConn(connID)
		h.logger.Warnw("SSE initial write error",
			"conn_id", connID,
			"error", err,
		)
		return
	}
	c.Writer.Flush()

	// Create keepalive ticker
	keepAliveTicker := time.NewTicker(sseKeepAliveInterval)
	defer keepAliveTicker.Stop()

	// Get request context for cancellation
	ctx := c.Request.Context()

	// Event loop
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			h.adminHub.UnregisterConn(connID)
			h.logger.Infow("SSE connection closed by client",
				"conn_id", connID,
				"user_id", userID,
			)
			return

		case data, ok := <-conn.Send:
			if !ok {
				// Channel closed
				return
			}
			// Write event data
			_, err := c.Writer.Write(data)
			if err != nil {
				h.adminHub.UnregisterConn(connID)
				h.logger.Warnw("SSE write error",
					"conn_id", connID,
					"error", err,
				)
				return
			}
			c.Writer.Flush()

		case <-keepAliveTicker.C:
			// Send keepalive comment
			_, err := c.Writer.WriteString(": keepalive\n\n")
			if err != nil {
				h.adminHub.UnregisterConn(connID)
				h.logger.Warnw("SSE keepalive error",
					"conn_id", connID,
					"error", err,
				)
				return
			}
			c.Writer.Flush()
		}
	}
}
