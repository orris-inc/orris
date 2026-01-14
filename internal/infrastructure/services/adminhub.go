// Package services provides infrastructure services.
package services

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// DefaultEventCacheSize is the default number of events cached per user.
	DefaultEventCacheSize = 100

	// DefaultEventCacheTTL is the default TTL for cached events.
	DefaultEventCacheTTL = 5 * time.Minute
)

// CachedEvent represents a cached SSE event for replay.
type CachedEvent struct {
	ID        string    // Event ID (timestamp-sequence format)
	UserID    uint      // Owner user ID
	EventType string    // "node" or "agent"
	AgentSID  string    // Optional, for filtering
	Data      []byte    // Complete SSE data (with id field)
	Timestamp time.Time // For expiration cleanup
}

// SSEEventCache manages cached events per user using a ring buffer.
type SSEEventCache struct {
	events   []*CachedEvent // Ring buffer
	head     int            // Next write position
	size     int            // Current event count
	capacity int            // Max capacity
	mu       sync.RWMutex
}

// NewSSEEventCache creates a new event cache with given capacity.
func NewSSEEventCache(capacity int) *SSEEventCache {
	return &SSEEventCache{
		events:   make([]*CachedEvent, capacity),
		capacity: capacity,
	}
}

// Add adds an event to the cache.
func (c *SSEEventCache) Add(event *CachedEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.events[c.head] = event
	c.head = (c.head + 1) % c.capacity
	if c.size < c.capacity {
		c.size++
	}
}

// GetEventsAfter returns all events after the given event ID.
// Events are returned in chronological order.
// If eventType is not empty, only events of that type are returned.
// If agentSID is not empty, only events for that agent are returned.
func (c *SSEEventCache) GetEventsAfter(lastEventID string, eventType string, agentSID string, ttl time.Duration) []*CachedEvent {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.size == 0 {
		return nil
	}

	now := biztime.NowUTC()
	var result []*CachedEvent
	found := lastEventID == "" // If no lastEventID, return all non-expired events

	// Calculate start position (oldest event)
	start := (c.head - c.size + c.capacity) % c.capacity

	for i := 0; i < c.size; i++ {
		idx := (start + i) % c.capacity
		event := c.events[idx]
		if event == nil {
			continue
		}

		// Skip expired events
		if now.Sub(event.Timestamp) > ttl {
			continue
		}

		// Skip until we find the lastEventID
		if !found {
			if event.ID == lastEventID {
				found = true
			}
			continue
		}

		// Filter by event type
		if eventType != "" && event.EventType != eventType {
			continue
		}

		// Filter by agent SID (if specified)
		if agentSID != "" && event.AgentSID != agentSID {
			continue
		}

		result = append(result, event)
	}

	return result
}

// Cleanup removes expired events from the cache.
func (c *SSEEventCache) Cleanup(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := biztime.NowUTC()
	for i := 0; i < c.capacity; i++ {
		if c.events[i] != nil && now.Sub(c.events[i].Timestamp) > ttl {
			c.events[i] = nil
		}
	}
}

// eventIDGenerator generates unique event IDs.
type eventIDGenerator struct {
	lastTimestamp int64
	sequence      int64
	mu            sync.Mutex
}

// Generate generates a new unique event ID in format "timestamp-sequence".
func (g *eventIDGenerator) Generate() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := biztime.NowUTC().UnixMilli()
	if now == g.lastTimestamp {
		g.sequence++
	} else {
		g.lastTimestamp = now
		g.sequence = 0
	}

	return fmt.Sprintf("%d-%d", now, g.sequence)
}

// ParseEventID parses an event ID and returns timestamp and sequence.
func ParseEventID(id string) (timestamp int64, sequence int64, ok bool) {
	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		return 0, 0, false
	}

	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, false
	}

	seq, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, false
	}

	return ts, seq, true
}

// AgentEventType represents the type of agent SSE event.
// Agents include both Node agents and Forward agents.
type AgentEventType string

// Node agent event types for SSE.
const (
	NodeEventOnline      AgentEventType = "node:online"
	NodeEventOffline     AgentEventType = "node:offline"
	NodeEventStatus      AgentEventType = "node:status"
	NodeEventUpdated     AgentEventType = "node:updated"
	NodeEventBatchStatus AgentEventType = "nodes:status" // Batch status for aggregated push
)

