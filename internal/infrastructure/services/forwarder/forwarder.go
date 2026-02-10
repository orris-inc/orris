// Package forwarder provides TCP/UDP port forwarding functionality.
package forwarder

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/infrastructure/services/protocol"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// udpBufPool reuses 64KB buffers for UDP read operations to reduce GC pressure.
var udpBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 65535)
		return &buf
	},
}

const (
	// DefaultBufferSize is the default buffer size for data transfer.
	DefaultBufferSize = 32 * 1024 // 32KB

	// DefaultUDPTimeout is the default timeout for UDP sessions.
	DefaultUDPTimeout = 60 * time.Second

	// DefaultMaxTCPConnsPerRule is the default maximum number of concurrent TCP connections per forwarding rule.
	DefaultMaxTCPConnsPerRule = 1024

	// tcpConnsWarnThreshold is the fraction of max capacity at which a warning is logged.
	tcpConnsWarnThreshold = 0.8
)

// TrafficRecorder records traffic statistics.
type TrafficRecorder interface {
	RecordTraffic(ruleID uint, upload, download int64)
}

// Manager manages TCP/UDP forwarding rules.
type Manager struct {
	mu              sync.RWMutex
	rules           map[uint]*ForwardingRule
	repo            forward.Repository
	trafficRecorder TrafficRecorder
	logger          logger.Interface
	maxTCPConns     int // max concurrent TCP connections per rule; 0 means use DefaultMaxTCPConnsPerRule
}

// ForwardingRule represents an active forwarding rule.
type ForwardingRule struct {
	ID               uint
	ListenPort       uint16
	TargetAddress    string
	TargetPort       uint16
	Protocol         string
	cancel           context.CancelFunc
	tcpListener      net.Listener
	udpConn          *net.UDPConn
	uploadBytes      atomic.Int64
	downloadBytes    atomic.Int64
	running          atomic.Bool
	blockedProtocols map[string]struct{} // protocols to block (O(1) lookup)
	sniffer          *protocol.Sniffer
	tcpSemaphore     chan struct{} // limits concurrent TCP connections
}

// NewManager creates a new forwarding manager.
func NewManager(repo forward.Repository, logger logger.Interface) *Manager {
	return &Manager{
		rules:  make(map[uint]*ForwardingRule),
		repo:   repo,
		logger: logger,
	}
}

// SetTrafficRecorder sets the traffic recorder.
// This method is thread-safe.
func (m *Manager) SetTrafficRecorder(recorder TrafficRecorder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trafficRecorder = recorder
}

// SetMaxTCPConns sets the maximum number of concurrent TCP connections per rule.
// This only affects rules started after this call. Pass 0 to use the default.
func (m *Manager) SetMaxTCPConns(max int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxTCPConns = max
}

// effectiveMaxTCPConns returns the configured max or the default.
// Caller must hold at least a read lock.
func (m *Manager) effectiveMaxTCPConns() int {
	if m.maxTCPConns > 0 {
		return m.maxTCPConns
	}
	return DefaultMaxTCPConnsPerRule
}

// Start starts forwarding for a rule.
func (m *Manager) Start(ruleID uint, listenPort uint16, targetAddress string, targetPort uint16, protocolType string, blockedProtocols []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already running
	if rule, exists := m.rules[ruleID]; exists && rule.running.Load() {
		return fmt.Errorf("forwarding rule %d is already running", ruleID)
	}

	maxConns := m.effectiveMaxTCPConns()
	ctx, cancel := context.WithCancel(context.Background())
	rule := &ForwardingRule{
		ID:            ruleID,
		ListenPort:    listenPort,
		TargetAddress: targetAddress,
		TargetPort:    targetPort,
		Protocol:      protocolType,
		cancel:        cancel,
		tcpSemaphore:  make(chan struct{}, maxConns),
	}

	// Initialize blocked protocols map for O(1) lookup
	if len(blockedProtocols) > 0 {
		rule.blockedProtocols = make(map[string]struct{}, len(blockedProtocols))
		for _, p := range blockedProtocols {
			rule.blockedProtocols[p] = struct{}{}
		}
		rule.sniffer = protocol.NewSniffer()
	}

	target := net.JoinHostPort(targetAddress, fmt.Sprintf("%d", targetPort))

	// Start TCP forwarding
	if protocolType == "tcp" || protocolType == "both" {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
		if err != nil {
			cancel()
			return fmt.Errorf("failed to listen on TCP port %d: %w", listenPort, err)
		}
		rule.tcpListener = listener
		goroutine.SafeGo(m.logger, "forwarder-handle-tcp", func() {
			m.handleTCP(ctx, rule, target)
		})
	}

	// Start UDP forwarding
	if protocolType == "udp" || protocolType == "both" {
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", listenPort))
		if err != nil {
			if rule.tcpListener != nil {
				rule.tcpListener.Close()
			}
			cancel()
			return fmt.Errorf("failed to resolve UDP address: %w", err)
		}

		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			if rule.tcpListener != nil {
				rule.tcpListener.Close()
			}
			cancel()
			return fmt.Errorf("failed to listen on UDP port %d: %w", listenPort, err)
		}
		rule.udpConn = conn
		goroutine.SafeGo(m.logger, "forwarder-handle-udp", func() {
			m.handleUDP(ctx, rule, target)
		})
	}

	rule.running.Store(true)
	m.rules[ruleID] = rule

	m.logger.Infow("forwarding started",
		"rule_id", ruleID,
		"listen_port", listenPort,
		"target", target,
		"protocol", protocolType)

	return nil
}

