// Package node provides HTTP handlers for node agent management.
package node

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

const (
	nodeWriteWait  = 10 * time.Second
	nodePongWait   = 60 * time.Second
	nodePingPeriod = 30 * time.Second
)

var nodeUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Should be configured in production
	},
}

// NodeHubHandler handles WebSocket connections for node agents.
type NodeHubHandler struct {
	hub    *services.AgentHub
	logger logger.Interface
}

// NewNodeHubHandler creates a new NodeHubHandler.
func NewNodeHubHandler(hub *services.AgentHub, log logger.Interface) *NodeHubHandler {
	return &NodeHubHandler{
		hub:    hub,
		logger: log,
	}
}

// NodeAgentWS handles WebSocket connections from node agents.
// GET /ws/node-agent?token=xxx
func (h *NodeHubHandler) NodeAgentWS(c *gin.Context) {
	nodeIDVal, exists := c.Get("node_id")
	if !exists {
		h.logger.Warnw("node_id not found in context for hub ws",
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	nodeID, ok := nodeIDVal.(uint)
	if !ok {
		h.logger.Errorw("invalid node_id type in context",
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "internal error")
		return
	}

	conn, err := nodeUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Errorw("failed to upgrade to websocket",
			"error", err,
			"node_id", nodeID,
			"ip", c.ClientIP(),
		)
		return
	}

	nodeConn := h.hub.RegisterNodeAgent(nodeID, conn)

	h.logger.Infow("node agent hub websocket connected",
		"node_id", nodeID,
		"ip", c.ClientIP(),
	)

	// Start read and write pumps
	go h.writePump(nodeID, conn, nodeConn.Send)
	h.readPump(nodeID, conn)
}

// readPump reads messages from node agent WebSocket.
func (h *NodeHubHandler) readPump(nodeID uint, conn *websocket.Conn) {
	defer func() {
		h.hub.UnregisterNodeAgent(nodeID)
		conn.Close()
	}()

	conn.SetReadLimit(65536)
	conn.SetReadDeadline(time.Now().Add(nodePongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(nodePongWait))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Warnw("node agent hub websocket read error",
					"error", err,
					"node_id", nodeID,
				)
			}
			break
		}

		var msg dto.NodeHubMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			h.logger.Warnw("failed to parse node agent hub message",
				"error", err,
				"node_id", nodeID,
			)
			continue
		}

		switch msg.Type {
		case dto.NodeMsgTypeStatus:
			h.hub.HandleNodeStatus(nodeID, msg.Data)
		case dto.NodeMsgTypeHeartbeat:
			// Heartbeat handled by pong, just log
		case dto.NodeMsgTypeEvent:
			h.handleNodeEvent(nodeID, msg.Data)
		default:
			h.logger.Warnw("unhandled node agent hub message type",
				"type", msg.Type,
				"node_id", nodeID,
			)
		}
	}
}

// writePump writes messages to node agent WebSocket.
func (h *NodeHubHandler) writePump(nodeID uint, conn *websocket.Conn, send chan []byte) {
	ticker := time.NewTicker(nodePingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case msg, ok := <-send:
			conn.SetWriteDeadline(time.Now().Add(nodeWriteWait))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				h.logger.Warnw("failed to write to node agent hub websocket",
					"error", err,
					"node_id", nodeID,
				)
				return
			}

		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(nodeWriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleNodeEvent processes event from node agent.
func (h *NodeHubHandler) handleNodeEvent(nodeID uint, data any) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		h.logger.Warnw("failed to marshal node event data",
			"error", err,
			"node_id", nodeID,
		)
		return
	}

	var event dto.NodeEventData
	if err := json.Unmarshal(dataBytes, &event); err != nil {
		h.logger.Warnw("failed to parse node event data",
			"error", err,
			"node_id", nodeID,
		)
		return
	}

	h.logger.Infow("node agent event received",
		"node_id", nodeID,
		"event_type", event.EventType,
		"message", event.Message,
	)
}
