// Package node provides HTTP handlers for node agent management.
package node

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/id"
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

// NodeAddressChangeNotifier defines the interface for notifying node address changes.
type NodeAddressChangeNotifier interface {
	NotifyNodeAddressChange(ctx context.Context, nodeID uint) error
}

// NodeIPUpdater defines the interface for updating node public IPs.
type NodeIPUpdater interface {
	UpdatePublicIP(ctx context.Context, nodeID uint, ipv4, ipv6 string) error
}

// SubscriptionSyncer defines the interface for syncing subscriptions to nodes on connect.
type SubscriptionSyncer interface {
	SyncSubscriptionsOnNodeConnect(ctx context.Context, nodeID uint) error
}

// NodeTrafficEnforcer defines the interface for node subscription traffic limit enforcement.
// When traffic is recorded, the enforcer checks if the subscription has exceeded its limit
// and suspends it if necessary.
type NodeTrafficEnforcer interface {
	// CheckAndEnforceLimitForNode checks traffic limit for node subscriptions only.
	// Skips forward and hybrid subscriptions.
	CheckAndEnforceLimitForNode(ctx context.Context, subscriptionID uint) error
}

// NodeSubscriptionUsageReader defines the interface for reading node subscription usage.
type NodeSubscriptionUsageReader interface {
	// GetCurrentPeriodUsage returns the total usage for the current billing period.
	GetCurrentPeriodUsage(ctx context.Context, subscriptionID uint, periodStart, periodEnd time.Time) (int64, error)
}

// NodeSubscriptionQuotaCache defines the interface for subscription quota caching.
// This is a local interface to avoid import cycle with cache package.
type NodeSubscriptionQuotaCache interface {
	// GetQuota retrieves subscription quota information from cache.
	// Returns nil if cache does not exist.
	GetQuota(ctx context.Context, subscriptionID uint) (*CachedQuotaInfo, error)

	// MarkSuspended marks the subscription as suspended in cache.
	MarkSuspended(ctx context.Context, subscriptionID uint) error
}

// CachedQuotaInfo represents the cached subscription quota information.
// This mirrors cache.CachedQuota to avoid import cycle.
type CachedQuotaInfo struct {
	Limit       int64     // Traffic limit in bytes
	PeriodStart time.Time // Billing period start
	PeriodEnd   time.Time // Billing period end
	PlanType    string    // node/forward/hybrid
	Suspended   bool      // Whether the subscription is suspended
	NotFound    bool      // Null marker: subscription confirmed not found/inactive in DB
}

// NodeSubscriptionQuotaLoader defines the interface for lazy loading subscription quota.
// This is used when quota cache miss occurs to load quota from database.
type NodeSubscriptionQuotaLoader interface {
	// LoadQuotaByID loads subscription quota from database and caches it.
	// Returns the cached quota info, or nil if subscription/plan not found.
	LoadQuotaByID(ctx context.Context, subscriptionID uint) (*CachedQuotaInfo, error)
}

// NodeHubHandler handles WebSocket connections for node agents.
type NodeHubHandler struct {
	hub                   *services.AgentHub
	nodeRepo              node.NodeRepository
	trafficBuffer         SubscriptionTrafficBufferWriter
	subscriptionResolver  usecases.SubscriptionIDResolver
	addressChangeNotifier NodeAddressChangeNotifier
	ipUpdater             NodeIPUpdater
	subscriptionSyncer    SubscriptionSyncer
	trafficEnforcer       NodeTrafficEnforcer
	logger                logger.Interface

	// quotaCache caches subscription quota info for real-time traffic limit checking
	quotaCache NodeSubscriptionQuotaCache

	// usageReader reads current period usage for traffic limit comparison
	usageReader NodeSubscriptionUsageReader

	// quotaLoader loads quota from database when cache miss occurs (lazy loading)
	quotaLoader NodeSubscriptionQuotaLoader
}

// NewNodeHubHandler creates a new NodeHubHandler.
func NewNodeHubHandler(
	hub *services.AgentHub,
	nodeRepo node.NodeRepository,
	trafficBuffer SubscriptionTrafficBufferWriter,
	subscriptionResolver usecases.SubscriptionIDResolver,
	log logger.Interface,
) *NodeHubHandler {
	return &NodeHubHandler{
		hub:                  hub,
		nodeRepo:             nodeRepo,
		trafficBuffer:        trafficBuffer,
		subscriptionResolver: subscriptionResolver,
		logger:               log,
	}
}

