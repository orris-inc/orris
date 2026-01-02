// Package services provides infrastructure services.
package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AgentEventType represents the type of agent SSE event.
// Agents include both Node agents and Forward agents.
type AgentEventType string

// Node agent event types for SSE.
const (
	NodeEventOnline  AgentEventType = "node:online"
	NodeEventOffline AgentEventType = "node:offline"
	NodeEventStatus  AgentEventType = "node:status"
	NodeEventUpdated AgentEventType = "node:updated"
)

// Forward agent event types for SSE.
const (
	ForwardAgentEventOnline  AgentEventType = "agent:online"
	ForwardAgentEventOffline AgentEventType = "agent:offline"
	ForwardAgentEventStatus  AgentEventType = "agent:status"
	ForwardAgentEventUpdated AgentEventType = "agent:updated"
)

// AgentEvent represents an SSE event for agent status updates.
// Used for both Node agents and Forward agents.
type AgentEvent struct {
	Type      AgentEventType `json:"type"`
	AgentID   string         `json:"agentId"`
	AgentName string         `json:"agentName,omitempty"`
	Timestamp int64          `json:"timestamp"`
	Data      any            `json:"data,omitempty"`
}

// SSEConn represents an SSE connection from admin frontend.
type SSEConn struct {
	ID           string
	UserID       uint
	Send         chan []byte
	NodeFilters  map[string]bool // nil means subscribe to all nodes
	AgentFilters map[string]bool // nil means subscribe to all agents
	ConnectedAt  time.Time
	closed       atomic.Bool
}

// TrySend attempts to send data to the SSE connection.
// Returns false if the channel is closed or full.
func (c *SSEConn) TrySend(data []byte) (sent bool) {
	if c.closed.Load() {
		return false
	}

	defer func() {
		if r := recover(); r != nil {
			sent = false
		}
	}()

	select {
	case c.Send <- data:
		return true
	default:
		return false
	}
}

// Close marks the connection as closed and closes the send channel.
func (c *SSEConn) Close() {
	if c.closed.CompareAndSwap(false, true) {
		close(c.Send)
	}
}

// ShouldReceive checks if this connection should receive events for the given node.
func (c *SSEConn) ShouldReceive(nodeSID string) bool {
	if c.NodeFilters == nil {
		return true // No filter, receive all
	}
	return c.NodeFilters[nodeSID]
}

// ShouldReceiveAgent checks if this connection should receive events for the given agent.
func (c *SSEConn) ShouldReceiveAgent(agentSID string) bool {
	if c.AgentFilters == nil {
		return true // No filter, receive all
	}
	return c.AgentFilters[agentSID]
}

// AdminHub manages SSE connections from admin frontend.
type AdminHub struct {
	// SSE connections: map[connID]*SSEConn
	conns   map[string]*SSEConn
	connsMu sync.RWMutex

	// Connections per user for rate limiting
	userConns   map[uint]int
	userConnsMu sync.RWMutex

	// Status throttling: map[nodeSID]lastPushTime
	statusThrottle   map[string]time.Time
	statusThrottleMu sync.RWMutex

	// Agent status throttling: map[agentSID]lastPushTime
	agentThrottle   map[string]time.Time
	agentThrottleMu sync.RWMutex

	// Configuration
	maxConnsPerUser  int
	statusThrottleMs int64

	// Shutdown signal
	done     chan struct{}
	shutdown atomic.Bool

	logger logger.Interface
}

// AdminHubConfig holds configuration for AdminHub.
type AdminHubConfig struct {
	MaxConnsPerUser  int   // Max SSE connections per user (default: 5)
	StatusThrottleMs int64 // Throttle interval for status events in ms (default: 5000)
}

