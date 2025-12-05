package forward

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	hubWriteWait  = 10 * time.Second
	hubPongWait   = 60 * time.Second
	hubPingPeriod = 30 * time.Second
)

// HubConn represents a WebSocket connection to the AgentHub.
type HubConn struct {
	conn   *websocket.Conn
	send   chan *HubMessage
	mu     sync.Mutex
	closed bool

	// Message handler callback
	onMessage func(msg *HubMessage)
}

// HubMessage is the unified WebSocket message envelope.
type HubMessage struct {
	Type      string `json:"type"`
	AgentID   uint   `json:"agent_id,omitempty"`
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
)

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
func (hc *HubConn) SendProbeResult(result *ProbeResult) error {
	msg := &HubMessage{
		Type:      MsgTypeProbeResult,
		Timestamp: time.Now().Unix(),
		Data:      result,
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

		if hc.onMessage != nil {
			hc.onMessage(&msg)
		}
	}
}

// writePump writes messages to the WebSocket.
func (hc *HubConn) writePump(ctx context.Context) error {
	ticker := time.NewTicker(hubPingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-hc.send:
			hc.conn.SetWriteDeadline(time.Now().Add(hubWriteWait))
			if !ok {
				hc.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return nil
			}

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
func (hc *HubConn) Close() error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if hc.closed {
		return nil
	}

	hc.closed = true
	close(hc.send)

	_ = hc.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)

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
type ProbeTaskHandler func(task *ProbeTask) *ProbeResult

// RunHubLoop connects to the hub and handles messages.
// This is a convenience method that manages the connection lifecycle.
// The probeHandler is called when a probe task is received.
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