// Forward agent event types for SSE.
const (
	ForwardAgentEventOnline      AgentEventType = "agent:online"
	ForwardAgentEventOffline     AgentEventType = "agent:offline"
	ForwardAgentEventStatus      AgentEventType = "agent:status"
	ForwardAgentEventUpdated     AgentEventType = "agent:updated"
	ForwardAgentEventBatchStatus AgentEventType = "agents:status" // Batch status for aggregated push
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

// BatchAgentStatusEvent represents a batch status event for aggregated push.
type BatchAgentStatusEvent struct {
	Type      AgentEventType              `json:"type"`
	Timestamp int64                       `json:"timestamp"`
	Agents    map[string]*AgentStatusData `json:"agents"` // agentSID -> status data
}

// AgentStatusData holds status data for a single agent in batch events.
type AgentStatusData struct {
	Name   string `json:"name,omitempty"`
	Status any    `json:"status"`
}

// AgentStatusQuerier queries agent status from storage.
type AgentStatusQuerier interface {
	// GetBatchStatus returns status for multiple agents by their SIDs.
	// Returns a map of agentSID -> (name, status).
	GetBatchStatus(agentSIDs []string) (map[string]*AgentStatusData, error)
}

// NodeStatusQuerier queries node status from storage.
type NodeStatusQuerier interface {
	// GetBatchStatus returns status for multiple nodes by their SIDs.
	// Returns a map of nodeSID -> (name, status).
	GetBatchStatus(nodeSIDs []string) (map[string]*AgentStatusData, error)
}

// SSEConn represents an SSE connection from admin frontend.
type SSEConn struct {
	ID           string
	UserID       uint
	Send         chan []byte
	NodeFilters  map[string]bool // nil means subscribe to all nodes (if SubscribedToNodes is true)
	AgentFilters map[string]bool // nil means subscribe to all agents (if SubscribedToAgents is true)
	ConnectedAt  time.Time
	closed       atomic.Bool

	// Subscription type flags - explicitly track what this connection subscribes to
	SubscribedToNodes  bool // true if this connection receives node events
	SubscribedToAgents bool // true if this connection receives forward agent events
}

// TrySend attempts to send data to the SSE connection.
// Returns false if the channel is closed or full.
// When channel is full, the connection is closed to notify event loop.
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
		// Channel full - client is likely disconnected or too slow.
		// Close the connection to notify the event loop.
		c.Close()
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
	// First check if this connection subscribes to node events at all
	if !c.SubscribedToNodes {
		return false
	}
	if c.NodeFilters == nil {
		return true // No filter, receive all
	}
	return c.NodeFilters[nodeSID]
}

// ShouldReceiveAgent checks if this connection should receive events for the given agent.
func (c *SSEConn) ShouldReceiveAgent(agentSID string) bool {
	// First check if this connection subscribes to agent events at all
	if !c.SubscribedToAgents {
		return false
	}
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

	// Event caching for Last-Event-ID support: map[userID]*SSEEventCache
	eventCaches   map[uint]*SSEEventCache
	eventCachesMu sync.RWMutex
	eventIDGen    *eventIDGenerator

	// Agent status querier for batch status retrieval (optional)
	agentStatusQuerier AgentStatusQuerier

	// Node status querier for batch status retrieval (optional)
	nodeStatusQuerier NodeStatusQuerier

	// Configuration
	maxConnsPerUser  int
	statusThrottleMs int64
	agentBroadcastMs int64         // Aggregated broadcast interval for agents (default: 5000ms)
	nodeBroadcastMs  int64         // Aggregated broadcast interval for nodes (default: 5000ms)
	eventCacheSize   int           // Events cached per user (default: 100)
	eventCacheTTL    time.Duration // Event cache TTL (default: 5 minutes)

	// Shutdown signal
	done     chan struct{}
	shutdown atomic.Bool

	logger logger.Interface
}

// AdminHubConfig holds configuration for AdminHub.
type AdminHubConfig struct {
	MaxConnsPerUser  int           // Max SSE connections per user (default: 5)
	StatusThrottleMs int64         // Throttle interval for status events in ms (default: 5000) - used for cleanup
	AgentBroadcastMs int64         // Aggregated broadcast interval for agent status in ms (default: 5000, min: 1000)
	NodeBroadcastMs  int64         // Aggregated broadcast interval for node status in ms (default: 5000, min: 1000)
	EventCacheSize   int           // Events cached per user (default: 100)
	EventCacheTTL    time.Duration // Event cache TTL (default: 5 minutes)
}

