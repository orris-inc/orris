package forward

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ProbeConn represents a WebSocket connection for probe communication.
type ProbeConn struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	closed bool
}

// ConnectProbe establishes a WebSocket connection for probe tasks.
// The connection allows the agent to receive probe tasks and send results.
func (c *Client) ConnectProbe(ctx context.Context) (*ProbeConn, error) {
	// Build WebSocket URL
	wsURL, err := c.buildProbeWSURL()
	if err != nil {
		return nil, fmt.Errorf("build websocket url: %w", err)
	}

	// Create dialer with context
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// Connect with token in query parameter
	conn, resp, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("websocket dial failed: status=%d, err=%w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("websocket dial: %w", err)
	}

	return &ProbeConn{
		conn:   conn,
		closed: false,
	}, nil
}

// buildProbeWSURL builds the WebSocket URL for probe connection.
func (c *Client) buildProbeWSURL() (string, error) {
	// Parse base URL
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
		// Assume wss for unknown schemes
		u.Scheme = "wss"
	}

	// Build path
	u.Path = strings.TrimSuffix(u.Path, "/") + "/forward-agent-api/probe-ws"

	// Add token as query parameter (WebSocket doesn't support custom headers easily)
	q := u.Query()
	q.Set("token", c.token)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// ReadTask reads the next probe task from the server.
// This blocks until a task is received or the connection is closed.
func (pc *ProbeConn) ReadTask() (*ProbeTask, error) {
	pc.mu.Lock()
	if pc.closed {
		pc.mu.Unlock()
		return nil, ErrConnectionClosed
	}
	pc.mu.Unlock()

	for {
		_, message, err := pc.conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("read message: %w", err)
		}

		var msg ProbeMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			// Skip malformed messages
			continue
		}

		if msg.Type != ProbeMessageTypeTask {
			// Skip non-task messages
			continue
		}

		// Parse task data
		taskData, err := json.Marshal(msg.Data)
		if err != nil {
			continue
		}

		var task ProbeTask
		if err := json.Unmarshal(taskData, &task); err != nil {
			continue
		}

		return &task, nil
	}
}

// WriteResult sends a probe result to the server.
func (pc *ProbeConn) WriteResult(result *ProbeResult) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.closed {
		return ErrConnectionClosed
	}

	msg := ProbeMessage{
		Type: ProbeMessageTypeResult,
		Data: result,
	}

	if err := pc.conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// SetReadDeadline sets the read deadline for the connection.
func (pc *ProbeConn) SetReadDeadline(t time.Time) error {
	return pc.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline for the connection.
func (pc *ProbeConn) SetWriteDeadline(t time.Time) error {
	return pc.conn.SetWriteDeadline(t)
}

// Ping sends a ping message to the server.
func (pc *ProbeConn) Ping() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.closed {
		return ErrConnectionClosed
	}

	return pc.conn.WriteMessage(websocket.PingMessage, nil)
}

// Close closes the probe connection.
func (pc *ProbeConn) Close() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.closed {
		return nil
	}

	pc.closed = true

	// Send close message
	_ = pc.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)

	return pc.conn.Close()
}

// IsClosed returns true if the connection is closed.
func (pc *ProbeConn) IsClosed() bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.closed
}

// ProbeHandler is a callback function for handling probe tasks.
type ProbeHandler func(task *ProbeTask) *ProbeResult

// RunProbeLoop runs a loop that reads probe tasks and sends results.
// This is a convenience method that handles the read/write loop.
// The handler function should execute the probe and return the result.
// The loop runs until the context is canceled or an error occurs.
func (c *Client) RunProbeLoop(ctx context.Context, handler ProbeHandler) error {
	conn, err := c.ConnectProbe(ctx)
	if err != nil {
		return fmt.Errorf("connect probe: %w", err)
	}
	defer conn.Close()

	// Start ping goroutine to keep connection alive
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := conn.Ping(); err != nil {
					return
				}
			}
		}
	}()

	// Read and process tasks
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		task, err := conn.ReadTask()
		if err != nil {
			// Check if context was canceled
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("read task: %w", err)
		}

		// Execute probe in goroutine to not block reading
		go func(t *ProbeTask) {
			result := handler(t)
			if result != nil {
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteResult(result); err != nil {
					// Log error but don't stop the loop
					_ = err
				}
			}
		}(task)
	}
}

// ErrConnectionClosed is returned when the connection is already closed.
var ErrConnectionClosed = &ConnectionClosedError{}

// ConnectionClosedError represents a closed connection error.
type ConnectionClosedError struct{}

func (e *ConnectionClosedError) Error() string {
	return "connection closed"
}

// IsConnectionClosed returns true if the error is a connection closed error.
func IsConnectionClosed(err error) bool {
	_, ok := err.(*ConnectionClosedError)
	return ok
}

// IsWebSocketCloseError returns true if the error is a WebSocket close error.
func IsWebSocketCloseError(err error) bool {
	return websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseAbnormalClosure,
	)
}

// GetUnderlyingConn returns the underlying websocket connection.
// Use this only when you need low-level access to the connection.
func (pc *ProbeConn) GetUnderlyingConn() *websocket.Conn {
	return pc.conn
}

// SetPongHandler sets the pong handler for the connection.
func (pc *ProbeConn) SetPongHandler(handler func(string) error) {
	pc.conn.SetPongHandler(handler)
}

// Response returns the HTTP response from the handshake.
// Note: This is only available during the dial and may be nil.
func (c *Client) ProbeWSHandshake(ctx context.Context) (*http.Response, error) {
	wsURL, err := c.buildProbeWSURL()
	if err != nil {
		return nil, err
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	_, resp, err := dialer.DialContext(ctx, wsURL, nil)
	return resp, err
}