// Stop stops forwarding for a rule.
func (m *Manager) Stop(ruleID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rule, exists := m.rules[ruleID]
	if !exists {
		return fmt.Errorf("forwarding rule %d not found", ruleID)
	}

	if !rule.running.Load() {
		return nil
	}

	// Cancel context to stop all goroutines
	rule.cancel()

	// Close listeners
	if rule.tcpListener != nil {
		rule.tcpListener.Close()
	}
	if rule.udpConn != nil {
		rule.udpConn.Close()
	}

	rule.running.Store(false)

	// Flush traffic stats
	if m.trafficRecorder != nil {
		upload := rule.uploadBytes.Load()
		download := rule.downloadBytes.Load()
		if upload > 0 || download > 0 {
			m.trafficRecorder.RecordTraffic(ruleID, upload, download)
		}
	}

	m.logger.Infow("forwarding stopped", "rule_id", ruleID)
	return nil
}

// IsRunning checks if a rule is currently forwarding.
func (m *Manager) IsRunning(ruleID uint) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rule, exists := m.rules[ruleID]
	if !exists {
		return false
	}
	return rule.running.Load()
}

// GetStats returns the traffic stats for a rule.
func (m *Manager) GetStats(ruleID uint) (upload, download int64, running bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rule, exists := m.rules[ruleID]
	if !exists {
		return 0, 0, false
	}
	return rule.uploadBytes.Load(), rule.downloadBytes.Load(), rule.running.Load()
}

// StopAll stops all forwarding rules.
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for ruleID, rule := range m.rules {
		if rule.running.Load() {
			rule.cancel()
			if rule.tcpListener != nil {
				rule.tcpListener.Close()
			}
			if rule.udpConn != nil {
				rule.udpConn.Close()
			}
			rule.running.Store(false)

			// Flush traffic stats
			if m.trafficRecorder != nil {
				upload := rule.uploadBytes.Load()
				download := rule.downloadBytes.Load()
				if upload > 0 || download > 0 {
					m.trafficRecorder.RecordTraffic(ruleID, upload, download)
				}
			}
		}
	}

	m.logger.Infow("all forwarding rules stopped")
}

// handleTCP handles TCP forwarding.
func (m *Manager) handleTCP(ctx context.Context, rule *ForwardingRule, target string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := rule.tcpListener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				m.logger.Warnw("TCP accept error", "rule_id", rule.ID, "error", err)
				continue
			}
		}

		select {
		case rule.tcpSemaphore <- struct{}{}:
			// Check if connections reached the warning threshold (80% capacity).
			active := len(rule.tcpSemaphore)
			maxConns := cap(rule.tcpSemaphore)
			if active >= int(float64(maxConns)*tcpConnsWarnThreshold) {
				m.logger.Warnw("TCP connections approaching limit",
					"rule_id", rule.ID,
					"active", active,
					"max", maxConns,
				)
			}
			goroutine.SafeGo(m.logger, "forwarder-handle-tcp-connection", func() {
				defer func() { <-rule.tcpSemaphore }()
				m.handleTCPConnection(ctx, rule, conn, target)
			})
		default:
			// Max connections reached, reject the incoming connection.
			m.logger.Warnw("max TCP connections reached, rejecting",
				"rule_id", rule.ID,
				"max", cap(rule.tcpSemaphore),
			)
			conn.Close()
		}
	}
}

// handleTCPConnection handles a single TCP connection.
func (m *Manager) handleTCPConnection(ctx context.Context, rule *ForwardingRule, clientConn net.Conn, target string) {
	defer clientConn.Close()

	var conn net.Conn = clientConn

	// Protocol filtering
	if rule.sniffer != nil && len(rule.blockedProtocols) > 0 {
		info, peekedConn, err := rule.sniffer.Sniff(clientConn)
		if err != nil {
			// Sniff failed - data may have been partially read, connection is corrupted
			m.logger.Warnw("protocol sniff failed, closing connection",
				"rule_id", rule.ID,
				"error", err)
			return
		}

		// Check if protocol is blocked (O(1) map lookup)
		if _, blocked := rule.blockedProtocols[string(info.Protocol)]; blocked {
			m.logger.Infow("blocked protocol detected",
				"rule_id", rule.ID,
				"protocol", info.Protocol,
				"client", clientConn.RemoteAddr().String())
			return // close connection
		}
		conn = peekedConn
	}

	// Connect to target
	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		m.logger.Warnw("TCP dial target error", "rule_id", rule.ID, "target", target, "error", err)
		return
	}
	defer targetConn.Close()

	// Create a buffered channel to prevent goroutine leaks.
	// Buffer size of 2 ensures both goroutines can send without blocking,
	// even if the receiver exits early due to context cancellation.
	done := make(chan struct{}, 2)

	// Copy data bidirectionally
	goroutine.SafeGo(m.logger, "forwarder-tcp-upload", func() {
		defer func() { done <- struct{}{} }()
		n, _ := io.Copy(targetConn, conn)
		rule.uploadBytes.Add(n)
	})

	goroutine.SafeGo(m.logger, "forwarder-tcp-download", func() {
		defer func() { done <- struct{}{} }()
		n, _ := io.Copy(conn, targetConn)
		rule.downloadBytes.Add(n)
	})

	// Wait for context cancellation or both goroutines to finish
	select {
	case <-ctx.Done():
		// Close connections to unblock io.Copy goroutines
		clientConn.Close()
		targetConn.Close()
		// Drain the done channel to ensure goroutines complete
		<-done
		<-done
		return
	case <-done:
		<-done
	}
}