// NewAdminHub creates a new AdminHub instance.
func NewAdminHub(log logger.Interface, config *AdminHubConfig) *AdminHub {
	maxConns := 5
	throttleMs := int64(5000)
	agentBroadcastMs := int64(5000)
	nodeBroadcastMs := int64(5000)
	eventCacheSize := DefaultEventCacheSize
	eventCacheTTL := DefaultEventCacheTTL

	if config != nil {
		if config.MaxConnsPerUser > 0 {
			maxConns = config.MaxConnsPerUser
		}
		if config.StatusThrottleMs > 0 {
			throttleMs = config.StatusThrottleMs
		}
		if config.AgentBroadcastMs >= 1000 {
			agentBroadcastMs = config.AgentBroadcastMs
		}
		if config.NodeBroadcastMs >= 1000 {
			nodeBroadcastMs = config.NodeBroadcastMs
		}
		if config.EventCacheSize > 0 {
			eventCacheSize = config.EventCacheSize
		}
		if config.EventCacheTTL > 0 {
			eventCacheTTL = config.EventCacheTTL
		}
	}

	h := &AdminHub{
		conns:            make(map[string]*SSEConn),
		userConns:        make(map[uint]int),
		statusThrottle:   make(map[string]time.Time),
		eventCaches:      make(map[uint]*SSEEventCache),
		eventIDGen:       &eventIDGenerator{},
		maxConnsPerUser:  maxConns,
		statusThrottleMs: throttleMs,
		agentBroadcastMs: agentBroadcastMs,
		nodeBroadcastMs:  nodeBroadcastMs,
		eventCacheSize:   eventCacheSize,
		eventCacheTTL:    eventCacheTTL,
		done:             make(chan struct{}),
		logger:           log,
	}

	// Start background goroutines
	go h.cleanupLoop()
	go h.agentBroadcastLoop()
	go h.nodeBroadcastLoop()

	return h
}

// cleanupLoop periodically cleans up the throttle cache and closed connections.
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
			h.cleanupClosedConns()
		}
	}
}

