package forward

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// TunnelClient is a WebSocket tunnel client for Entry agents.
// It connects to an Exit agent and forwards data through the tunnel.
type TunnelClient struct {
	endpoint *ExitEndpoint
	conn     *websocket.Conn
	logger   *slog.Logger

	writeMu   sync.Mutex
	forwarder *EntryForwarder

	reconnectInterval time.Duration
	heartbeatInterval time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// TunnelClientOption configures TunnelClient.
type TunnelClientOption func(*TunnelClient)

// WithReconnectInterval sets the reconnect interval.
func WithReconnectInterval(d time.Duration) TunnelClientOption {
	return func(t *TunnelClient) {
		t.reconnectInterval = d
	}
}

// WithHeartbeatInterval sets the heartbeat interval.
func WithHeartbeatInterval(d time.Duration) TunnelClientOption {
	return func(t *TunnelClient) {
		t.heartbeatInterval = d
	}
}

// NewTunnelClient creates a new tunnel client.
func NewTunnelClient(endpoint *ExitEndpoint, logger *slog.Logger, opts ...TunnelClientOption) *TunnelClient {
	if logger == nil {
		logger = slog.Default()
	}
	t := &TunnelClient{
		endpoint:          endpoint,
		logger:            logger.With("component", "tunnel_client"),
		reconnectInterval: 5 * time.Second,
		heartbeatInterval: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// SetForwarder sets the entry forwarder for handling incoming data.
func (t *TunnelClient) SetForwarder(f *EntryForwarder) {
	t.forwarder = f
}

// Start starts the tunnel client with auto-reconnect.
func (t *TunnelClient) Start(ctx context.Context) error {
	t.ctx, t.cancel = context.WithCancel(ctx)

	if err := t.connect(); err != nil {
		return fmt.Errorf("initial connection failed: %w", err)
	}

	t.wg.Add(2)
	go t.readLoop()
	go t.heartbeatLoop()

	return nil
}

// Stop stops the tunnel client.
func (t *TunnelClient) Stop() error {
	if t.cancel != nil {
		t.cancel()
	}
	if t.conn != nil {
		t.conn.Close()
	}
	t.wg.Wait()
	t.logger.Info("tunnel client stopped")
	return nil
}

// SendMessage sends a message through the tunnel.
func (t *TunnelClient) SendMessage(msg *TunnelMessage) error {
	data, err := msg.Encode()
	if err != nil {
		return fmt.Errorf("encode message: %w", err)
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if t.conn == nil {
		return fmt.Errorf("not connected")
	}

	if err := t.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

func (t *TunnelClient) connect() error {
	url := fmt.Sprintf("ws://%s:%d/tunnel", t.endpoint.Address, t.endpoint.WsPort)
	t.logger.Info("connecting to exit agent", "url", url)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(t.ctx, url, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	t.conn = conn
	t.logger.Info("connected to exit agent")
	return nil
}

func (t *TunnelClient) reconnect() bool {
	for {
		select {
		case <-t.ctx.Done():
			return false
		case <-time.After(t.reconnectInterval):
		}

		t.logger.Info("attempting to reconnect...")
		if err := t.connect(); err != nil {
			t.logger.Error("reconnect failed", "error", err)
			continue
		}
		return true
	}
}

func (t *TunnelClient) readLoop() {
	defer t.wg.Done()

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		_, data, err := t.conn.ReadMessage()
		if err != nil {
			t.logger.Error("read error", "error", err)
			if !t.reconnect() {
				return
			}
			continue
		}

		msg, err := DecodeMessage(bytes.NewReader(data))
		if err != nil {
			t.logger.Error("decode message error", "error", err)
			continue
		}

		t.handleMessage(msg)
	}
}

func (t *TunnelClient) handleMessage(msg *TunnelMessage) {
	if t.forwarder == nil {
		return
	}

	switch msg.Type {
	case MsgData:
		t.forwarder.HandleData(msg.ConnID, msg.Payload)
	case MsgClose:
		t.forwarder.HandleClose(msg.ConnID)
	case MsgPong:
		t.logger.Debug("received pong")
	default:
		t.logger.Warn("unknown message type", "type", msg.Type)
	}
}

func (t *TunnelClient) heartbeatLoop() {
	defer t.wg.Done()

	ticker := time.NewTicker(t.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			if err := t.SendMessage(NewPingMessage()); err != nil {
				t.logger.Error("send ping failed", "error", err)
			}
		}
	}
}
