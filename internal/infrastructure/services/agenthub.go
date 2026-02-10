// Package services provides infrastructure services.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/orris-inc/orris/internal/infrastructure/pubsub"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	dto "github.com/orris-inc/orris/internal/shared/hubprotocol/forward"
	nodedto "github.com/orris-inc/orris/internal/shared/hubprotocol/node"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AgentHubConfig holds configuration for AgentHub.
type AgentHubConfig struct {
	// NodeStatusTimeoutMs is the timeout in milliseconds for node status updates.
	// If a node doesn't report status within this duration, it's considered offline.
	// Default: 5000 (5 seconds)
	NodeStatusTimeoutMs int64
}

// AgentHub manages WebSocket connections for forward agents and node agents.
type AgentHub struct {
	// Forward agent connections: map[AgentID]*AgentHubConn
	agents   map[uint]*AgentHubConn
	agentsMu sync.RWMutex

	// Node agent connections: map[NodeID]*NodeHubConn
	nodes   map[uint]*NodeHubConn
	nodesMu sync.RWMutex

	// Status handler for forward agent
	statusHandler   StatusHandler
	statusHandlerMu sync.RWMutex

	// Status handler for node agent
	nodeStatusHandler   StatusHandler
	nodeStatusHandlerMu sync.RWMutex

	// Message handlers for specific message types (domain extensions)
	messageHandlers   []MessageHandler
	messageHandlersMu sync.RWMutex

	// Callbacks for forward agents
	onAgentOnline  func(agentID uint)
	onAgentOffline func(agentID uint)

	// Callbacks for node agents
	onNodeOnline  func(nodeID uint)
	onNodeOffline func(nodeID uint)

	// Configuration
	nodeStatusTimeout time.Duration

	// Shutdown signal
	done     chan struct{}
	shutdown atomic.Bool

	logger logger.Interface

	// Optional event bus for cross-instance command relay
	eventBus   pubsub.HubCommandPublisher
	eventBusMu sync.RWMutex
}

// AgentHubConn represents a forward agent WebSocket connection.
type AgentHubConn struct {
	AgentID     uint
	Conn        *websocket.Conn
	Send        chan *dto.HubMessage
	LastSeen    time.Time
	ConnectedAt time.Time
	closed      atomic.Bool // Indicates if the connection has been closed
}

// TrySend attempts to send a message to the agent.
// Returns false if the channel is closed or full.
// Uses recover to handle the race condition between closed check and send.
func (c *AgentHubConn) TrySend(msg *dto.HubMessage) (sent bool) {
	if c.closed.Load() {
		return false
	}

	// Recover from panic if channel is closed between the check and send
	defer func() {
		if r := recover(); r != nil {
			sent = false
		}
	}()

	select {
	case c.Send <- msg:
		return true
	default:
		return false
	}
}

// Close marks the connection as closed and closes the send channel.
// Safe to call multiple times.
func (c *AgentHubConn) Close() {
	if c.closed.CompareAndSwap(false, true) {
		close(c.Send)
	}
}

// NodeHubConn represents a node agent WebSocket connection.
type NodeHubConn struct {
	NodeID      uint
	Conn        *websocket.Conn
	Send        chan []byte // Generic byte channel for node messages
	LastSeen    time.Time
	ConnectedAt time.Time
	closed      atomic.Bool
}

// TrySend attempts to send a message to the node.
// Returns false if the channel is closed or full.
func (c *NodeHubConn) TrySend(msg []byte) (sent bool) {
	if c.closed.Load() {
		return false
	}

	defer func() {
		if r := recover(); r != nil {
			sent = false
		}
	}()

	select {
	case c.Send <- msg:
		return true
	default:
		return false
	}
}

// Close marks the connection as closed and closes the send channel.
// Safe to call multiple times.
func (c *NodeHubConn) Close() {
	if c.closed.CompareAndSwap(false, true) {
		close(c.Send)
	}
}

// NewAgentHub creates a new AgentHub instance.
func NewAgentHub(log logger.Interface, config *AgentHubConfig) *AgentHub {
	nodeTimeout := 5 * time.Second // default 5 seconds
	if config != nil && config.NodeStatusTimeoutMs > 0 {
		nodeTimeout = time.Duration(config.NodeStatusTimeoutMs) * time.Millisecond
	}

	h := &AgentHub{
		agents:            make(map[uint]*AgentHubConn),
		nodes:             make(map[uint]*NodeHubConn),
		messageHandlers:   make([]MessageHandler, 0),
		nodeStatusTimeout: nodeTimeout,
		done:              make(chan struct{}),
		logger:            log,
	}

	// Start background timeout checker
	goroutine.SafeGo(log, "agenthub-node-timeout-checker", func() {
		h.nodeTimeoutChecker()
	})

	return h
}

