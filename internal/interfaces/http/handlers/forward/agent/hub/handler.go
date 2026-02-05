// Package hub provides WebSocket hub handlers for forward agent connections.
package hub

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Should be configured in production
	},
}

// Handler handles WebSocket connections for forward agent hub.
type Handler struct {
	hub       *services.AgentHub
	agentRepo forward.AgentRepository
	logger    logger.Interface
}

// NewHandler creates a new Handler.
func NewHandler(hub *services.AgentHub, agentRepo forward.AgentRepository, log logger.Interface) *Handler {
	return &Handler{
		hub:       hub,
		agentRepo: agentRepo,
		logger:    log,
	}
}

// ForwardAgentWS handles WebSocket connections from forward agents.
// GET /ws/forward-agent
func (h *Handler) ForwardAgentWS(c *gin.Context) {
	agentIDVal, exists := c.Get("forward_agent_id")
	if !exists {
		h.logger.Warnw("forward_agent_id not found in context for hub ws",
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	agentID := agentIDVal.(uint)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Errorw("failed to upgrade to websocket",
			"error", err,
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		return
	}

	agentConn := h.hub.RegisterAgent(agentID, conn)

	h.logger.Infow("forward agent hub websocket connected",
		"agent_id", agentID,
		"ip", c.ClientIP(),
	)

	// Note: Full config sync is handled by OnAgentOnline callback in router.go
	// to centralize the logic and avoid duplicate sync calls.

	// Start read and write pumps
	go h.writePump(agentID, conn, agentConn.Send)
	h.readPump(agentID, conn)
}

// readPump reads messages from agent WebSocket.
func (h *Handler) readPump(agentID uint, conn *websocket.Conn) {
	defer func() {
		h.hub.UnregisterAgent(agentID)
		conn.Close()
	}()

	conn.SetReadLimit(65536)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Warnw("forward agent hub websocket read error",
					"error", err,
					"agent_id", agentID,
				)
			}
			break
		}

		var msg dto.HubMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			h.logger.Warnw("failed to parse forward agent hub message",
				"error", err,
				"agent_id", agentID,
			)
			continue
		}

		switch msg.Type {
		case dto.MsgTypeStatus:
			h.hub.HandleAgentStatus(agentID, msg.Data)
		case dto.MsgTypeHeartbeat:
			// Heartbeat handled by pong, just log
		case dto.MsgTypeEvent:
			h.handleAgentEvent(agentID, msg.Data)
		default:
			// Route to registered message handlers
			if !h.hub.RouteAgentMessage(agentID, msg.Type, msg.Data) {
				h.logger.Warnw("unhandled forward agent hub message type",
					"type", msg.Type,
					"agent_id", agentID,
				)
			}
		}
	}
}

// writePump writes messages to agent WebSocket.
func (h *Handler) writePump(agentID uint, conn *websocket.Conn, send chan *dto.HubMessage) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case msg, ok := <-send:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.WriteJSON(msg); err != nil {
				h.logger.Warnw("failed to write to forward agent hub websocket",
					"error", err,
					"agent_id", agentID,
				)
				return
			}

		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleAgentEvent processes event from agent.
func (h *Handler) handleAgentEvent(agentID uint, data any) {
	// Route to registered message handlers first (e.g., TrafficMessageHandler)
	// This allows domain-specific handlers to process events like "traffic"
	if h.hub.RouteAgentMessage(agentID, dto.MsgTypeEvent, data) {
		return // Event was handled by a registered handler
	}

	// Fallback: log unhandled events
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	var event dto.AgentEventData
	if err := json.Unmarshal(dataBytes, &event); err != nil {
		return
	}

	h.logger.Infow("forward agent event received",
		"agent_id", agentID,
		"event_type", event.EventType,
		"message", event.Message,
	)
}

// BroadcastAPIURLChangedRequest represents the request body for broadcasting API URL change to forward agents.
type BroadcastAPIURLChangedRequest struct {
	NewURL string `json:"new_url" binding:"required,url" example:"https://new-api.example.com"`
	Reason string `json:"reason,omitempty" example:"server migration"`
}