// NewAdminHub creates a new AdminHub instance.
func NewAdminHub(log logger.Interface, config *AdminHubConfig) *AdminHub {
	maxConns := 5
	throttleMs := int64(5000)

	if config != nil {
		if config.MaxConnsPerUser > 0 {
			maxConns = config.MaxConnsPerUser
		}
		if config.StatusThrottleMs > 0 {
			throttleMs = config.StatusThrottleMs
		}
	}

	h := &AdminHub{
		conns:            make(map[string]*SSEConn),
		userConns:        make(map[uint]int),
		statusThrottle:   make(map[string]time.Time),
		agentThrottle:    make(map[string]time.Time),
		maxConnsPerUser:  maxConns,
		statusThrottleMs: throttleMs,
		done:             make(chan struct{}),
		logger:           log,
	}

	// Start background cleanup goroutine
	go h.cleanupLoop()

	return h
}

// cleanupLoop periodically cleans up the throttle cache.
func (h *AdminHub) cleanupLoop() {
	// Cleanup interval: 2x throttle duration, minimum 10 seconds
	interval := time.Duration(h.statusThrottleMs*2) * time.Millisecond
	if interval < 10*time.Second {
		interval = 10 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.done:
			return
		case <-ticker.C:
			h.CleanupThrottleCache()
		}
	}
}

// Shutdown stops the AdminHub and releases resources.
// Safe to call multiple times.
func (h *AdminHub) Shutdown() {
	if !h.shutdown.CompareAndSwap(false, true) {
		return // Already shutdown
	}

	close(h.done)

	// Close all connections
	h.connsMu.Lock()
	for _, conn := range h.conns {
		conn.Close()
	}
	h.conns = make(map[string]*SSEConn)
	h.connsMu.Unlock()
}

// RegisterConn registers a new SSE connection for node events.
// Returns the connection or nil if max connections exceeded or hub is shutdown.
func (h *AdminHub) RegisterConn(connID string, userID uint, nodeFilters []string) *SSEConn {
	return h.RegisterConnWithFilters(connID, userID, nodeFilters, nil)
}

// RegisterConnWithFilters registers a new SSE connection with both node and agent filters.
// Returns the connection or nil if max connections exceeded or hub is shutdown.
func (h *AdminHub) RegisterConnWithFilters(connID string, userID uint, nodeFilters, agentFilters []string) *SSEConn {
	// Check if shutdown
	if h.shutdown.Load() {
		return nil
	}

	// Build node filter map
	var nodeFilterMap map[string]bool
	if len(nodeFilters) > 0 {
		nodeFilterMap = make(map[string]bool, len(nodeFilters))
		for _, sid := range nodeFilters {
			nodeFilterMap[sid] = true
		}
	}

	// Build agent filter map
	var agentFilterMap map[string]bool
	if len(agentFilters) > 0 {
		agentFilterMap = make(map[string]bool, len(agentFilters))
		for _, sid := range agentFilters {
			agentFilterMap[sid] = true
		}
	}

	conn := &SSEConn{
		ID:           connID,
		UserID:       userID,
		Send:         make(chan []byte, 64),
		NodeFilters:  nodeFilterMap,
		AgentFilters: agentFilterMap,
		ConnectedAt:  biztime.NowUTC(),
	}

	// IMPORTANT: Always acquire locks in consistent order (connsMu -> userConnsMu)
	// to prevent deadlock with UnregisterConn
	h.connsMu.Lock()
	defer h.connsMu.Unlock()

	h.userConnsMu.Lock()
	defer h.userConnsMu.Unlock()

	// Check user connection limit
	if h.userConns[userID] >= h.maxConnsPerUser {
		h.logger.Warnw("SSE connection limit exceeded",
			"user_id", userID,
			"limit", h.maxConnsPerUser,
		)
		return nil
	}

	h.conns[connID] = conn

	// Increment count only after successful registration
	h.userConns[userID]++

	h.logger.Infow("SSE connection registered",
		"conn_id", connID,
		"user_id", userID,
		"node_filters", nodeFilters,
		"agent_filters", agentFilters,
	)

	return conn
}