// Shutdown stops the AgentHub and releases resources.
// Safe to call multiple times.
func (h *AgentHub) Shutdown() {
	if !h.shutdown.CompareAndSwap(false, true) {
		return // Already shutdown
	}

	close(h.done)
}

// SetEventBus sets the event bus for cross-instance command relay.
// When set, commands to non-local agents will be published to Redis
// for delivery by other instances.
func (h *AgentHub) SetEventBus(eventBus pubsub.HubCommandPublisher) {
	h.eventBusMu.Lock()
	defer h.eventBusMu.Unlock()
	h.eventBus = eventBus
}

// getEventBus safely retrieves the event bus with read lock.
func (h *AgentHub) getEventBus() pubsub.HubCommandPublisher {
	h.eventBusMu.RLock()
	defer h.eventBusMu.RUnlock()
	return h.eventBus
}

// nodeTimeoutChecker periodically checks for nodes that haven't reported status.
func (h *AgentHub) nodeTimeoutChecker() {
	// Check interval: half of timeout duration, minimum 1 second
	interval := h.nodeStatusTimeout / 2
	if interval < time.Second {
		interval = time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.done:
			return
		case <-ticker.C:
			h.checkNodeTimeouts()
		}
	}
}

// checkNodeTimeouts checks all nodes and disconnects those that have timed out.
func (h *AgentHub) checkNodeTimeouts() {
	now := biztime.NowUTC()

	// Collect timed out nodes under read lock
	h.nodesMu.RLock()
	var timedOutNodes []uint
	for nodeID, conn := range h.nodes {
		if now.Sub(conn.LastSeen) > h.nodeStatusTimeout {
			timedOutNodes = append(timedOutNodes, nodeID)
		}
	}
	h.nodesMu.RUnlock()

	// Disconnect timed out nodes
	for _, nodeID := range timedOutNodes {
		h.logger.Warnw("node status timeout, disconnecting",
			"node_id", nodeID,
			"timeout", h.nodeStatusTimeout,
		)
		h.disconnectNode(nodeID)
	}
}

// disconnectNode disconnects a node due to timeout.
func (h *AgentHub) disconnectNode(nodeID uint) {
	h.nodesMu.Lock()
	conn, ok := h.nodes[nodeID]
	if ok {
		conn.Close()
		conn.Conn.Close()
		delete(h.nodes, nodeID)
	}
	h.nodesMu.Unlock()

	if ok {
		h.logger.Infow("node agent disconnected due to timeout",
			"node_id", nodeID,
		)

		if h.onNodeOffline != nil {
			goroutine.SafeGo(h.logger, "agenthub-on-node-offline-timeout", func() {
				h.onNodeOffline(nodeID)
			})
		}
	}
}

// RegisterStatusHandler registers a status handler for forward agent.
func (h *AgentHub) RegisterStatusHandler(handler StatusHandler) {
	h.statusHandlerMu.Lock()
	defer h.statusHandlerMu.Unlock()
	h.statusHandler = handler
	h.logger.Infow("forward agent status handler registered")
}

// RegisterMessageHandler registers a message handler for domain-specific messages.
func (h *AgentHub) RegisterMessageHandler(handler MessageHandler) {
	h.messageHandlersMu.Lock()
	defer h.messageHandlersMu.Unlock()
	h.messageHandlers = append(h.messageHandlers, handler)
	h.logger.Infow("message handler registered", "handler", handler.String())
}

// RouteAgentMessage routes a message from an agent to registered handlers.
func (h *AgentHub) RouteAgentMessage(agentID uint, msgType string, data any) bool {
	h.messageHandlersMu.RLock()
	defer h.messageHandlersMu.RUnlock()

	for _, handler := range h.messageHandlers {
		if handler.HandleMessage(agentID, msgType, data) {
			return true
		}
	}
	return false
}

// SendMessageToAgent sends a generic message to an agent.
func (h *AgentHub) SendMessageToAgent(agentID uint, msg *dto.HubMessage) error {
	h.agentsMu.RLock()
	agentConn, ok := h.agents[agentID]
	h.agentsMu.RUnlock()

	if !ok {
		return ErrAgentNotConnected
	}

	if !agentConn.TrySend(msg) {
		return ErrSendChannelFull
	}
	return nil
}

