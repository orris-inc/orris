package forward

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/gorilla/websocket"
)

const (
	hubWriteWait  = 10 * time.Second
	hubPongWait   = 60 * time.Second
	hubPingPeriod = 30 * time.Second
)

// ReconnectConfig holds reconnection strategy parameters.
type ReconnectConfig struct {
	// InitialInterval is the first retry delay (default: 1s)
	InitialInterval time.Duration

	// MaxInterval is the maximum retry delay (default: 60s)
	MaxInterval time.Duration

	// MaxElapsedTime is the total time to keep retrying. 0 means never stop (default: 0)
	MaxElapsedTime time.Duration

	// Multiplier is the exponential backoff multiplier (default: 2.0)
	Multiplier float64

	// RandomizationFactor adds jitter to prevent thundering herd (default: 0.1)
	RandomizationFactor float64

	// OnConnected is called when successfully connected
	OnConnected func()

	// OnDisconnected is called when disconnected (with error)
	OnDisconnected func(err error)

	// OnReconnecting is called before each reconnection attempt
	OnReconnecting func(attempt uint64, delay time.Duration)

	// OnConfigSync is called when a config sync message is received from the server.
	// This is used to handle incremental or full configuration updates.
	OnConfigSync ConfigSyncHandler
}

// DefaultReconnectConfig returns the default reconnection configuration.
func DefaultReconnectConfig() *ReconnectConfig {
	return &ReconnectConfig{
		InitialInterval:     1 * time.Second,
		MaxInterval:         60 * time.Second,
		MaxElapsedTime:      0, // Never give up
		Multiplier:          2.0,
		RandomizationFactor: 0.1,
	}
}

// HubConn represents a WebSocket connection to the AgentHub.
type HubConn struct {
	conn   *websocket.Conn
	send   chan *HubMessage
	done   chan struct{}  // Signal channel for graceful shutdown
	Events chan *HubEvent // Event channel for agent to receive events
	mu     sync.Mutex     // Protects closed and send channel access
	closed bool

	// Message handler callback
	onMessage func(msg *HubMessage)
}

// HubMessage is the unified WebSocket message envelope.
type HubMessage struct {
	Type      string `json:"type"`
	AgentID   string `json:"agent_id,omitempty"` // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Timestamp int64  `json:"timestamp"`
	Data      any    `json:"data,omitempty"`
}

// Hub message type constants.
const (
	// Agent -> Server message types.
	MsgTypeStatus    = "status"
	MsgTypeHeartbeat = "heartbeat"
	MsgTypeEvent     = "event"

	// Server -> Agent message types.
	MsgTypeCommand = "command"

	// Probe message types.
	MsgTypeProbeTask   = "probe_task"   // Server -> Agent
	MsgTypeProbeResult = "probe_result" // Agent -> Server

	// Config sync message types.
	MsgTypeConfigSync = "config_sync" // Server -> Agent
	MsgTypeConfigAck  = "config_ack"  // Agent -> Server
)

// HubEventType represents the type of event received from hub.
type HubEventType string

const (
	HubEventConfigSync HubEventType = "config_sync"
	HubEventProbeTask  HubEventType = "probe_task"
)

// HubEvent is a unified event structure for agent to consume.
type HubEvent struct {
	Type       HubEventType
	ConfigSync *ConfigSyncData
	ProbeTask  *ProbeTask
}

// ConfigSyncData represents configuration synchronization data.
type ConfigSyncData struct {
	Version            uint64         `json:"version"`
	FullSync           bool           `json:"full_sync"`
	Added              []RuleSyncData `json:"added,omitempty"`
	Updated            []RuleSyncData `json:"updated,omitempty"`
	Removed            []string       `json:"removed,omitempty"`              // Rule IDs to remove (Stripe-style prefixed, e.g., "fr_xxx")
	ClientToken        string         `json:"client_token,omitempty"`         // Agent's token for tunnel handshake (full sync only)
	TokenSigningSecret string         `json:"token_signing_secret,omitempty"` // Secret for local agent token verification (full sync only)
}

