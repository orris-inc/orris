// Package node provides HTTP handlers for node agent management.
package node

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// SubscriptionTrafficBufferWriter defines the interface for writing subscription traffic to buffer.
type SubscriptionTrafficBufferWriter interface {
	AddTraffic(nodeID, subscriptionID uint, upload, download int64)
}

const (
	nodeWriteWait  = 10 * time.Second
	nodePongWait   = 60 * time.Second
	nodePingPeriod = 30 * time.Second

	// eventTypeTraffic is the event type for traffic updates from node agents.
	eventTypeTraffic = "traffic"

	// maxTrafficPerReport is the maximum traffic bytes allowed per single report (1TB).
	// Prevents integer overflow attacks.
	maxNodeTrafficPerReport int64 = 1 << 40
)

var nodeUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Should be configured in production
	},
}

// trafficReport represents traffic data for a single subscription (matches orrisp api.TrafficReport).
type trafficReport struct {
	SubscriptionSID string `json:"subscription_id"`
	Upload          int64  `json:"upload"`
	Download        int64  `json:"download"`
}

// NodeHubHandler handles WebSocket connections for node agents.
type NodeHubHandler struct {
	hub                  *services.AgentHub
	trafficBuffer        SubscriptionTrafficBufferWriter
	subscriptionResolver usecases.SubscriptionIDResolver
	logger               logger.Interface
}

// NewNodeHubHandler creates a new NodeHubHandler.
func NewNodeHubHandler(
	hub *services.AgentHub,
	trafficBuffer SubscriptionTrafficBufferWriter,
	subscriptionResolver usecases.SubscriptionIDResolver,
	log logger.Interface,
) *NodeHubHandler {
	return &NodeHubHandler{
		hub:                  hub,
		trafficBuffer:        trafficBuffer,
		subscriptionResolver: subscriptionResolver,
		logger:               log,
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

	// Handle traffic event
	if event.EventType == eventTypeTraffic {
		h.handleTrafficEvent(nodeID, event.Extra)
		return
	}

	h.logger.Debugw("node agent event received",
		"node_id", nodeID,
		"event_type", event.EventType,
		"message", event.Message,
	)
}

// handleTrafficEvent processes traffic data from node agent.
func (h *NodeHubHandler) handleTrafficEvent(nodeID uint, extra any) {
	if extra == nil {
		return
	}

	// Parse traffic data from Extra field
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		h.logger.Warnw("failed to marshal traffic extra data",
			"error", err,
			"node_id", nodeID,
		)
		return
	}

	var reports []trafficReport
	if err := json.Unmarshal(extraBytes, &reports); err != nil {
		h.logger.Warnw("failed to parse traffic reports",
			"error", err,
			"node_id", nodeID,
		)
		return
	}

	if len(reports) == 0 {
		return
	}

	h.logger.Debugw("traffic event received",
		"node_id", nodeID,
		"reports_count", len(reports),
	)

	// Collect unique subscription SIDs
	sids := make([]string, 0, len(reports))
	for _, r := range reports {
		if r.SubscriptionSID != "" {
			sids = append(sids, r.SubscriptionSID)
		}
	}

	if len(sids) == 0 {
		return
	}

	// Resolve subscription SIDs to internal IDs
	ctx := context.Background()
	sidToID, err := h.subscriptionResolver.GetIDsBySIDs(ctx, sids)
	if err != nil {
		h.logger.Warnw("failed to resolve subscription SIDs",
			"error", err,
			"node_id", nodeID,
		)
		return
	}

	// Add traffic to buffer
	addedCount := 0
	for _, r := range reports {
		// Skip if SID not resolved
		subID, ok := sidToID[r.SubscriptionSID]
		if !ok {
			h.logger.Debugw("subscription SID not found, skipping",
				"subscription_sid", r.SubscriptionSID,
				"node_id", nodeID,
			)
			continue
		}

		// Validate traffic values
		if r.Upload < 0 || r.Download < 0 {
			h.logger.Warnw("negative traffic rejected",
				"subscription_sid", r.SubscriptionSID,
				"node_id", nodeID,
			)
			continue
		}

		// Reject excessively large values to prevent integer overflow
		if r.Upload > maxNodeTrafficPerReport || r.Download > maxNodeTrafficPerReport {
			h.logger.Warnw("excessive traffic rejected",
				"subscription_sid", r.SubscriptionSID,
				"node_id", nodeID,
				"upload", r.Upload,
				"download", r.Download,
			)
			continue
		}

		// Skip zero traffic
		if r.Upload == 0 && r.Download == 0 {
			continue
		}

		// Add to buffer (will be flushed to Redis periodically)
		h.trafficBuffer.AddTraffic(nodeID, subID, r.Upload, r.Download)
		addedCount++
	}

	if addedCount > 0 {
		h.logger.Debugw("traffic added to buffer",
			"node_id", nodeID,
			"added_count", addedCount,
		)
	}
}
