package forward

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// TrafficCounter tracks upload and download bytes.
type TrafficCounter struct {
	uploadBytes   atomic.Int64
	downloadBytes atomic.Int64
}

// AddUpload adds to upload bytes counter.
func (t *TrafficCounter) AddUpload(n int64) {
	t.uploadBytes.Add(n)
}

// AddDownload adds to download bytes counter.
func (t *TrafficCounter) AddDownload(n int64) {
	t.downloadBytes.Add(n)
}

// GetAndReset returns current values and resets counters.
func (t *TrafficCounter) GetAndReset() (upload, download int64) {
	upload = t.uploadBytes.Swap(0)
	download = t.downloadBytes.Swap(0)
	return
}

// DirectForwarder handles direct forwarding (local port -> target).
type DirectForwarder struct {
	rule     *Rule
	listener net.Listener
	traffic  *TrafficCounter
	logger   *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewDirectForwarder creates a new direct forwarder.
func NewDirectForwarder(rule *Rule, logger *slog.Logger) *DirectForwarder {
	if logger == nil {
		logger = slog.Default()
	}
	return &DirectForwarder{
		rule:    rule,
		traffic: &TrafficCounter{},
		logger:  logger.With("rule_id", rule.ID, "rule_type", "direct"),
	}
}

// Start starts the direct forwarder.
func (f *DirectForwarder) Start(ctx context.Context) error {
	f.ctx, f.cancel = context.WithCancel(ctx)

	addr := fmt.Sprintf(":%d", f.rule.ListenPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	f.listener = listener

	f.logger.Info("direct forwarder started", "listen_port", f.rule.ListenPort,
		"target", fmt.Sprintf("%s:%d", f.rule.TargetAddress, f.rule.TargetPort))

	f.wg.Add(1)
	go f.acceptLoop()

	return nil
}

// Stop stops the direct forwarder.
func (f *DirectForwarder) Stop() error {
	if f.cancel != nil {
		f.cancel()
	}
	if f.listener != nil {
		f.listener.Close()
	}
	f.wg.Wait()
	f.logger.Info("direct forwarder stopped")
	return nil
}

// Traffic returns the traffic counter.
func (f *DirectForwarder) Traffic() *TrafficCounter {
	return f.traffic
}

// RuleID returns the rule ID.
func (f *DirectForwarder) RuleID() uint {
	return f.rule.ID
}

func (f *DirectForwarder) acceptLoop() {
	defer f.wg.Done()

	for {
		conn, err := f.listener.Accept()
		if err != nil {
			select {
			case <-f.ctx.Done():
				return
			default:
				f.logger.Error("accept error", "error", err)
				continue
			}
		}

		f.wg.Add(1)
		go f.handleConn(conn)
	}
}

func (f *DirectForwarder) handleConn(clientConn net.Conn) {
	defer f.wg.Done()
	defer clientConn.Close()

	targetAddr := fmt.Sprintf("%s:%d", f.rule.TargetAddress, f.rule.TargetPort)
	targetConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		f.logger.Error("dial target failed", "target", targetAddr, "error", err)
		return
	}
	defer targetConn.Close()

	f.logger.Debug("connection established",
		"client", clientConn.RemoteAddr(),
		"target", targetAddr)

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Target (upload)
	go func() {
		defer wg.Done()
		n, _ := io.Copy(targetConn, clientConn)
		f.traffic.AddUpload(n)
		targetConn.(*net.TCPConn).CloseWrite()
	}()

	// Target -> Client (download)
	go func() {
		defer wg.Done()
		n, _ := io.Copy(clientConn, targetConn)
		f.traffic.AddDownload(n)
		clientConn.(*net.TCPConn).CloseWrite()
	}()

	wg.Wait()
}

// TunnelSender is an interface for sending data through tunnel.
type TunnelSender interface {
	SendMessage(msg *TunnelMessage) error
}

// EntryForwarder handles entry forwarding (local port -> WS tunnel -> exit agent).
type EntryForwarder struct {
	rule     *Rule
	listener net.Listener
	traffic  *TrafficCounter
	logger   *slog.Logger

	tunnel TunnelSender
	connMu sync.RWMutex
	conns  map[uint64]net.Conn // connID -> client connection

	nextConnID atomic.Uint64
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewEntryForwarder creates a new entry forwarder.
func NewEntryForwarder(rule *Rule, tunnel TunnelSender, logger *slog.Logger) *EntryForwarder {
	if logger == nil {
		logger = slog.Default()
	}
	return &EntryForwarder{
		rule:    rule,
		tunnel:  tunnel,
		traffic: &TrafficCounter{},
		conns:   make(map[uint64]net.Conn),
		logger:  logger.With("rule_id", rule.ID, "rule_type", "entry"),
	}
}

// Start starts the entry forwarder.
func (f *EntryForwarder) Start(ctx context.Context) error {
	f.ctx, f.cancel = context.WithCancel(ctx)

	addr := fmt.Sprintf(":%d", f.rule.ListenPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	f.listener = listener

	f.logger.Info("entry forwarder started", "listen_port", f.rule.ListenPort,
		"exit_agent_id", f.rule.ExitAgentID)

	f.wg.Add(1)
	go f.acceptLoop()

	return nil
}

// Stop stops the entry forwarder.
func (f *EntryForwarder) Stop() error {
	if f.cancel != nil {
		f.cancel()
	}
	if f.listener != nil {
		f.listener.Close()
	}

	// Close all client connections
	f.connMu.Lock()
	for _, conn := range f.conns {
		conn.Close()
	}
	f.conns = make(map[uint64]net.Conn)
	f.connMu.Unlock()

	f.wg.Wait()
	f.logger.Info("entry forwarder stopped")
	return nil
}

// Traffic returns the traffic counter.
func (f *EntryForwarder) Traffic() *TrafficCounter {
	return f.traffic
}

// RuleID returns the rule ID.
func (f *EntryForwarder) RuleID() uint {
	return f.rule.ID
}

// HandleData handles data received from tunnel (exit -> entry -> client).
func (f *EntryForwarder) HandleData(connID uint64, data []byte) {
	f.connMu.RLock()
	conn, ok := f.conns[connID]
	f.connMu.RUnlock()

	if !ok {
		return
	}

	n, err := conn.Write(data)
	if err != nil {
		f.logger.Debug("write to client failed", "conn_id", connID, "error", err)
		f.closeConn(connID)
		return
	}
	f.traffic.AddDownload(int64(n))
}

// HandleClose handles close message from tunnel.
func (f *EntryForwarder) HandleClose(connID uint64) {
	f.closeConn(connID)
}

func (f *EntryForwarder) acceptLoop() {
	defer f.wg.Done()

	for {
		conn, err := f.listener.Accept()
		if err != nil {
			select {
			case <-f.ctx.Done():
				return
			default:
				f.logger.Error("accept error", "error", err)
				continue
			}
		}

		f.wg.Add(1)
		go f.handleConn(conn)
	}
}

func (f *EntryForwarder) handleConn(clientConn net.Conn) {
	defer f.wg.Done()

	connID := f.nextConnID.Add(1)

	f.connMu.Lock()
	f.conns[connID] = clientConn
	f.connMu.Unlock()

	defer f.closeConn(connID)

	f.logger.Debug("new connection", "conn_id", connID, "client", clientConn.RemoteAddr())

	// Send connect message to tunnel
	if err := f.tunnel.SendMessage(NewConnectMessage(connID)); err != nil {
		f.logger.Error("send connect message failed", "conn_id", connID, "error", err)
		return
	}

	// Read from client and send to tunnel
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-f.ctx.Done():
			return
		default:
		}

		n, err := clientConn.Read(buf)
		if err != nil {
			if err != io.EOF {
				f.logger.Debug("read from client failed", "conn_id", connID, "error", err)
			}
			return
		}

		f.traffic.AddUpload(int64(n))

		if err := f.tunnel.SendMessage(NewDataMessage(connID, buf[:n])); err != nil {
			f.logger.Error("send data message failed", "conn_id", connID, "error", err)
			return
		}
	}
}

func (f *EntryForwarder) closeConn(connID uint64) {
	f.connMu.Lock()
	conn, ok := f.conns[connID]
	if ok {
		delete(f.conns, connID)
	}
	f.connMu.Unlock()

	if ok {
		conn.Close()
		// Send close message to tunnel
		f.tunnel.SendMessage(NewCloseMessage(connID))
	}
}

// ExitForwarder handles exit forwarding (WS tunnel -> target).
type ExitForwarder struct {
	rule    *Rule
	traffic *TrafficCounter
	logger  *slog.Logger

	tunnel TunnelSender
	connMu sync.RWMutex
	conns  map[uint64]net.Conn // connID -> target connection

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewExitForwarder creates a new exit forwarder.
func NewExitForwarder(rule *Rule, tunnel TunnelSender, logger *slog.Logger) *ExitForwarder {
	if logger == nil {
		logger = slog.Default()
	}
	return &ExitForwarder{
		rule:    rule,
		tunnel:  tunnel,
		traffic: &TrafficCounter{},
		conns:   make(map[uint64]net.Conn),
		logger:  logger.With("rule_id", rule.ID, "rule_type", "exit"),
	}
}

// Start starts the exit forwarder.
func (f *ExitForwarder) Start(ctx context.Context) error {
	f.ctx, f.cancel = context.WithCancel(ctx)
	f.logger.Info("exit forwarder started",
		"target", fmt.Sprintf("%s:%d", f.rule.TargetAddress, f.rule.TargetPort))
	return nil
}

// Stop stops the exit forwarder.
func (f *ExitForwarder) Stop() error {
	if f.cancel != nil {
		f.cancel()
	}

	// Close all target connections
	f.connMu.Lock()
	for _, conn := range f.conns {
		conn.Close()
	}
	f.conns = make(map[uint64]net.Conn)
	f.connMu.Unlock()

	f.wg.Wait()
	f.logger.Info("exit forwarder stopped")
	return nil
}

// Traffic returns the traffic counter.
func (f *ExitForwarder) Traffic() *TrafficCounter {
	return f.traffic
}

// RuleID returns the rule ID.
func (f *ExitForwarder) RuleID() uint {
	return f.rule.ID
}

// HandleConnect handles connect message from tunnel.
func (f *ExitForwarder) HandleConnect(connID uint64) {
	targetAddr := fmt.Sprintf("%s:%d", f.rule.TargetAddress, f.rule.TargetPort)
	targetConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		f.logger.Error("dial target failed", "conn_id", connID, "target", targetAddr, "error", err)
		f.tunnel.SendMessage(NewCloseMessage(connID))
		return
	}

	f.connMu.Lock()
	f.conns[connID] = targetConn
	f.connMu.Unlock()

	f.logger.Debug("target connection established", "conn_id", connID, "target", targetAddr)

	// Start reading from target
	f.wg.Add(1)
	go f.readFromTarget(connID, targetConn)
}

// HandleData handles data message from tunnel (entry -> exit -> target).
func (f *ExitForwarder) HandleData(connID uint64, data []byte) {
	f.connMu.RLock()
	conn, ok := f.conns[connID]
	f.connMu.RUnlock()

	if !ok {
		return
	}

	n, err := conn.Write(data)
	if err != nil {
		f.logger.Debug("write to target failed", "conn_id", connID, "error", err)
		f.closeConn(connID)
		return
	}
	f.traffic.AddUpload(int64(n))
}

// HandleClose handles close message from tunnel.
func (f *ExitForwarder) HandleClose(connID uint64) {
	f.closeConn(connID)
}

func (f *ExitForwarder) readFromTarget(connID uint64, targetConn net.Conn) {
	defer f.wg.Done()
	defer f.closeConn(connID)

	buf := make([]byte, 32*1024)
	for {
		select {
		case <-f.ctx.Done():
			return
		default:
		}

		n, err := targetConn.Read(buf)
		if err != nil {
			if err != io.EOF {
				f.logger.Debug("read from target failed", "conn_id", connID, "error", err)
			}
			return
		}

		f.traffic.AddDownload(int64(n))

		if err := f.tunnel.SendMessage(NewDataMessage(connID, buf[:n])); err != nil {
			f.logger.Error("send data message failed", "conn_id", connID, "error", err)
			return
		}
	}
}

func (f *ExitForwarder) closeConn(connID uint64) {
	f.connMu.Lock()
	conn, ok := f.conns[connID]
	if ok {
		delete(f.conns, connID)
	}
	f.connMu.Unlock()

	if ok {
		conn.Close()
		f.tunnel.SendMessage(NewCloseMessage(connID))
	}
}