// SetOnAgentOnline sets the callback for agent online events.
func (h *AgentHub) SetOnAgentOnline(fn func(agentID uint)) {
	h.onAgentOnline = fn
}

// SetOnAgentOffline sets the callback for agent offline events.
func (h *AgentHub) SetOnAgentOffline(fn func(agentID uint)) {
	h.onAgentOffline = fn
}

// RegisterAgent registers a forward agent WebSocket connection.
func (h *AgentHub) RegisterAgent(agentID uint, conn *websocket.Conn) *AgentHubConn {
	h.agentsMu.Lock()
	defer h.agentsMu.Unlock()

	// Close existing connection if any
	if existing, ok := h.agents[agentID]; ok {
		existing.Close() // Use Close() to safely close the channel
		existing.Conn.Close()
	}

	agentConn := &AgentHubConn{
		AgentID:     agentID,
		Conn:        conn,
		Send:        make(chan *dto.HubMessage, 256),
		LastSeen:    biztime.NowUTC(),
		ConnectedAt: biztime.NowUTC(),
	}
	h.agents[agentID] = agentConn

	h.logger.Debugw("forward agent connected via websocket",
		"agent_id", agentID,
	)

	if h.onAgentOnline != nil {
		goroutine.SafeGo(h.logger, "agenthub-on-agent-online", func() {
			h.onAgentOnline(agentID)
		})
	}

	return agentConn
}

// UnregisterAgent removes an agent connection.
func (h *AgentHub) UnregisterAgent(agentID uint) {
	h.agentsMu.Lock()
	defer h.agentsMu.Unlock()

	if conn, ok := h.agents[agentID]; ok {
		conn.Close() // Use Close() to safely close the channel
		delete(h.agents, agentID)

		h.logger.Infow("forward agent disconnected",
			"agent_id", agentID,
		)

		if h.onAgentOffline != nil {
			goroutine.SafeGo(h.logger, "agenthub-on-agent-offline", func() {
				h.onAgentOffline(agentID)
			})
		}
	}
}

// HandleAgentStatus handles status update from an agent.
func (h *AgentHub) HandleAgentStatus(agentID uint, data any) {
	// Update last seen
	h.agentsMu.Lock()
	if conn, ok := h.agents[agentID]; ok {
		conn.LastSeen = biztime.NowUTC()
	}
	h.agentsMu.Unlock()

	// Call registered status handler
	h.statusHandlerMu.RLock()
	handler := h.statusHandler
	h.statusHandlerMu.RUnlock()

	if handler != nil {
		handler.HandleStatus(agentID, data)
	}
}

// SendCommandToAgent sends a command to a specific agent.
// If the agent is not connected to this instance but an event bus is configured,
// the command is published to Redis for cross-instance delivery.
func (h *AgentHub) SendCommandToAgent(agentID uint, cmd *dto.CommandData) error {
	h.agentsMu.RLock()
	agentConn, ok := h.agents[agentID]
	h.agentsMu.RUnlock()

	if ok {
		// Local delivery
		msg := &dto.HubMessage{
			Type:      dto.MsgTypeCommand,
			AgentID:   "", // Agent already knows its own ID; this field is primarily for logging/debug
			Timestamp: biztime.NowUTC().Unix(),
			Data:      cmd,
		}
		if !agentConn.TrySend(msg) {
			return ErrSendChannelFull
		}
		return nil
	}

	// Not local - publish to Redis for other instances
	eb := h.getEventBus()
	if eb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return eb.PublishAgentCommand(ctx, agentID, cmd)
	}

	return ErrAgentNotConnected
}

// IsAgentOnline checks if an agent is connected.
func (h *AgentHub) IsAgentOnline(agentID uint) bool {
	h.agentsMu.RLock()
	defer h.agentsMu.RUnlock()
	_, ok := h.agents[agentID]
	return ok
}

// GetOnlineAgents returns list of online agent IDs.
func (h *AgentHub) GetOnlineAgents() []uint {
	h.agentsMu.RLock()
	defer h.agentsMu.RUnlock()

	ids := make([]uint, 0, len(h.agents))
	for id := range h.agents {
		ids = append(ids, id)
	}
	return ids
}

// GetAgentSendChan returns the send channel for an agent connection.
func (h *AgentHub) GetAgentSendChan(agentID uint) chan *dto.HubMessage {
	h.agentsMu.RLock()
	defer h.agentsMu.RUnlock()

	if conn, ok := h.agents[agentID]; ok {
		return conn.Send
	}
	return nil
}