// handleUDP handles UDP forwarding.
func (m *Manager) handleUDP(ctx context.Context, rule *ForwardingRule, target string) {
	type udpSession struct {
		targetConn *net.UDPConn
		lastActive time.Time
	}

	sessions := make(map[string]*udpSession)
	var sessionsMu sync.Mutex

	// Cleanup old sessions periodically
	goroutine.SafeGo(m.logger, "forwarder-udp-session-cleanup", func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sessionsMu.Lock()
				now := time.Now()
				for key, session := range sessions {
					if now.Sub(session.lastActive) > DefaultUDPTimeout {
						session.targetConn.Close()
						delete(sessions, key)
					}
				}
				sessionsMu.Unlock()
			}
		}
	})

	targetAddr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		m.logger.Errorw("UDP resolve target error", "rule_id", rule.ID, "target", target, "error", err)
		return
	}

	bufPtr := udpBufPool.Get().(*[]byte)
	buf := *bufPtr
	defer udpBufPool.Put(bufPtr)
	for {
		select {
		case <-ctx.Done():
			// Cleanup all sessions
			sessionsMu.Lock()
			for _, session := range sessions {
				session.targetConn.Close()
			}
			sessionsMu.Unlock()
			return
		default:
		}

		rule.udpConn.SetReadDeadline(time.Now().Add(time.Second))
		n, clientAddr, err := rule.udpConn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			select {
			case <-ctx.Done():
				return
			default:
				m.logger.Warnw("UDP read error", "rule_id", rule.ID, "error", err)
				continue
			}
		}

		rule.uploadBytes.Add(int64(n))

		clientKey := clientAddr.String()
		sessionsMu.Lock()
		session, exists := sessions[clientKey]
		if !exists {
			// Create new session
			targetConn, err := net.DialUDP("udp", nil, targetAddr)
			if err != nil {
				sessionsMu.Unlock()
				m.logger.Warnw("UDP dial target error", "rule_id", rule.ID, "target", target, "error", err)
				continue
			}

			session = &udpSession{
				targetConn: targetConn,
				lastActive: time.Now(),
			}
			sessions[clientKey] = session

			// Start response handler for this session
			s := session
			cAddr := clientAddr
			goroutine.SafeGo(m.logger, "forwarder-udp-response-handler", func() {
				respBufPtr := udpBufPool.Get().(*[]byte)
				respBuf := *respBufPtr
				defer udpBufPool.Put(respBufPtr)
				for {
					select {
					case <-ctx.Done():
						return
					default:
					}

					s.targetConn.SetReadDeadline(time.Now().Add(DefaultUDPTimeout))
					n, err := s.targetConn.Read(respBuf)
					if err != nil {
						if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
							sessionsMu.Lock()
							if time.Since(s.lastActive) > DefaultUDPTimeout {
								s.targetConn.Close()
								delete(sessions, cAddr.String())
								sessionsMu.Unlock()
								return
							}
							sessionsMu.Unlock()
							continue
						}
						return
					}

					rule.downloadBytes.Add(int64(n))
					rule.udpConn.WriteToUDP(respBuf[:n], cAddr)

					sessionsMu.Lock()
					s.lastActive = time.Now()
					sessionsMu.Unlock()
				}
			})
		}

		session.lastActive = time.Now()
		sessionsMu.Unlock()

		// Forward to target
		session.targetConn.Write(buf[:n])
	}
}

// StartEnabledRules starts all enabled rules from the database.
// Note: BlockedProtocols is now managed at the Agent level, not per-rule.
// This method starts rules without protocol blocking. For protocol blocking,
// use Start() directly with the agent's blocked protocols list.
func (m *Manager) StartEnabledRules(ctx context.Context) error {
	rules, err := m.repo.ListEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to list enabled rules: %w", err)
	}

	for _, rule := range rules {
		if err := m.Start(
			rule.ID(),
			rule.ListenPort(),
			rule.TargetAddress(),
			rule.TargetPort(),
			rule.Protocol().String(),
			nil, // BlockedProtocols now handled at agent level
		); err != nil {
			m.logger.Warnw("failed to start forwarding rule",
				"rule_id", rule.ID(),
				"error", err)
		}
	}

	return nil
}
