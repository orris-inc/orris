// Package services provides infrastructure services.
package services

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AgentHub manages WebSocket connections for forward agents.
// Simplified version: only supports forward agent connections for probe functionality.
type AgentHub struct {
	// Forward agent connections: map[AgentID]*AgentHubConn
	agents   map[uint]*AgentHubConn
	agentsMu sync.RWMutex

	// Status handler for forward agent
	statusHandler   StatusHandler
	statusHandlerMu sync.RWMutex

	// Message handlers for specific message types (domain extensions)
	messageHandlers   []MessageHandler
	messageHandlersMu sync.RWMutex

	// Callbacks
	onAgentOnline  func(agentID uint)
	onAgentOffline func(agentID uint)

	logger logger.Interface
}

// AgentHubConn represents a forward agent WebSocket connection.
type AgentHubConn struct {
	AgentID     uint
	Conn        *websocket.Conn
	Send        chan *dto.HubMessage
	LastSeen    time.Time
	ConnectedAt time.Time
}

// NewAgentHub creates a new AgentHub instance.
func NewAgentHub(log logger.Interface) *AgentHub {
	return &AgentHub{
		agents:          make(map[uint]*AgentHubConn),
		messageHandlers: make([]MessageHandler, 0),
		logger:          log,
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
	defer h.agentsMu.RUnlock()

	agentConn, ok := h.agents[agentID]
	if !ok {
		return ErrAgentNotConnected
	}

	select {
	case agentConn.Send <- msg:
		return nil
	default:
		return ErrSendChannelFull
	}
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
		close(existing.Send)
		existing.Conn.Close()
	}

	agentConn := &AgentHubConn{
		AgentID:     agentID,
		Conn:        conn,
		Send:        make(chan *dto.HubMessage, 256),
		LastSeen:    time.Now(),
		ConnectedAt: time.Now(),
	}
	h.agents[agentID] = agentConn

	h.logger.Infow("forward agent connected via websocket",
		"agent_id", agentID,
	)

	if h.onAgentOnline != nil {
		go h.onAgentOnline(agentID)
	}

	return agentConn
}

// UnregisterAgent removes an agent connection.
func (h *AgentHub) UnregisterAgent(agentID uint) {
	h.agentsMu.Lock()
	defer h.agentsMu.Unlock()

	if conn, ok := h.agents[agentID]; ok {
		close(conn.Send)
		delete(h.agents, agentID)

		h.logger.Infow("forward agent disconnected",
			"agent_id", agentID,
		)

		if h.onAgentOffline != nil {
			go h.onAgentOffline(agentID)
		}
	}
}

// HandleAgentStatus handles status update from an agent.
func (h *AgentHub) HandleAgentStatus(agentID uint, data any) {
	// Update last seen
	h.agentsMu.Lock()
	if conn, ok := h.agents[agentID]; ok {
		conn.LastSeen = time.Now()
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
func (h *AgentHub) SendCommandToAgent(agentID uint, cmd *dto.CommandData) error {
	h.agentsMu.RLock()
	defer h.agentsMu.RUnlock()

	agentConn, ok := h.agents[agentID]
	if !ok {
		return ErrAgentNotConnected
	}

	msg := &dto.HubMessage{
		Type:      dto.MsgTypeCommand,
		AgentID:   "", // Agent already knows its own ID; this field is primarily for logging/debug
		Timestamp: time.Now().Unix(),
		Data:      cmd,
	}

	select {
	case agentConn.Send <- msg:
		return nil
	default:
		return ErrSendChannelFull
	}
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

// HubErrors defines agent hub related errors.
var (
	ErrAgentNotConnected = &HubError{Code: "AGENT_NOT_CONNECTED", Message: "agent not connected"}
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