// UnregisterConn removes an SSE connection.
func (h *AdminHub) UnregisterConn(connID string) {
	// IMPORTANT: Always acquire locks in consistent order (connsMu -> userConnsMu)
	// to prevent deadlock with RegisterConn
	h.connsMu.Lock()
	h.userConnsMu.Lock()

	conn, ok := h.conns[connID]
	if ok {
		delete(h.conns, connID)
		if h.userConns[conn.UserID] > 0 {
			h.userConns[conn.UserID]--
		}
	}

	h.userConnsMu.Unlock()
	h.connsMu.Unlock()

	if ok {
		conn.Close()

		h.logger.Infow("SSE connection unregistered",
			"conn_id", connID,
			"user_id", conn.UserID,
		)
	}
}

// Broadcast sends a node event to all matching SSE connections.
func (h *AdminHub) Broadcast(event *AgentEvent) {
	// Check throttling for status events
	if event.Type == NodeEventStatus {
		if !h.shouldPushStatus(event.AgentID) {
			return
		}
	}

	data, err := h.formatSSEEvent(event)
	if err != nil {
		h.logger.Errorw("failed to format SSE event",
			"event_type", event.Type,
			"error", err,
		)
		return
	}

	h.connsMu.RLock()
	defer h.connsMu.RUnlock()

	for _, conn := range h.conns {
		if conn.ShouldReceive(event.AgentID) {
			if !conn.TrySend(data) {
				h.logger.Warnw("failed to send SSE event, channel full",
					"conn_id", conn.ID,
					"event_type", event.Type,
				)
			}
		}
	}
}

// BroadcastNodeOnline broadcasts a node online event.
func (h *AdminHub) BroadcastNodeOnline(nodeSID, nodeName string) {
	h.Broadcast(&AgentEvent{
		Type:      NodeEventOnline,
		AgentID:   nodeSID,
		AgentName: nodeName,
		Timestamp: biztime.NowUTC().Unix(),
	})
}

// BroadcastNodeOffline broadcasts a node offline event.
func (h *AdminHub) BroadcastNodeOffline(nodeSID, nodeName string) {
	h.Broadcast(&AgentEvent{
		Type:      NodeEventOffline,
		AgentID:   nodeSID,
		AgentName: nodeName,
		Timestamp: biztime.NowUTC().Unix(),
	})
}

// BroadcastNodeStatus broadcasts a node status update event.
func (h *AdminHub) BroadcastNodeStatus(nodeSID string, status any) {
	h.Broadcast(&AgentEvent{
		Type:      NodeEventStatus,
		AgentID:   nodeSID,
		Timestamp: biztime.NowUTC().Unix(),
		Data:      status,
	})
}

// BroadcastNodeUpdated broadcasts a node updated event.
func (h *AdminHub) BroadcastNodeUpdated(nodeSID string, changes any) {
	h.Broadcast(&AgentEvent{
		Type:      NodeEventUpdated,
		AgentID:   nodeSID,
		Timestamp: biztime.NowUTC().Unix(),
		Data:      changes,
	})
}

// BroadcastForwardAgentOnline broadcasts a forward agent online event.
func (h *AdminHub) BroadcastForwardAgentOnline(agentSID, agentName string) {
	h.BroadcastForwardAgent(&AgentEvent{
		Type:      ForwardAgentEventOnline,
		AgentID:   agentSID,
		AgentName: agentName,
		Timestamp: biztime.NowUTC().Unix(),
	})
}

// BroadcastForwardAgentOffline broadcasts a forward agent offline event.
func (h *AdminHub) BroadcastForwardAgentOffline(agentSID, agentName string) {
	h.BroadcastForwardAgent(&AgentEvent{
		Type:      ForwardAgentEventOffline,
		AgentID:   agentSID,
		AgentName: agentName,
		Timestamp: biztime.NowUTC().Unix(),
	})
}

// BroadcastForwardAgentStatus broadcasts a forward agent status update event.
func (h *AdminHub) BroadcastForwardAgentStatus(agentSID string, status any) {
	h.BroadcastForwardAgent(&AgentEvent{
		Type:      ForwardAgentEventStatus,
		AgentID:   agentSID,
		Timestamp: biztime.NowUTC().Unix(),
		Data:      status,
	})
}