// SetAddressChangeNotifier sets the address change notifier (optional).
func (h *NodeHubHandler) SetAddressChangeNotifier(notifier NodeAddressChangeNotifier) {
	h.addressChangeNotifier = notifier
}

// SetIPUpdater sets the IP updater (optional).
func (h *NodeHubHandler) SetIPUpdater(updater NodeIPUpdater) {
	h.ipUpdater = updater
}

// SetSubscriptionSyncer sets the subscription syncer for pushing subscriptions on connect (optional).
func (h *NodeHubHandler) SetSubscriptionSyncer(syncer SubscriptionSyncer) {
	h.subscriptionSyncer = syncer
}

// SetTrafficEnforcer sets the traffic enforcer for node subscription limit checking (optional).
func (h *NodeHubHandler) SetTrafficEnforcer(enforcer NodeTrafficEnforcer) {
	h.trafficEnforcer = enforcer
}

// SetQuotaCache sets the quota cache for real-time traffic limit checking (optional).
func (h *NodeHubHandler) SetQuotaCache(cache NodeSubscriptionQuotaCache) {
	h.quotaCache = cache
}

// SetUsageReader sets the usage reader for real-time traffic limit checking (optional).
func (h *NodeHubHandler) SetUsageReader(reader NodeSubscriptionUsageReader) {
	h.usageReader = reader
}

// SetQuotaLoader sets the quota loader for lazy loading when cache miss occurs (optional).
func (h *NodeHubHandler) SetQuotaLoader(loader NodeSubscriptionQuotaLoader) {
	h.quotaLoader = loader
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

	clientIP := c.ClientIP()
	h.logger.Infow("node agent hub websocket connected",
		"node_id", nodeID,
		"ip", clientIP,
	)

	// Check if node IP has changed, update database, and notify forward agents
	h.checkAndNotifyIPChange(c.Request.Context(), nodeID, clientIP)

	// Sync subscriptions to node on connect (async to not block connection)
	h.syncSubscriptionsOnConnect(nodeID)

	// Start read and write pumps
	goroutine.SafeGo(h.logger, "node-agent-write-pump", func() {
		h.writePump(nodeID, conn, nodeConn.Send)
	})
	h.readPump(nodeID, conn)
}

// checkAndNotifyIPChange checks if the node's public IP has changed,
// updates the database first, then notifies forward agents.
func (h *NodeHubHandler) checkAndNotifyIPChange(ctx context.Context, nodeID uint, newIP string) {
	if h.addressChangeNotifier == nil || h.ipUpdater == nil || newIP == "" {
		return
	}

	// Get current IPs from database
	currentIPv4, currentIPv6, err := h.nodeRepo.GetPublicIPs(ctx, nodeID)
	if err != nil {
		h.logger.Warnw("failed to get current public IPs for change detection",
			"error", err,
			"node_id", nodeID,
		)
		return
	}

	// Check if the new IP differs from current IP
	// newIP could be IPv4 or IPv6 depending on the connection
	isIPv4 := false
	parsedIP := net.ParseIP(newIP)
	if parsedIP == nil {
		return
	}

	// Determine IP version and check if changed (including first-time set)
	var ipChanged bool
	if parsedIP.To4() != nil {
		isIPv4 = true
		ipChanged = newIP != currentIPv4
	} else {
		ipChanged = newIP != currentIPv6
	}

	if !ipChanged {
		return
	}

	h.logger.Infow("node IP changed on websocket connect",
		"node_id", nodeID,
		"new_ip", newIP,
		"is_ipv4", isIPv4,
		"current_ipv4", currentIPv4,
		"current_ipv6", currentIPv6,
	)

	// Step 1: Update database first (so that sync reads the new IP)
	var updateIPv4, updateIPv6 string
	if isIPv4 {
		updateIPv4 = newIP
	} else {
		updateIPv6 = newIP
	}
	if err := h.ipUpdater.UpdatePublicIP(ctx, nodeID, updateIPv4, updateIPv6); err != nil {
		h.logger.Warnw("failed to update node public IP",
			"error", err,
			"node_id", nodeID,
			"new_ip", newIP,
		)
		return
	}

	h.logger.Infow("node IP updated in database, notifying forward agents",
		"node_id", nodeID,
		"new_ip", newIP,
	)

	// Step 2: Notify forward agents asynchronously (now database has the new IP)
	goroutine.SafeGo(h.logger, "notify-node-ip-change", func() {
		notifyCtx := context.Background()
		if err := h.addressChangeNotifier.NotifyNodeAddressChange(notifyCtx, nodeID); err != nil {
			h.logger.Warnw("failed to notify forward agents of node IP change",
				"error", err,
				"node_id", nodeID,
			)
		}
	})
}

