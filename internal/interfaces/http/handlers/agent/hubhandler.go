// Package agent provides HTTP handlers for agent management.
package agent

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/infrastructure/services"
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

// HubHandler handles WebSocket connections for forward agent hub.
type HubHandler struct {
	hub    *services.AgentHub
	logger logger.Interface
}

// NewHubHandler creates a new HubHandler.
func NewHubHandler(hub *services.AgentHub, log logger.Interface) *HubHandler {
	return &HubHandler{
		hub:    hub,
		logger: log,
	}
}

// ForwardAgentWS handles WebSocket connections from forward agents.
// GET /ws/forward-agent
func (h *HubHandler) ForwardAgentWS(c *gin.Context) {
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

	// Start read and write pumps
	go h.writePump(agentID, conn, agentConn.Send)
	h.readPump(agentID, conn)
}

// readPump reads messages from agent WebSocket.
func (h *HubHandler) readPump(agentID uint, conn *websocket.Conn) {
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
func (h *HubHandler) writePump(agentID uint, conn *websocket.Conn, send chan *dto.HubMessage) {
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
func (h *HubHandler) handleAgentEvent(agentID uint, data any) {
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