// StatusHandler defines the interface for handling agent status updates.
type StatusHandler interface {
	// HandleStatus processes status update from an agent.
	HandleStatus(agentID uint, data any)
}

// MessageHandler defines the interface for handling specific message types.
// Each domain can register handlers for message types they care about.
type MessageHandler interface {
	// String returns the handler name for logging purposes (implements fmt.Stringer).
	String() string
	// HandleMessage processes a message from a forward agent.
	// Returns true if the message was handled, false otherwise.
	HandleMessage(agentID uint, msgType string, data any) bool
}

// ============================================================================
// Node Agent Methods
// ============================================================================

// RegisterNodeStatusHandler registers a status handler for node agent.
func (h *AgentHub) RegisterNodeStatusHandler(handler StatusHandler) {
	h.nodeStatusHandlerMu.Lock()
	defer h.nodeStatusHandlerMu.Unlock()
	h.nodeStatusHandler = handler
	h.logger.Infow("node agent status handler registered")
}

// SetOnNodeOnline sets the callback for node online events.
func (h *AgentHub) SetOnNodeOnline(fn func(nodeID uint)) {
	h.onNodeOnline = fn
}

// SetOnNodeOffline sets the callback for node offline events.
func (h *AgentHub) SetOnNodeOffline(fn func(nodeID uint)) {
	h.onNodeOffline = fn
}

// RegisterNodeAgent registers a node agent WebSocket connection.
func (h *AgentHub) RegisterNodeAgent(nodeID uint, conn *websocket.Conn) *NodeHubConn {
	h.nodesMu.Lock()
	defer h.nodesMu.Unlock()

	// Close existing connection if any
	if existing, ok := h.nodes[nodeID]; ok {
		existing.Close()
		existing.Conn.Close()
	}

	nodeConn := &NodeHubConn{
		NodeID:      nodeID,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		LastSeen:    biztime.NowUTC(),
		ConnectedAt: biztime.NowUTC(),
	}
	h.nodes[nodeID] = nodeConn

	h.logger.Infow("node agent connected via websocket",
		"node_id", nodeID,
	)

	if h.onNodeOnline != nil {
		goroutine.SafeGo(h.logger, "agenthub-on-node-online", func() {
			h.onNodeOnline(nodeID)
		})
	}

	return nodeConn
}

// UnregisterNodeAgent removes a node agent connection.
func (h *AgentHub) UnregisterNodeAgent(nodeID uint) {
	h.nodesMu.Lock()
	defer h.nodesMu.Unlock()

	if conn, ok := h.nodes[nodeID]; ok {
		conn.Close()
		delete(h.nodes, nodeID)

		h.logger.Infow("node agent disconnected",
			"node_id", nodeID,
		)

		if h.onNodeOffline != nil {
			goroutine.SafeGo(h.logger, "agenthub-on-node-offline", func() {
				h.onNodeOffline(nodeID)
			})
		}
	}
}

// HandleNodeStatus handles status update from a node agent.
func (h *AgentHub) HandleNodeStatus(nodeID uint, data any) {
	// Update last seen
	h.nodesMu.Lock()
	if conn, ok := h.nodes[nodeID]; ok {
		conn.LastSeen = biztime.NowUTC()
	}
	h.nodesMu.Unlock()

	// Call registered status handler
	h.nodeStatusHandlerMu.RLock()
	handler := h.nodeStatusHandler
	h.nodeStatusHandlerMu.RUnlock()

	if handler != nil {
		handler.HandleStatus(nodeID, data)
	}
}

// IsNodeOnline checks if a node agent is connected.
func (h *AgentHub) IsNodeOnline(nodeID uint) bool {
	h.nodesMu.RLock()
	defer h.nodesMu.RUnlock()
	_, ok := h.nodes[nodeID]
	return ok
}

// GetOnlineNodes returns list of online node IDs.
func (h *AgentHub) GetOnlineNodes() []uint {
	h.nodesMu.RLock()
	defer h.nodesMu.RUnlock()

	ids := make([]uint, 0, len(h.nodes))
	for id := range h.nodes {
		ids = append(ids, id)
	}
	return ids
}

// GetNodeSendChan returns the send channel for a node connection.
func (h *AgentHub) GetNodeSendChan(nodeID uint) chan []byte {
	h.nodesMu.RLock()
	defer h.nodesMu.RUnlock()

	if conn, ok := h.nodes[nodeID]; ok {
		return conn.Send
	}
	return nil
}