// syncSubscriptionsOnConnect pushes all active subscriptions to the node when it connects.
func (h *NodeHubHandler) syncSubscriptionsOnConnect(nodeID uint) {
	if h.subscriptionSyncer == nil {
		return
	}

	goroutine.SafeGo(h.logger, "sync-subscriptions-on-node-connect", func() {
		ctx := context.Background()
		if err := h.subscriptionSyncer.SyncSubscriptionsOnNodeConnect(ctx, nodeID); err != nil {
			h.logger.Warnw("failed to sync subscriptions on node connect",
				"node_id", nodeID,
				"error", err,
			)
		}
	})
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

		// Real-time traffic limit check using cached quota.
		// Compare current usage against cached limit to detect over-limit immediately.
		// Note: Only node/hybrid subscriptions are checked here; forward subscriptions are handled
		// by Forward Agent traffic reporting flow.
		h.checkAndEnforceTrafficLimit(ctx, subID)
	}

}

// checkAndEnforceTrafficLimit performs real-time traffic limit check using cached quota.
// If the subscription exceeds its limit, it marks the subscription as suspended and triggers enforcement.
func (h *NodeHubHandler) checkAndEnforceTrafficLimit(ctx context.Context, subscriptionID uint) {
	// Skip if dependencies are not set
	if h.quotaCache == nil || h.usageReader == nil || h.trafficEnforcer == nil {
		return
	}

	// 1. Get quota from cache
	quota, err := h.quotaCache.GetQuota(ctx, subscriptionID)
	if err != nil {
		h.logger.Warnw("failed to get quota cache for traffic limit check",
			"subscription_id", subscriptionID,
			"error", err,
		)
		return
	}

	// 2. If null marker hit, skip DB lookup (anti-penetration)
	if quota != nil && quota.NotFound {
		return
	}

	// 3. If cache miss, try lazy loading from database
	if quota == nil {
		if h.quotaLoader == nil {
			// No loader configured, skip check
			return
		}
		quota, err = h.quotaLoader.LoadQuotaByID(ctx, subscriptionID)
		if err != nil {
			h.logger.Warnw("failed to load quota from database",
				"subscription_id", subscriptionID,
				"error", err,
			)
			return
		}
		if quota == nil {
			// Subscription/plan not found or not active, skip check
			return
		}
	}

	// 4. Check if already suspended
	if quota.Suspended {
		return
	}

	// 5. Check if plan type is node or hybrid (skip forward-only subscriptions)
	if quota.PlanType != "node" && quota.PlanType != "hybrid" {
		return
	}

	// 6. Get current period usage
	usage, err := h.usageReader.GetCurrentPeriodUsage(ctx, subscriptionID, quota.PeriodStart, quota.PeriodEnd)
	if err != nil {
		h.logger.Warnw("failed to get node subscription usage for traffic limit check",
			"subscription_id", subscriptionID,
			"error", err,
		)
		return
	}

	// 7. Compare usage against limit (skip unlimited quotas where Limit == 0)
	if quota.Limit > 0 && usage >= quota.Limit {
		h.logger.Debugw("node subscription exceeded traffic limit, triggering enforcement",
			"subscription_id", subscriptionID,
			"usage", usage,
			"limit", quota.Limit,
		)

		// Mark as suspended in cache to prevent duplicate enforcement
		if err := h.quotaCache.MarkSuspended(ctx, subscriptionID); err != nil {
			h.logger.Warnw("failed to mark subscription as suspended in cache",
				"subscription_id", subscriptionID,
				"error", err,
			)
		}

		// Async enforcement to avoid blocking traffic report flow
		sid := subscriptionID
		goroutine.SafeGo(h.logger, "enforce-node-traffic-limit", func() {
			enforceCtx := context.Background()
			if err := h.trafficEnforcer.CheckAndEnforceLimitForNode(enforceCtx, sid); err != nil {
				h.logger.Warnw("failed to enforce node traffic limit",
					"subscription_id", sid,
					"error", err,
				)
			}
		})
	}
}

