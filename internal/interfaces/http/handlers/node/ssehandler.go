// Package node provides HTTP handlers for node management.
package node

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

// NodeSSEHandler handles SSE connections for node events.
type NodeSSEHandler struct {
	adminHub *services.AdminHub
	logger   logger.Interface
}

// NewNodeSSEHandler creates a new NodeSSEHandler.
func NewNodeSSEHandler(adminHub *services.AdminHub, log logger.Interface) *NodeSSEHandler {
	return &NodeSSEHandler{
		adminHub: adminHub,
		logger:   log,
	}
}

// Events handles GET /nodes/events
// Establishes an SSE connection for real-time node status updates.
func (h *NodeSSEHandler) Events(c *gin.Context) {
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

	// Parse node_ids filter
	var nodeFilters []string
	if nodeIDs := c.Query("node_ids"); nodeIDs != "" {
		filters := strings.Split(nodeIDs, ",")
		for _, f := range filters {
			f = strings.TrimSpace(f)
			if f != "" {
				nodeFilters = append(nodeFilters, f)
			}
		}
	}

	// Generate connection ID
	connID := uuid.New().String()

	// Register SSE connection
	conn := h.adminHub.RegisterConn(connID, userID, nodeFilters)
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

	h.logger.Infow("SSE connection established",
		"conn_id", connID,
		"user_id", userID,
		"filters", nodeFilters,
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