// SendMessageToNode sends a message to a specific node.
func (h *AgentHub) SendMessageToNode(nodeID uint, msg []byte) error {
	h.nodesMu.RLock()
	nodeConn, ok := h.nodes[nodeID]
	h.nodesMu.RUnlock()

	if !ok {
		return ErrNodeNotConnected
	}

	if !nodeConn.TrySend(msg) {
		return ErrSendChannelFull
	}
	return nil
}

// SendCommandToNode sends a command to a specific node agent.
// If the node is not connected to this instance but an event bus is configured,
// the command is published to Redis for cross-instance delivery.
func (h *AgentHub) SendCommandToNode(nodeID uint, cmd *nodedto.NodeCommandData) error {
	h.nodesMu.RLock()
	nodeConn, ok := h.nodes[nodeID]
	h.nodesMu.RUnlock()

	if ok {
		// Local delivery
		msg := &nodedto.NodeHubMessage{
			Type:      nodedto.NodeMsgTypeCommand,
			Timestamp: biztime.NowUTC().Unix(),
			Data:      cmd,
		}
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		if !nodeConn.TrySend(msgBytes) {
			return ErrSendChannelFull
		}
		return nil
	}

	// Not local - publish to Redis for other instances
	eb := h.getEventBus()
	if eb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return eb.PublishNodeCommand(ctx, nodeID, cmd)
	}

	return ErrNodeNotConnected
}

// HandleRemoteAgentCommand handles an agent command received from Redis PubSub.
// Only delivers if the agent is connected to this instance.
func (h *AgentHub) HandleRemoteAgentCommand(agentID uint, cmd *dto.CommandData) {
	h.agentsMu.RLock()
	agentConn, ok := h.agents[agentID]
	h.agentsMu.RUnlock()

	if !ok {
		return // Agent not on this instance
	}

	msg := &dto.HubMessage{
		Type:      dto.MsgTypeCommand,
		Timestamp: biztime.NowUTC().Unix(),
		Data:      cmd,
	}

	if !agentConn.TrySend(msg) {
		h.logger.Warnw("failed to deliver remote agent command, channel full",
			"agent_id", agentID,
			"action", cmd.Action,
		)
	}
}

// HandleRemoteNodeCommand handles a node command received from Redis PubSub.
// Only delivers if the node is connected to this instance.
func (h *AgentHub) HandleRemoteNodeCommand(nodeID uint, cmd *nodedto.NodeCommandData) {
	h.nodesMu.RLock()
	nodeConn, ok := h.nodes[nodeID]
	h.nodesMu.RUnlock()

	if !ok {
		return // Node not on this instance
	}

	msg := &nodedto.NodeHubMessage{
		Type:      nodedto.NodeMsgTypeCommand,
		Timestamp: biztime.NowUTC().Unix(),
		Data:      cmd,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		h.logger.Errorw("failed to marshal remote node command",
			"node_id", nodeID,
			"error", err,
		)
		return
	}

	if !nodeConn.TrySend(msgBytes) {
		h.logger.Warnw("failed to deliver remote node command, channel full",
			"node_id", nodeID,
			"action", cmd.Action,
		)
	}
}

// extractURLHost extracts the host from a URL for safe logging (avoids leaking credentials).
func extractURLHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "<invalid-url>"
	}
	return parsed.Host
}

// BroadcastAPIURLChanged notifies all connected agents that the API URL has changed.
// Agents should update their local configuration and reconnect to the new URL.
// Returns (agents_notified, agents_online) to ensure consistency.
func (h *AgentHub) BroadcastAPIURLChanged(newURL, reason string) (notified int, online int) {
	h.agentsMu.RLock()
	defer h.agentsMu.RUnlock()

	payload := &dto.APIURLChangedPayload{
		NewURL: newURL,
		Reason: reason,
	}

	cmd := &dto.CommandData{
		CommandID: fmt.Sprintf("api_url_changed_%s", uuid.NewString()),
		Action:    dto.CmdActionAPIURLChanged,
		Payload:   payload,
	}

	msg := &dto.HubMessage{
		Type:      dto.MsgTypeCommand,
		Timestamp: biztime.NowUTC().Unix(),
		Data:      cmd,
	}

	urlHost := extractURLHost(newURL)
	online = len(h.agents)
	for agentID, conn := range h.agents {
		if conn.TrySend(msg) {
			notified++
			h.logger.Infow("sent API URL change notification to agent",
				"agent_id", agentID,
				"url_host", urlHost,
			)
		} else {
			h.logger.Warnw("failed to send API URL change notification to agent",
				"agent_id", agentID,
				"url_host", urlHost,
			)
		}
	}

	h.logger.Infow("broadcasted API URL change to agents",
		"url_host", urlHost,
		"reason", reason,
		"agents_notified", notified,
		"agents_total", online,
	)

	return notified, online
}