// cleanupClosedConns removes connections that have been marked as closed.
func (h *AdminHub) cleanupClosedConns() {
	// Collect closed connection IDs
	var closedIDs []string

	h.connsMu.RLock()
	for connID, conn := range h.conns {
		if conn.closed.Load() {
			closedIDs = append(closedIDs, connID)
		}
	}
	h.connsMu.RUnlock()

	// Remove closed connections
	for _, connID := range closedIDs {
		h.UnregisterConn(connID)
	}

	if len(closedIDs) > 0 {
		h.logger.Debugw("cleaned up closed SSE connections",
			"count", len(closedIDs),
		)
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

// RegisterConn registers a new SSE connection for node events only.
// Returns the connection or nil if max connections exceeded or hub is shutdown.
func (h *AdminHub) RegisterConn(connID string, userID uint, nodeFilters []string) *SSEConn {
	return h.RegisterConnWithSubscription(connID, userID, nodeFilters, nil, true, false)
}

// RegisterConnWithFilters registers a new SSE connection for agent events only.
// This is used by forward agent SSE handler.
// Returns the connection or nil if max connections exceeded or hub is shutdown.
func (h *AdminHub) RegisterConnWithFilters(connID string, userID uint, nodeFilters, agentFilters []string) *SSEConn {
	return h.RegisterConnWithSubscription(connID, userID, nodeFilters, agentFilters, false, true)
}

// RegisterConnWithSubscription registers a new SSE connection with explicit subscription types.
// subscribeNodes: if true, this connection receives node events
// subscribeAgents: if true, this connection receives forward agent events
// Returns the connection or nil if max connections exceeded or hub is shutdown.
func (h *AdminHub) RegisterConnWithSubscription(connID string, userID uint, nodeFilters, agentFilters []string, subscribeNodes, subscribeAgents bool) *SSEConn {
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
		ID:                 connID,
		UserID:             userID,
		Send:               make(chan []byte, 64),
		NodeFilters:        nodeFilterMap,
		AgentFilters:       agentFilterMap,
		ConnectedAt:        biztime.NowUTC(),
		SubscribedToNodes:  subscribeNodes,
		SubscribedToAgents: subscribeAgents,
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

	h.logger.Debugw("SSE connection registered",
		"conn_id", connID,
		"user_id", userID,
		"node_filters", nodeFilters,
		"agent_filters", agentFilters,
		"subscribe_nodes", subscribeNodes,
		"subscribe_agents", subscribeAgents,
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

		h.logger.Debugw("SSE connection unregistered",
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

	result, err := h.formatSSEEvent(event)
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
			// Cache event for this user
			h.cacheEvent(conn.UserID, result, "node")

			if !conn.TrySend(result.Data) {
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
// Note: For status events, use the aggregated broadcast via agentBroadcastLoop instead.
func (h *AdminHub) BroadcastForwardAgent(event *AgentEvent) {
	result, err := h.formatSSEEvent(event)
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
			// Cache event for this user
			h.cacheEvent(conn.UserID, result, "agent")

			if !conn.TrySend(result.Data) {
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

// formatSSEEvent formats an event as SSE data.
// formatSSEEventResult contains the formatted event data and metadata.
type formatSSEEventResult struct {
	ID       string // Event ID
	Data     []byte // Complete SSE data with id field
	AgentSID string // Agent SID for filtering
}

func (h *AdminHub) formatSSEEvent(event *AgentEvent) (*formatSSEEventResult, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	eventID := h.eventIDGen.Generate()

	// SSE format: "id: <id>\nevent: <type>\ndata: <json>\n\n"
	sseData := []byte(fmt.Sprintf("id: %s\nevent: %s\ndata: %s\n\n", eventID, event.Type, data))

	return &formatSSEEventResult{
		ID:       eventID,
		Data:     sseData,
		AgentSID: event.AgentID,
	}, nil
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

	// Cleanup expired events from all user caches
	h.eventCachesMu.RLock()
	for _, cache := range h.eventCaches {
		cache.Cleanup(h.eventCacheTTL)
	}
	h.eventCachesMu.RUnlock()
}

// cacheEvent caches an event for the given user.
func (h *AdminHub) cacheEvent(userID uint, result *formatSSEEventResult, eventType string) {
	h.eventCachesMu.Lock()
	cache, ok := h.eventCaches[userID]
	if !ok {
		cache = NewSSEEventCache(h.eventCacheSize)
		h.eventCaches[userID] = cache
	}
	h.eventCachesMu.Unlock()

	cache.Add(&CachedEvent{
		ID:        result.ID,
		UserID:    userID,
		EventType: eventType,
		AgentSID:  result.AgentSID,
		Data:      result.Data,
		Timestamp: biztime.NowUTC(),
	})
}

// GetEventsAfter returns events after the given lastEventID for replay.
// eventType can be "node", "agent", or empty for all types.
// agentSIDs can be used to filter events by specific agents.
func (h *AdminHub) GetEventsAfter(userID uint, lastEventID string, eventType string, agentSIDs []string) []*CachedEvent {
	h.eventCachesMu.RLock()
	cache, ok := h.eventCaches[userID]
	h.eventCachesMu.RUnlock()

	if !ok {
		return nil
	}

	// If specific agent SIDs are requested, filter for each and combine
	if len(agentSIDs) > 0 {
		var result []*CachedEvent
		seenIDs := make(map[string]bool)
		for _, sid := range agentSIDs {
			events := cache.GetEventsAfter(lastEventID, eventType, sid, h.eventCacheTTL)
			for _, e := range events {
				if !seenIDs[e.ID] {
					seenIDs[e.ID] = true
					result = append(result, e)
				}
			}
		}
		return result
	}

	return cache.GetEventsAfter(lastEventID, eventType, "", h.eventCacheTTL)
}

// SetAgentStatusQuerier sets the agent status querier for batch status retrieval.
// Must be called before any SSE connections are established.
func (h *AdminHub) SetAgentStatusQuerier(querier AgentStatusQuerier) {
	h.agentStatusQuerier = querier
}

// agentBroadcastLoop periodically broadcasts aggregated agent status to SSE connections.
func (h *AdminHub) agentBroadcastLoop() {
	interval := time.Duration(h.agentBroadcastMs) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.done:
			return
		case <-ticker.C:
			h.broadcastAggregatedAgentStatus()
		}
	}
}

// broadcastAggregatedAgentStatus collects all subscribed agent SIDs and broadcasts their status.
func (h *AdminHub) broadcastAggregatedAgentStatus() {
	if h.agentStatusQuerier == nil {
		return
	}

	// Collect all unique agent SIDs from connections that subscribe to agent events
	h.connsMu.RLock()
	if len(h.conns) == 0 {
		h.connsMu.RUnlock()
		return
	}

	// Build a set of unique agent SIDs and track which connections subscribe to all
	allAgentSIDs := make(map[string]struct{})
	hasSubscribeAll := false
	subscribeAllConns := make([]*SSEConn, 0)
	filteredConns := make([]*SSEConn, 0)

	for _, conn := range h.conns {
		// Skip connections that don't subscribe to agent events
		if !conn.SubscribedToAgents {
			continue
		}

		if conn.AgentFilters == nil {
			// This connection subscribes to all agents
			hasSubscribeAll = true
			subscribeAllConns = append(subscribeAllConns, conn)
		} else {
			filteredConns = append(filteredConns, conn)
			for agentSID := range conn.AgentFilters {
				allAgentSIDs[agentSID] = struct{}{}
			}
		}
	}
	h.connsMu.RUnlock()

	// If no agents to query, skip
	if len(allAgentSIDs) == 0 && !hasSubscribeAll {
		return
	}

	// Query batch status from storage
	// If hasSubscribeAll is true, we pass nil to get all agents
	var queryAgentSIDs []string
	if hasSubscribeAll {
		queryAgentSIDs = nil // nil means get all
	} else {
		queryAgentSIDs = make([]string, 0, len(allAgentSIDs))
		for sid := range allAgentSIDs {
			queryAgentSIDs = append(queryAgentSIDs, sid)
		}
	}

	statusMap, err := h.agentStatusQuerier.GetBatchStatus(queryAgentSIDs)
	if err != nil {
		h.logger.Errorw("failed to get batch agent status",
			"error", err,
			"agent_count", len(queryAgentSIDs),
		)
		return
	}

	if len(statusMap) == 0 {
		return
	}

	// Build and send batch events per connection type
	// This ensures each connection only receives agents they subscribed to
	timestamp := biztime.NowUTC().Unix()

	// For connections that subscribe to all agents, send full status map
	if len(subscribeAllConns) > 0 {
		fullEvent := &BatchAgentStatusEvent{
			Type:      ForwardAgentEventBatchStatus,
			Timestamp: timestamp,
			Agents:    statusMap,
		}
		fullResult, err := h.formatBatchSSEEvent(fullEvent)
		if err != nil {
			h.logger.Errorw("failed to format full batch SSE event",
				"error", err,
			)
		} else {
			for _, conn := range subscribeAllConns {
				// Cache event for this user
				h.cacheEvent(conn.UserID, fullResult, "agent")

				if !conn.TrySend(fullResult.Data) {
					h.logger.Warnw("failed to send batch SSE event, channel full",
						"conn_id", conn.ID,
					)
				}
			}
		}
	}

	// For connections with specific filters, send only their subscribed agents
	for _, conn := range filteredConns {
		filteredStatus := make(map[string]*AgentStatusData)
		for agentSID := range conn.AgentFilters {
			if status, ok := statusMap[agentSID]; ok {
				filteredStatus[agentSID] = status
			}
		}

		if len(filteredStatus) == 0 {
			continue
		}

		filteredEvent := &BatchAgentStatusEvent{
			Type:      ForwardAgentEventBatchStatus,
			Timestamp: timestamp,
			Agents:    filteredStatus,
		}
		filteredResult, err := h.formatBatchSSEEvent(filteredEvent)
		if err != nil {
			h.logger.Errorw("failed to format filtered batch SSE event",
				"error", err,
				"conn_id", conn.ID,
			)
			continue
		}

		// Cache event for this user
		h.cacheEvent(conn.UserID, filteredResult, "agent")

		if !conn.TrySend(filteredResult.Data) {
			h.logger.Warnw("failed to send batch SSE event, channel full",
				"conn_id", conn.ID,
			)
		}
	}
}

// formatBatchSSEEvent formats a batch event as SSE data.
func (h *AdminHub) formatBatchSSEEvent(event *BatchAgentStatusEvent) (*formatSSEEventResult, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	eventID := h.eventIDGen.Generate()

	// SSE format: "id: <id>\nevent: <type>\ndata: <json>\n\n"
	sseData := []byte(fmt.Sprintf("id: %s\nevent: %s\ndata: %s\n\n", eventID, event.Type, data))

	return &formatSSEEventResult{
		ID:       eventID,
		Data:     sseData,
		AgentSID: "", // Batch events don't have a single agent SID
	}, nil
}

// BroadcastAgentStatusToConn sends current agent status to a specific connection.
// Used for initial status push when a new SSE connection is established.
func (h *AdminHub) BroadcastAgentStatusToConn(conn *SSEConn) {
	if h.agentStatusQuerier == nil {
		return
	}

	// Get agent SIDs this connection is interested in
	var agentSIDs []string
	if conn.AgentFilters == nil {
		agentSIDs = nil // nil means get all
	} else {
		agentSIDs = make([]string, 0, len(conn.AgentFilters))
		for sid := range conn.AgentFilters {
			agentSIDs = append(agentSIDs, sid)
		}
	}

	statusMap, err := h.agentStatusQuerier.GetBatchStatus(agentSIDs)
	if err != nil {
		h.logger.Errorw("failed to get initial agent status",
			"error", err,
			"conn_id", conn.ID,
		)
		return
	}

	if len(statusMap) == 0 {
		return
	}

	// Build and send batch event
	batchEvent := &BatchAgentStatusEvent{
		Type:      ForwardAgentEventBatchStatus,
		Timestamp: biztime.NowUTC().Unix(),
		Agents:    statusMap,
	}

	result, err := h.formatBatchSSEEvent(batchEvent)
	if err != nil {
		h.logger.Errorw("failed to format initial batch SSE event",
			"error", err,
			"conn_id", conn.ID,
		)
		return
	}

	// Cache event for this user
	h.cacheEvent(conn.UserID, result, "agent")

	if !conn.TrySend(result.Data) {
		h.logger.Warnw("failed to send initial batch SSE event, channel full",
			"conn_id", conn.ID,
		)
	}
}

// SetNodeStatusQuerier sets the node status querier for batch status retrieval.
// Must be called before any SSE connections are established.
func (h *AdminHub) SetNodeStatusQuerier(querier NodeStatusQuerier) {
	h.nodeStatusQuerier = querier
}

// nodeBroadcastLoop periodically broadcasts aggregated node status to SSE connections.
func (h *AdminHub) nodeBroadcastLoop() {
	interval := time.Duration(h.nodeBroadcastMs) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.done:
			return
		case <-ticker.C:
			h.broadcastAggregatedNodeStatus()
		}
	}
}

// broadcastAggregatedNodeStatus collects all subscribed node SIDs and broadcasts their status.
func (h *AdminHub) broadcastAggregatedNodeStatus() {
	if h.nodeStatusQuerier == nil {
		return
	}

	// Collect all unique node SIDs from connections that subscribe to node events
	h.connsMu.RLock()
	if len(h.conns) == 0 {
		h.connsMu.RUnlock()
		return
	}

	// Build a set of unique node SIDs and track which connections subscribe to all
	allNodeSIDs := make(map[string]struct{})
	hasSubscribeAll := false
	subscribeAllConns := make([]*SSEConn, 0)
	filteredConns := make([]*SSEConn, 0)

	for _, conn := range h.conns {
		// Skip connections that don't subscribe to node events
		if !conn.SubscribedToNodes {
			continue
		}

		if conn.NodeFilters == nil {
			// This connection subscribes to all nodes
			hasSubscribeAll = true
			subscribeAllConns = append(subscribeAllConns, conn)
		} else {
			filteredConns = append(filteredConns, conn)
			for nodeSID := range conn.NodeFilters {
				allNodeSIDs[nodeSID] = struct{}{}
			}
		}
	}
	h.connsMu.RUnlock()

	// If no nodes to query, skip
	if len(allNodeSIDs) == 0 && !hasSubscribeAll {
		return
	}

	// Query batch status from storage
	// If hasSubscribeAll is true, we pass nil to get all nodes
	var queryNodeSIDs []string
	if hasSubscribeAll {
		queryNodeSIDs = nil // nil means get all
	} else {
		queryNodeSIDs = make([]string, 0, len(allNodeSIDs))
		for sid := range allNodeSIDs {
			queryNodeSIDs = append(queryNodeSIDs, sid)
		}
	}

	statusMap, err := h.nodeStatusQuerier.GetBatchStatus(queryNodeSIDs)
	if err != nil {
		h.logger.Errorw("failed to get batch node status",
			"error", err,
			"node_count", len(queryNodeSIDs),
		)
		return
	}

	if len(statusMap) == 0 {
		return
	}

	// Build and send batch events per connection type
	// This ensures each connection only receives nodes they subscribed to
	timestamp := biztime.NowUTC().Unix()

	// For connections that subscribe to all nodes, send full status map
	if len(subscribeAllConns) > 0 {
		fullEvent := &BatchAgentStatusEvent{
			Type:      NodeEventBatchStatus,
			Timestamp: timestamp,
			Agents:    statusMap,
		}
		fullResult, err := h.formatBatchSSEEvent(fullEvent)
		if err != nil {
			h.logger.Errorw("failed to format full batch node SSE event",
				"error", err,
			)
		} else {
			for _, conn := range subscribeAllConns {
				// Cache event for this user
				h.cacheEvent(conn.UserID, fullResult, "node")

				if !conn.TrySend(fullResult.Data) {
					h.logger.Warnw("failed to send batch node SSE event, channel full",
						"conn_id", conn.ID,
					)
				}
			}
		}
	}

	// For connections with specific filters, send only their subscribed nodes
	for _, conn := range filteredConns {
		filteredStatus := make(map[string]*AgentStatusData)
		for nodeSID := range conn.NodeFilters {
			if status, ok := statusMap[nodeSID]; ok {
				filteredStatus[nodeSID] = status
			}
		}

		if len(filteredStatus) == 0 {
			continue
		}

		filteredEvent := &BatchAgentStatusEvent{
			Type:      NodeEventBatchStatus,
			Timestamp: timestamp,
			Agents:    filteredStatus,
		}
		filteredResult, err := h.formatBatchSSEEvent(filteredEvent)
		if err != nil {
			h.logger.Errorw("failed to format filtered batch node SSE event",
				"error", err,
				"conn_id", conn.ID,
			)
			continue
		}

		// Cache event for this user
		h.cacheEvent(conn.UserID, filteredResult, "node")

		if !conn.TrySend(filteredResult.Data) {
			h.logger.Warnw("failed to send batch node SSE event, channel full",
				"conn_id", conn.ID,
			)
		}
	}
}

// BroadcastNodeStatusToConn sends current node status to a specific connection.
// Used for initial status push when a new SSE connection is established.
func (h *AdminHub) BroadcastNodeStatusToConn(conn *SSEConn) {
	if h.nodeStatusQuerier == nil {
		return
	}

	// Get node SIDs this connection is interested in
	var nodeSIDs []string
	if conn.NodeFilters == nil {
		nodeSIDs = nil // nil means get all
	} else {
		nodeSIDs = make([]string, 0, len(conn.NodeFilters))
		for sid := range conn.NodeFilters {
			nodeSIDs = append(nodeSIDs, sid)
		}
	}

	statusMap, err := h.nodeStatusQuerier.GetBatchStatus(nodeSIDs)
	if err != nil {
		h.logger.Errorw("failed to get initial node status",
			"error", err,
			"conn_id", conn.ID,
		)
		return
	}

	if len(statusMap) == 0 {
		return
	}

	// Build and send batch event
	batchEvent := &BatchAgentStatusEvent{
		Type:      NodeEventBatchStatus,
		Timestamp: biztime.NowUTC().Unix(),
		Agents:    statusMap,
	}

	result, err := h.formatBatchSSEEvent(batchEvent)
	if err != nil {
		h.logger.Errorw("failed to format initial batch node SSE event",
			"error", err,
			"conn_id", conn.ID,
		)
		return
	}

	// Cache event for this user
	h.cacheEvent(conn.UserID, result, "node")

	if !conn.TrySend(result.Data) {
		h.logger.Warnw("failed to send initial batch node SSE event, channel full",
			"conn_id", conn.ID,
		)
	}
}
