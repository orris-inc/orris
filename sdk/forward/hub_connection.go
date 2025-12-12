package forward

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
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