// BroadcastAPIURLChangedResponse represents the response for API URL change broadcast to forward agents.
type BroadcastAPIURLChangedResponse struct {
	AgentsNotified int `json:"agents_notified"`
	AgentsOnline   int `json:"agents_online"`
}

// BroadcastAPIURLChanged handles POST /forward-agents/broadcast-url-change
// Notifies connected forward agents that the API URL has changed.
// Forward agents should update their local configuration and reconnect to the new URL.
// Note: For nodes, use POST /nodes/broadcast-url-change instead.
func (h *Handler) BroadcastAPIURLChanged(c *gin.Context) {
	var req BroadcastAPIURLChangedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for broadcast API URL change",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	notified, online := h.hub.BroadcastAPIURLChanged(req.NewURL, req.Reason)

	// Get operator info for audit logging
	var operatorID any = "unknown"
	if userID, exists := c.Get("user_id"); exists {
		operatorID = userID
	}

	h.logger.Infow("API URL change broadcast to forward agents completed",
		"url_host", extractURLHost(req.NewURL),
		"reason", req.Reason,
		"agents_notified", notified,
		"agents_online", online,
		"operator_id", operatorID,
		"ip", c.ClientIP(),
	)

	utils.SuccessResponse(c, http.StatusOK, "API URL change broadcast to forward agents completed", &BroadcastAPIURLChangedResponse{
		AgentsNotified: notified,
		AgentsOnline:   online,
	})
}

// NotifyAPIURLChangedRequest represents the request body for notifying a single agent of API URL change.
type NotifyAPIURLChangedRequest struct {
	NewURL string `json:"new_url" binding:"required,url" example:"https://new-api.example.com"`
	Reason string `json:"reason,omitempty" example:"server migration"`
}

// NotifyAPIURLChangedResponse represents the response for single agent API URL change notification.
type NotifyAPIURLChangedResponse struct {
	AgentID  string `json:"agent_id"`
	Notified bool   `json:"notified"`
}

// NotifyAPIURLChanged handles POST /forward-agents/:id/url-change
// Notifies a specific connected forward agent that the API URL has changed.
func (h *Handler) NotifyAPIURLChanged(c *gin.Context) {
	agentSID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardAgent, "forward agent")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req NotifyAPIURLChangedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for notify API URL change",
			"error", err,
			"agent_sid", agentSID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	// Get agent by SID to retrieve internal ID
	agent, err := h.agentRepo.GetBySID(c.Request.Context(), agentSID)
	if err != nil {
		h.logger.Errorw("failed to get agent",
			"agent_sid", agentSID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponseWithError(c, err)
		return
	}
	if agent == nil {
		utils.ErrorResponseWithError(c, errors.NewNotFoundError("forward agent not found"))
		return
	}

	err = h.hub.NotifyAgentAPIURLChanged(agent.ID(), req.NewURL, req.Reason)
	if err != nil {
		h.logger.Warnw("failed to notify agent of API URL change",
			"error", err,
			"agent_id", agent.ID(),
			"agent_sid", agentSID,
			"url_host", extractURLHost(req.NewURL),
			"ip", c.ClientIP(),
		)
		utils.SuccessResponse(c, http.StatusOK, "agent not connected or send failed", &NotifyAPIURLChangedResponse{
			AgentID:  agentSID,
			Notified: false,
		})
		return
	}

	// Get operator info for audit logging
	var operatorID any = "unknown"
	if userID, exists := c.Get("user_id"); exists {
		operatorID = userID
	}

	h.logger.Infow("API URL change notification sent to forward agent",
		"agent_id", agent.ID(),
		"agent_sid", agentSID,
		"url_host", extractURLHost(req.NewURL),
		"reason", req.Reason,
		"operator_id", operatorID,
		"ip", c.ClientIP(),
	)

	utils.SuccessResponse(c, http.StatusOK, "API URL change notification sent", &NotifyAPIURLChangedResponse{
		AgentID:  agentSID,
		Notified: true,
	})
}

// extractURLHost extracts the host from a URL for safe logging (avoids leaking credentials).
func extractURLHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "<invalid-url>"
	}
	return parsed.Host
}