// BroadcastAPIURLChangedRequest represents the request body for broadcasting API URL change to nodes.
type BroadcastAPIURLChangedRequest struct {
	NewURL string `json:"new_url" binding:"required,url" example:"https://new-api.example.com"`
	Reason string `json:"reason,omitempty" example:"server migration"`
}

// BroadcastAPIURLChangedResponse represents the response for API URL change broadcast to nodes.
type BroadcastAPIURLChangedResponse struct {
	NodesNotified int `json:"nodes_notified"`
	NodesOnline   int `json:"nodes_online"`
}

// BroadcastAPIURLChanged handles POST /nodes/broadcast-url-change
// Notifies connected node agents that the API URL has changed.
// Nodes should update their local configuration and reconnect to the new URL.
func (h *NodeHubHandler) BroadcastAPIURLChanged(c *gin.Context) {
	var req BroadcastAPIURLChangedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for broadcast API URL change to nodes",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	notified, online := h.hub.BroadcastNodeAPIURLChanged(req.NewURL, req.Reason)

	// Get operator info for audit logging
	var operatorID any = "unknown"
	if userID, exists := c.Get("user_id"); exists {
		operatorID = userID
	}

	h.logger.Infow("API URL change broadcast to nodes completed",
		"url_host", extractURLHost(req.NewURL),
		"reason", req.Reason,
		"nodes_notified", notified,
		"nodes_online", online,
		"operator_id", operatorID,
		"ip", c.ClientIP(),
	)

	utils.SuccessResponse(c, http.StatusOK, "API URL change broadcast to nodes completed", &BroadcastAPIURLChangedResponse{
		NodesNotified: notified,
		NodesOnline:   online,
	})
}

// NotifyAPIURLChangedRequest represents the request body for notifying a single node of API URL change.
type NotifyAPIURLChangedRequest struct {
	NewURL string `json:"new_url" binding:"required,url" example:"https://new-api.example.com"`
	Reason string `json:"reason,omitempty" example:"server migration"`
}

// NotifyAPIURLChangedResponse represents the response for single node API URL change notification.
type NotifyAPIURLChangedResponse struct {
	NodeID   string `json:"node_id"`
	Notified bool   `json:"notified"`
}

// NotifyAPIURLChanged handles POST /nodes/:id/url-change
// Notifies a specific connected node that the API URL has changed.
func (h *NodeHubHandler) NotifyAPIURLChanged(c *gin.Context) {
	nodeSID, err := utils.ParseSIDParam(c, "id", id.PrefixNode, "node")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req NotifyAPIURLChangedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for notify API URL change",
			"error", err,
			"node_sid", nodeSID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	// Get node by SID to retrieve internal ID
	n, err := h.nodeRepo.GetBySID(c.Request.Context(), nodeSID)
	if err != nil {
		h.logger.Errorw("failed to get node",
			"node_sid", nodeSID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponseWithError(c, err)
		return
	}
	if n == nil {
		utils.ErrorResponseWithError(c, errors.NewNotFoundError("node not found"))
		return
	}

	err = h.hub.NotifyNodeAPIURLChanged(n.ID(), req.NewURL, req.Reason)
	if err != nil {
		h.logger.Warnw("failed to notify node of API URL change",
			"error", err,
			"node_id", n.ID(),
			"node_sid", nodeSID,
			"url_host", extractURLHost(req.NewURL),
			"ip", c.ClientIP(),
		)
		utils.SuccessResponse(c, http.StatusOK, "node not connected or send failed", &NotifyAPIURLChangedResponse{
			NodeID:   nodeSID,
			Notified: false,
		})
		return
	}

	// Get operator info for audit logging
	var operatorID any = "unknown"
	if userID, exists := c.Get("user_id"); exists {
		operatorID = userID
	}

	h.logger.Infow("API URL change notification sent to node",
		"node_id", n.ID(),
		"node_sid", nodeSID,
		"url_host", extractURLHost(req.NewURL),
		"reason", req.Reason,
		"operator_id", operatorID,
		"ip", c.ClientIP(),
	)

	utils.SuccessResponse(c, http.StatusOK, "API URL change notification sent", &NotifyAPIURLChangedResponse{
		NodeID:   nodeSID,
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