// RuleSyncData represents rule synchronization data (aligned with server DTO).
type RuleSyncData struct {
	ID                     string   `json:"id"`       // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	ShortID                string   `json:"short_id"` // Deprecated: use ID instead
	RuleType               string   `json:"rule_type"`
	ListenPort             uint16   `json:"listen_port"`
	TargetAddress          string   `json:"target_address,omitempty"`
	TargetPort             uint16   `json:"target_port,omitempty"`
	BindIP                 string   `json:"bind_ip,omitempty"` // Bind IP address for outbound connections
	Protocol               string   `json:"protocol"`
	Role                   string   `json:"role,omitempty"`
	AgentID                string   `json:"agent_id,omitempty"` // Entry agent ID (for exit agents to verify handshake)
	NextHopAgentID         string   `json:"next_hop_agent_id,omitempty"`
	NextHopAddress         string   `json:"next_hop_address,omitempty"`
	NextHopWsPort          uint16   `json:"next_hop_ws_port,omitempty"`
	NextHopPort            uint16   `json:"next_hop_port,omitempty"`             // Next agent's listen port for direct_chain
	NextHopConnectionToken string   `json:"next_hop_connection_token,omitempty"` // Short-term token for next hop authentication
	ChainAgentIDs          []string `json:"chain_agent_ids,omitempty"`
	ChainPosition          int      `json:"chain_position,omitempty"`
	IsLastInChain          bool     `json:"is_last_in_chain,omitempty"`
}

