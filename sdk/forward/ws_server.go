package forward

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// TunnelServer is a WebSocket tunnel server for Exit agents.
// It accepts connections from Entry agents and forwards data to targets.
type TunnelServer struct {
	port   uint16
	logger *slog.Logger

	server   *http.Server
	upgrader websocket.Upgrader

	forwarderMu sync.RWMutex
	forwarders  map[uint]*ExitForwarder // ruleID -> forwarder

	connMu sync.RWMutex
	conns  map[*websocket.Conn]struct{}

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewTunnelServer creates a new tunnel server.
func NewTunnelServer(port uint16, logger *slog.Logger) *TunnelServer {
	if logger == nil {
		logger = slog.Default()
	}
	return &TunnelServer{
		port:       port,
		logger:     logger.With("component", "tunnel_server"),
		forwarders: make(map[uint]*ExitForwarder),
		conns:      make(map[*websocket.Conn]struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for internal use
			},
		},
	}
}

// AddForwarder adds an exit forwarder.
func (s *TunnelServer) AddForwarder(forwarder *ExitForwarder) {
	s.forwarderMu.Lock()
	s.forwarders[forwarder.RuleID()] = forwarder
	s.forwarderMu.Unlock()
}

// RemoveForwarder removes an exit forwarder.
func (s *TunnelServer) RemoveForwarder(ruleID uint) {
	s.forwarderMu.Lock()
	delete(s.forwarders, ruleID)
	s.forwarderMu.Unlock()
}

// Start starts the tunnel server.
func (s *TunnelServer) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/tunnel", s.handleTunnel)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.Info("tunnel server started", "port", s.port)
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			s.logger.Error("server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the tunnel server.
func (s *TunnelServer) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}

	// Close all WebSocket connections
	s.connMu.Lock()
	for conn := range s.conns {
		conn.Close()
	}
	s.conns = make(map[*websocket.Conn]struct{})
	s.connMu.Unlock()

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
	}

	s.wg.Wait()
	s.logger.Info("tunnel server stopped")
	return nil
}

func (s *TunnelServer) handleTunnel(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("upgrade failed", "error", err)
		return
	}

	s.connMu.Lock()
	s.conns[conn] = struct{}{}
	s.connMu.Unlock()

	s.logger.Info("entry agent connected", "remote", r.RemoteAddr)

	// Create a sender for this connection
	sender := &connSender{conn: conn}

	// Set sender for all forwarders
	s.forwarderMu.RLock()
	for _, f := range s.forwarders {
		f.tunnel = sender
	}
	s.forwarderMu.RUnlock()

	defer func() {
		s.connMu.Lock()
		delete(s.conns, conn)
		s.connMu.Unlock()
		conn.Close()
		s.logger.Info("entry agent disconnected", "remote", r.RemoteAddr)
	}()

	s.readLoop(conn)
}

func (s *TunnelServer) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				s.logger.Error("read error", "error", err)
			}
			return
		}

		msg, err := DecodeMessage(bytes.NewReader(data))
		if err != nil {
			s.logger.Error("decode message error", "error", err)
			continue
		}

		s.handleMessage(conn, msg)
	}
}

func (s *TunnelServer) handleMessage(conn *websocket.Conn, msg *TunnelMessage) {
	// For now, we route all messages to the first forwarder
	// In a multi-rule scenario, we'd need to include rule ID in the protocol
	s.forwarderMu.RLock()
	var forwarder *ExitForwarder
	for _, f := range s.forwarders {
		forwarder = f
		break
	}
	s.forwarderMu.RUnlock()

	if forwarder == nil {
		s.logger.Warn("no forwarder available")
		return
	}

	switch msg.Type {
	case MsgConnect:
		forwarder.HandleConnect(msg.ConnID)
	case MsgData:
		forwarder.HandleData(msg.ConnID, msg.Payload)
	case MsgClose:
		forwarder.HandleClose(msg.ConnID)
	case MsgPing:
		sender := &connSender{conn: conn}
		sender.SendMessage(NewPongMessage())
	default:
		s.logger.Warn("unknown message type", "type", msg.Type)
	}
}

// connSender implements TunnelSender for a WebSocket connection.
type connSender struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (s *connSender) SendMessage(msg *TunnelMessage) error {
	data, err := msg.Encode()
	if err != nil {
		return fmt.Errorf("encode message: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}