// BroadcastForwardAgentUpdated broadcasts a forward agent updated event.
func (h *AdminHub) BroadcastForwardAgentUpdated(agentSID string, changes any) {
	h.BroadcastForwardAgent(&AgentEvent{
		Type:      ForwardAgentEventUpdated,
		AgentID:   agentSID,
		Timestamp: biztime.NowUTC().Unix(),
		Data:      changes,
	})
}

// BroadcastForwardAgent sends a forward agent event to all matching SSE connections.
func (h *AdminHub) BroadcastForwardAgent(event *AgentEvent) {
	// Check throttling for status events
	if event.Type == ForwardAgentEventStatus {
		if !h.shouldPushAgentStatus(event.AgentID) {
			return
		}
	}

	data, err := h.formatSSEEvent(event)
	if err != nil {
		h.logger.Errorw("failed to format SSE event",
			"event_type", event.Type,
			"error", err,
		)
		return
	}

	h.connsMu.RLock()
	defer h.connsMu.RUnlock()

	for _, conn := range h.conns {
		if conn.ShouldReceiveAgent(event.AgentID) {
			if !conn.TrySend(data) {
				h.logger.Warnw("failed to send SSE event, channel full",
					"conn_id", conn.ID,
					"event_type", event.Type,
				)
			}
		}
	}
}

// GetConnCount returns the current number of SSE connections.
func (h *AdminHub) GetConnCount() int {
	h.connsMu.RLock()
	defer h.connsMu.RUnlock()
	return len(h.conns)
}

// shouldPushStatus checks if node status event should be pushed (throttle check).
func (h *AdminHub) shouldPushStatus(nodeSID string) bool {
	now := biztime.NowUTC()
	throttleDuration := time.Duration(h.statusThrottleMs) * time.Millisecond

	h.statusThrottleMu.Lock()
	defer h.statusThrottleMu.Unlock()

	lastPush, exists := h.statusThrottle[nodeSID]
	if exists && now.Sub(lastPush) < throttleDuration {
		return false
	}

	h.statusThrottle[nodeSID] = now
	return true
}

// shouldPushAgentStatus checks if agent status event should be pushed (throttle check).
func (h *AdminHub) shouldPushAgentStatus(agentSID string) bool {
	now := biztime.NowUTC()
	throttleDuration := time.Duration(h.statusThrottleMs) * time.Millisecond

	h.agentThrottleMu.Lock()
	defer h.agentThrottleMu.Unlock()

	lastPush, exists := h.agentThrottle[agentSID]
	if exists && now.Sub(lastPush) < throttleDuration {
		return false
	}

	h.agentThrottle[agentSID] = now
	return true
}

// formatSSEEvent formats an event as SSE data.
func (h *AdminHub) formatSSEEvent(event *AgentEvent) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	// SSE format: "event: <type>\ndata: <json>\n\n"
	return []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, data)), nil
}

// CleanupThrottleCache removes old entries from the throttle cache.
// Should be called periodically to prevent memory leaks.
func (h *AdminHub) CleanupThrottleCache() {
	now := biztime.NowUTC()
	threshold := time.Duration(h.statusThrottleMs*2) * time.Millisecond

	// Cleanup node status throttle
	h.statusThrottleMu.Lock()
	for nodeSID, lastPush := range h.statusThrottle {
		if now.Sub(lastPush) > threshold {
			delete(h.statusThrottle, nodeSID)
		}
	}
	h.statusThrottleMu.Unlock()

	// Cleanup agent status throttle
	h.agentThrottleMu.Lock()
	for agentSID, lastPush := range h.agentThrottle {
		if now.Sub(lastPush) > threshold {
			delete(h.agentThrottle, agentSID)
		}
	}
	h.agentThrottleMu.Unlock()
}