// ConfigAckData represents configuration acknowledgment data.
type ConfigAckData struct {
	Version uint64 `json:"version"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ConnectHub establishes a WebSocket connection to the AgentHub.
// The connection allows the agent to receive commands and send status updates.
func (c *Client) ConnectHub(ctx context.Context) (*HubConn, error) {
	wsURL, err := c.buildHubWSURL()
	if err != nil {
		return nil, fmt.Errorf("build websocket url: %w", err)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, resp, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("websocket dial failed: status=%d, err=%w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("websocket dial: %w", err)
	}

	hubConn := &HubConn{
		conn:   conn,
		send:   make(chan *HubMessage, 256),
		done:   make(chan struct{}),
		Events: make(chan *HubEvent, 256),
		closed: false,
	}

	return hubConn, nil
}

// buildHubWSURL builds the WebSocket URL for hub connection.
func (c *Client) buildHubWSURL() (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base url: %w", err)
	}

	// Convert http(s) to ws(s)
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		u.Scheme = "wss"
	}

	// Build path: /ws/forward-agent
	u.Path = strings.TrimSuffix(u.Path, "/") + "/ws/forward-agent"

	// Add token as query parameter
	q := u.Query()
	q.Set("token", c.token)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// SetMessageHandler sets the callback for incoming messages.
func (hc *HubConn) SetMessageHandler(handler func(msg *HubMessage)) {
	hc.onMessage = handler
}

// Send sends a message to the server.
func (hc *HubConn) Send(msg *HubMessage) error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if hc.closed {
		return ErrConnectionClosed
	}

	select {
	case hc.send <- msg:
		return nil
	default:
		return fmt.Errorf("send channel full")
	}
}

// SendStatus sends a status update to the server.
func (hc *HubConn) SendStatus(status *AgentStatus) error {
	msg := &HubMessage{
		Type:      MsgTypeStatus,
		Timestamp: time.Now().Unix(),
		Data:      status,
	}
	return hc.Send(msg)
}

// SendProbeResult sends a probe result to the server.
func (hc *HubConn) SendProbeResult(result *ProbeTaskResult) error {
	msg := &HubMessage{
		Type:      MsgTypeProbeResult,
		Timestamp: time.Now().Unix(),
		Data:      result,
	}
	return hc.Send(msg)
}

// SendConfigAck sends a configuration acknowledgment to the server.
func (hc *HubConn) SendConfigAck(ack *ConfigAckData) error {
	msg := &HubMessage{
		Type:      MsgTypeConfigAck,
		Timestamp: time.Now().Unix(),
		Data:      ack,
	}
	return hc.Send(msg)
}

// SendEvent sends an event to the server.
func (hc *HubConn) SendEvent(eventType, message string, extra any) error {
	msg := &HubMessage{
		Type:      MsgTypeEvent,
		Timestamp: time.Now().Unix(),
		Data: map[string]any{
			"event_type": eventType,
			"message":    message,
			"extra":      extra,
		},
	}
	return hc.Send(msg)
}

// Run starts the read and write pumps. This blocks until the connection is closed.
func (hc *HubConn) Run(ctx context.Context) error {
	errChan := make(chan error, 2)

	// Start write pump
	go func() {
		errChan <- hc.writePump(ctx)
	}()

	// Start read pump
	go func() {
		errChan <- hc.readPump(ctx)
	}()

	// Wait for either pump to exit
	err := <-errChan
	hc.Close()
	return err
}

// readPump reads messages from the WebSocket.
func (hc *HubConn) readPump(ctx context.Context) error {
	hc.conn.SetReadLimit(65536)
	hc.conn.SetReadDeadline(time.Now().Add(hubPongWait))
	hc.conn.SetPongHandler(func(string) error {
		hc.conn.SetReadDeadline(time.Now().Add(hubPongWait))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, message, err := hc.conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read message: %w", err)
		}

		var msg HubMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue // Skip malformed messages
		}

		// Convert message to event and send to Events channel
		hc.dispatchEvent(&msg)

		// Also call legacy onMessage handler for backward compatibility
		if hc.onMessage != nil {
			hc.onMessage(&msg)
		}
	}
}

// dispatchEvent converts HubMessage to HubEvent and sends to Events channel.
func (hc *HubConn) dispatchEvent(msg *HubMessage) {
	var event *HubEvent

	switch msg.Type {
	case MsgTypeConfigSync:
		configSync := parseConfigSync(msg.Data)
		if configSync != nil {
			event = &HubEvent{
				Type:       HubEventConfigSync,
				ConfigSync: configSync,
			}
		}
	case MsgTypeProbeTask:
		probeTask := parseProbeTask(msg.Data)
		if probeTask != nil {
			event = &HubEvent{
				Type:      HubEventProbeTask,
				ProbeTask: probeTask,
			}
		}
	default:
		// Ignore other message types for event channel
		return
	}

	if event != nil {
		select {
		case hc.Events <- event:
		default:
			// Event channel full, skip this event to avoid blocking
		}
	}
}

// parseConfigSync parses config sync data from message data.
func parseConfigSync(data any) *ConfigSyncData {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil
	}

	var configSync ConfigSyncData
	if err := json.Unmarshal(dataBytes, &configSync); err != nil {
		return nil
	}
	return &configSync
}

// writePump writes messages to the WebSocket.
// All writes to the websocket connection are done here to avoid concurrent writes.
func (hc *HubConn) writePump(ctx context.Context) error {
	ticker := time.NewTicker(hubPingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context canceled, send close message and exit
			hc.conn.SetWriteDeadline(time.Now().Add(hubWriteWait))
			hc.conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return ctx.Err()

		case <-hc.done:
			// Graceful shutdown requested, send close message
			hc.conn.SetWriteDeadline(time.Now().Add(hubWriteWait))
			hc.conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return nil

		case msg, ok := <-hc.send:
			if !ok {
				// Send channel closed, should not happen as we use done channel
				return nil
			}
			hc.conn.SetWriteDeadline(time.Now().Add(hubWriteWait))
			if err := hc.conn.WriteJSON(msg); err != nil {
				return fmt.Errorf("write message: %w", err)
			}

		case <-ticker.C:
			hc.conn.SetWriteDeadline(time.Now().Add(hubWriteWait))
			if err := hc.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return fmt.Errorf("ping: %w", err)
			}
		}
	}
}

// Close closes the hub connection.
// It signals the writePump to send close message and shutdown gracefully.
func (hc *HubConn) Close() error {
	hc.mu.Lock()
	if hc.closed {
		hc.mu.Unlock()
		return nil
	}
	hc.closed = true
	close(hc.done) // Signal writePump to shutdown and send close message
	hc.mu.Unlock()

	// Close Events channel (safe to close from any goroutine)
	// Note: send channel is not closed here to avoid send-on-closed-channel panic
	// writePump will exit via done channel
	close(hc.Events)

	return hc.conn.Close()
}

// IsClosed returns true if the connection is closed.
func (hc *HubConn) IsClosed() bool {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	return hc.closed
}

// HubMessageHandler is a callback function for handling hub messages.
type HubMessageHandler func(msg *HubMessage)

// ProbeTaskHandler is a callback function for handling probe tasks.
// Returns the probe result to be sent back to the server.
type ProbeTaskHandler func(task *ProbeTask) *ProbeTaskResult

// ConfigSyncHandler is a callback function for handling config sync events.
// The handler receives the config sync data and should return an error if processing failed.
type ConfigSyncHandler func(data *ConfigSyncData) error

// RunHubLoop connects to the hub and handles messages.
// This is a convenience method that manages the connection lifecycle without reconnection.
// For production use, prefer RunHubLoopWithReconnect which handles automatic reconnection.
func (c *Client) RunHubLoop(ctx context.Context, probeHandler ProbeTaskHandler) error {
	conn, err := c.ConnectHub(ctx)
	if err != nil {
		return fmt.Errorf("connect hub: %w", err)
	}
	defer conn.Close()

	// Set up message handler
	conn.SetMessageHandler(func(msg *HubMessage) {
		switch msg.Type {
		case MsgTypeProbeTask:
			if probeHandler != nil {
				go func() {
					task := parseProbeTask(msg.Data)
					if task != nil {
						result := probeHandler(task)
						if result != nil {
							conn.SendProbeResult(result)
						}
					}
				}()
			}
		case MsgTypeCommand:
			// Handle commands if needed
		}
	})

	// Run the connection
	return conn.Run(ctx)
}

// parseProbeTask parses probe task from message data.
func parseProbeTask(data any) *ProbeTask {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil
	}

	var task ProbeTask
	if err := json.Unmarshal(dataBytes, &task); err != nil {
		return nil
	}
	return &task
}

// RunHubLoopWithReconnect connects to the hub with automatic reconnection.
// It uses exponential backoff strategy to retry failed connections.
// The probeHandler is called when a probe task is received.
// This method blocks until the context is canceled.
//
// Use the ReconnectConfig callbacks to handle connection events:
//   - OnConnected: called when successfully connected
//   - OnDisconnected: called when disconnected (with error)
//   - OnReconnecting: called before each reconnection attempt
func (c *Client) RunHubLoopWithReconnect(ctx context.Context, probeHandler ProbeTaskHandler, config *ReconnectConfig) error {
	if config == nil {
		config = DefaultReconnectConfig()
	}

	// Create exponential backoff strategy
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = config.InitialInterval
	expBackoff.MaxInterval = config.MaxInterval
	expBackoff.Multiplier = config.Multiplier
	expBackoff.RandomizationFactor = config.RandomizationFactor
	expBackoff.Reset()

	var attempt uint64
	startTime := time.Now()

	// Reconnection loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		attempt++

		// Attempt to connect and run
		err := c.runHubLoopOnce(ctx, probeHandler, config)

		// Connection ended, call disconnect callback
		if config.OnDisconnected != nil {
			config.OnDisconnected(err)
		}

		// If context was canceled, exit immediately
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Check max elapsed time
		if config.MaxElapsedTime > 0 && time.Since(startTime) >= config.MaxElapsedTime {
			return fmt.Errorf("reconnection failed after %v: %w", config.MaxElapsedTime, err)
		}

		// Calculate next backoff delay
		delay := expBackoff.NextBackOff()
		if delay == backoff.Stop {
			return fmt.Errorf("reconnection failed: %w", err)
		}

		// Call reconnecting callback with attempt info
		if config.OnReconnecting != nil {
			config.OnReconnecting(attempt, delay)
		}

		// Wait before reconnecting
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// Continue to next connection attempt
		}
	}
}

// runHubLoopOnce executes a single hub connection lifecycle.
func (c *Client) runHubLoopOnce(ctx context.Context, probeHandler ProbeTaskHandler, config *ReconnectConfig) error {
	conn, err := c.ConnectHub(ctx)
	if err != nil {
		return fmt.Errorf("connect hub: %w", err)
	}
	defer conn.Close()

	// Call connected callback
	if config.OnConnected != nil {
		config.OnConnected()
	}

	// Set up message handler
	conn.SetMessageHandler(func(msg *HubMessage) {
		switch msg.Type {
		case MsgTypeProbeTask:
			if probeHandler != nil {
				go func() {
					task := parseProbeTask(msg.Data)
					if task != nil {
						result := probeHandler(task)
						if result != nil {
							conn.SendProbeResult(result)
						}
					}
				}()
			}
		case MsgTypeConfigSync:
			if config.OnConfigSync != nil {
				go func() {
					configSync := parseConfigSync(msg.Data)
					if configSync != nil {
						err := config.OnConfigSync(configSync)
						// Send acknowledgment back to server
						ack := &ConfigAckData{
							Version: configSync.Version,
							Success: err == nil,
						}
						if err != nil {
							ack.Error = err.Error()
						}
						conn.SendConfigAck(ack)
					}
				}()
			}
		case MsgTypeCommand:
			// Handle commands if needed
		}
	})

	// Run the connection (blocks until disconnected)
	return conn.Run(ctx)
}