// NotifyAgentAPIURLChanged notifies a specific agent that the API URL has changed.
func (h *AgentHub) NotifyAgentAPIURLChanged(agentID uint, newURL, reason string) error {
	payload := &dto.APIURLChangedPayload{
		NewURL: newURL,
		Reason: reason,
	}

	cmd := &dto.CommandData{
		CommandID: fmt.Sprintf("api_url_changed_%s", uuid.NewString()),
		Action:    dto.CmdActionAPIURLChanged,
		Payload:   payload,
	}

	return h.SendCommandToAgent(agentID, cmd)
}

// BroadcastNodeAPIURLChanged notifies all connected node agents that the API URL has changed.
// Node agents should update their local configuration and reconnect to the new URL.
// Returns (nodes_notified, nodes_online) to ensure consistency.
func (h *AgentHub) BroadcastNodeAPIURLChanged(newURL, reason string) (notified int, online int) {
	h.nodesMu.RLock()
	defer h.nodesMu.RUnlock()

	payload := &nodedto.NodeAPIURLChangedPayload{
		NewURL: newURL,
		Reason: reason,
	}

	cmd := &nodedto.NodeCommandData{
		CommandID: fmt.Sprintf("api_url_changed_%s", uuid.NewString()),
		Action:    nodedto.NodeCmdActionAPIURLChanged,
		Payload:   payload,
	}

	msg := &nodedto.NodeHubMessage{
		Type:      nodedto.NodeMsgTypeCommand,
		Timestamp: biztime.NowUTC().Unix(),
		Data:      cmd,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		h.logger.Errorw("failed to marshal node API URL change message",
			"error", err,
		)
		return 0, len(h.nodes)
	}

	urlHost := extractURLHost(newURL)
	online = len(h.nodes)
	for nodeID, conn := range h.nodes {
		if conn.TrySend(msgBytes) {
			notified++
			h.logger.Infow("sent API URL change notification to node",
				"node_id", nodeID,
				"url_host", urlHost,
			)
		} else {
			h.logger.Warnw("failed to send API URL change notification to node",
				"node_id", nodeID,
				"url_host", urlHost,
			)
		}
	}

	h.logger.Infow("broadcasted API URL change to nodes",
		"url_host", urlHost,
		"reason", reason,
		"nodes_notified", notified,
		"nodes_total", online,
	)

	return notified, online
}

// NotifyNodeAPIURLChanged notifies a specific node that the API URL has changed.
func (h *AgentHub) NotifyNodeAPIURLChanged(nodeID uint, newURL, reason string) error {
	payload := &nodedto.NodeAPIURLChangedPayload{
		NewURL: newURL,
		Reason: reason,
	}

	cmd := &nodedto.NodeCommandData{
		CommandID: fmt.Sprintf("api_url_changed_%s", uuid.NewString()),
		Action:    nodedto.NodeCmdActionAPIURLChanged,
		Payload:   payload,
	}

	return h.SendCommandToNode(nodeID, cmd)
}

// BroadcastAllAPIURLChanged notifies all connected agents (forward + node) that the API URL has changed.
// Returns (forward_notified, forward_online, node_notified, node_online).
func (h *AgentHub) BroadcastAllAPIURLChanged(newURL, reason string) (forwardNotified, forwardOnline, nodeNotified, nodeOnline int) {
	forwardNotified, forwardOnline = h.BroadcastAPIURLChanged(newURL, reason)
	nodeNotified, nodeOnline = h.BroadcastNodeAPIURLChanged(newURL, reason)
	return
}

// HubErrors defines agent hub related errors.
var (
	ErrAgentNotConnected = &HubError{Code: "AGENT_NOT_CONNECTED", Message: "agent not connected"}
	ErrNodeNotConnected  = &HubError{Code: "NODE_NOT_CONNECTED", Message: "node not connected"}
	ErrSendChannelFull   = &HubError{Code: "SEND_CHANNEL_FULL", Message: "send channel full"}
)

// HubError represents an agent hub error.
type HubError struct {
	Code    string
	Message string
}

// Error implements the error interface.
func (e *HubError) Error() string {
	return e.Message
}
