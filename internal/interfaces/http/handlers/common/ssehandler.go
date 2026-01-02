// Package common provides shared HTTP handler utilities.
package common

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
	// SSEKeepaliveInterval is the interval for sending keepalive messages.
	SSEKeepaliveInterval = 30 * time.Second

	// SSEContentType is the content type for SSE responses.
	SSEContentType = "text/event-stream"

	// MaxFilterIDs is the maximum number of filter IDs allowed in a single request.
	// This prevents memory exhaustion attacks via large filter lists.
	MaxFilterIDs = 100

	// MaxFilterIDLength is the maximum length of a single filter ID.
	// SID format: "fa_xK9mP2vL3nQ" or "node_xK9mP2vL3nQ" (max ~20 chars)
	MaxFilterIDLength = 32
)

// SSEHandlerBase provides common SSE functionality for agent and node handlers.
type SSEHandlerBase struct {
	adminHub *services.AdminHub
	logger   logger.Interface
}

// NewSSEHandlerBase creates a new SSEHandlerBase.
func NewSSEHandlerBase(adminHub *services.AdminHub, log logger.Interface) *SSEHandlerBase {
	return &SSEHandlerBase{
		adminHub: adminHub,
		logger:   log,
	}
}

// GetAdminHub returns the admin hub instance.
func (h *SSEHandlerBase) GetAdminHub() *services.AdminHub {
	return h.adminHub
}

// GetLogger returns the logger instance.
func (h *SSEHandlerBase) GetLogger() logger.Interface {
	return h.logger
}

// SetupSSEResponse sets common SSE response headers.
// Note: CORS headers are handled by global CORS middleware.
func (h *SSEHandlerBase) SetupSSEResponse(c *gin.Context) {
	c.Header("Content-Type", SSEContentType)
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable Nginx buffering
}

// GenerateConnID generates a unique connection ID.
func (h *SSEHandlerBase) GenerateConnID() string {
	return uuid.New().String()
}

// ParseFilterIDs parses filter IDs from query parameter.
// Returns nil if the parameter is empty.
// Limits the number of filter IDs to MaxFilterIDs and each ID length to MaxFilterIDLength
// to prevent memory exhaustion attacks.
func (h *SSEHandlerBase) ParseFilterIDs(c *gin.Context, paramName string) []string {
	idsParam := c.Query(paramName)
	if idsParam == "" {
		return nil
	}

	var filters []string
	parts := strings.Split(idsParam, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Skip IDs that are too long (invalid SID format)
		if len(p) > MaxFilterIDLength {
			continue
		}
		filters = append(filters, p)
		// Limit the number of filter IDs to prevent memory exhaustion
		if len(filters) >= MaxFilterIDs {
			h.logger.Warnw("filter IDs truncated to max limit",
				"param", paramName,
				"max", MaxFilterIDs,
			)
			break
		}
	}
	return filters
}

// GetUserID extracts the user ID from the request context.
// Returns the user ID and true if successful, or 0 and false if not found or invalid.
func (h *SSEHandlerBase) GetUserID(c *gin.Context) (uint, bool) {
	userIDVal, exists := c.Get(constants.ContextKeyUserID)
	if !exists {
		return 0, false
	}
	userID, ok := userIDVal.(uint)
	return userID, ok
}

// HandleUnauthorized sends an unauthorized error response.
func (h *SSEHandlerBase) HandleUnauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": message})
}

// HandleTooManyRequests sends a too many requests error response.
func (h *SSEHandlerBase) HandleTooManyRequests(c *gin.Context) {
	c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many connections"})
}

// SendInitialConnection sends the initial SSE connection comment.
// Returns true if successful, false if write failed.
func (h *SSEHandlerBase) SendInitialConnection(c *gin.Context) bool {
	if _, err := c.Writer.WriteString(": connected\n\n"); err != nil {
		return false
	}
	c.Writer.Flush()
	return true
}

// RunEventLoop runs the SSE event loop with keepalive.
// This is a blocking call that handles receiving data, keepalive, and disconnect.
// The loop exits when the client disconnects or an error occurs.
func (h *SSEHandlerBase) RunEventLoop(c *gin.Context, conn *services.SSEConn, connID string, userID uint, logPrefix string) {
	// Create keepalive ticker
	keepAliveTicker := time.NewTicker(SSEKeepaliveInterval)
	defer keepAliveTicker.Stop()

	// Get request context for cancellation
	ctx := c.Request.Context()

	// Event loop
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			h.adminHub.UnregisterConn(connID)
			h.logger.Infow(logPrefix+" connection closed by client",
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
			if _, err := c.Writer.Write(data); err != nil {
				h.adminHub.UnregisterConn(connID)
				h.logger.Warnw(logPrefix+" write error",
					"conn_id", connID,
					"error", err,
				)
				return
			}
			c.Writer.Flush()

		case <-keepAliveTicker.C:
			// Send keepalive comment
			if _, err := c.Writer.WriteString(": keepalive\n\n"); err != nil {
				h.adminHub.UnregisterConn(connID)
				h.logger.Warnw(logPrefix+" keepalive error",
					"conn_id", connID,
					"error", err,
				)
				return
			}
			c.Writer.Flush()
		}
	}
}

// HandleInitialWriteError handles the case when initial SSE write fails.
func (h *SSEHandlerBase) HandleInitialWriteError(connID string, err error) {
	h.adminHub.UnregisterConn(connID)
	h.logger.Warnw("SSE initial write error",
		"conn_id", connID,
		"error", err,
	)
}
