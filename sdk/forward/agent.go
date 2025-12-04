package forward

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Forwarder is an interface for all forwarder types.
type Forwarder interface {
	Start(ctx context.Context) error
	Stop() error
	Traffic() *TrafficCounter
	RuleID() uint
}

// Agent is the main forward agent that manages all forwarding rules.
type Agent struct {
	client *Client
	logger *slog.Logger

	// Configuration
	pollInterval      time.Duration
	trafficInterval   time.Duration
	reconnectInterval time.Duration
	heartbeatInterval time.Duration

	// State
	forwardersMu sync.RWMutex
	forwarders   map[uint]Forwarder // ruleID -> forwarder

	tunnelsMu sync.RWMutex
	tunnels   map[uint]*TunnelClient // exitAgentID -> tunnel

	tunnelServer *TunnelServer

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// AgentOption configures Agent.
type AgentOption func(*Agent)

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) AgentOption {
	return func(a *Agent) {
		a.logger = logger
	}
}

// WithPollInterval sets the rule polling interval.
func WithPollInterval(d time.Duration) AgentOption {
	return func(a *Agent) {
		a.pollInterval = d
	}
}

// WithTrafficReportInterval sets the traffic reporting interval.
func WithTrafficReportInterval(d time.Duration) AgentOption {
	return func(a *Agent) {
		a.trafficInterval = d
	}
}

// WithAgentReconnectInterval sets the reconnect interval for tunnels.
func WithAgentReconnectInterval(d time.Duration) AgentOption {
	return func(a *Agent) {
		a.reconnectInterval = d
	}
}

// WithAgentHeartbeatInterval sets the heartbeat interval for tunnels.
func WithAgentHeartbeatInterval(d time.Duration) AgentOption {
	return func(a *Agent) {
		a.heartbeatInterval = d
	}
}

// NewAgent creates a new forward agent.
func NewAgent(baseURL, token string, opts ...AgentOption) *Agent {
	a := &Agent{
		client:            NewClient(baseURL, token),
		logger:            slog.Default(),
		pollInterval:      30 * time.Second,
		trafficInterval:   60 * time.Second,
		reconnectInterval: 5 * time.Second,
		heartbeatInterval: 30 * time.Second,
		forwarders:        make(map[uint]Forwarder),
		tunnels:           make(map[uint]*TunnelClient),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Run starts the agent and blocks until context is cancelled.
func (a *Agent) Run(ctx context.Context) error {
	a.ctx, a.cancel = context.WithCancel(ctx)

	a.logger.Info("forward agent starting")

	// Initial rule sync
	if err := a.syncRules(); err != nil {
		return fmt.Errorf("initial rule sync: %w", err)
	}

	// Start background tasks
	a.wg.Add(2)
	go a.pollRulesLoop()
	go a.reportTrafficLoop()

	a.logger.Info("forward agent started")

	// Wait for context cancellation
	<-a.ctx.Done()

	a.logger.Info("forward agent stopping")
	a.stopAll()
	a.wg.Wait()
	a.logger.Info("forward agent stopped")

	return nil
}

// Stop stops the agent.
func (a *Agent) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *Agent) pollRulesLoop() {
	defer a.wg.Done()

	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			if err := a.syncRules(); err != nil {
				a.logger.Error("sync rules failed", "error", err)
			}
		}
	}
}

func (a *Agent) reportTrafficLoop() {
	defer a.wg.Done()

	ticker := time.NewTicker(a.trafficInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			// Final traffic report
			a.reportTraffic()
			return
		case <-ticker.C:
			a.reportTraffic()
		}
	}
}

func (a *Agent) syncRules() error {
	rules, err := a.client.GetRules(a.ctx)
	if err != nil {
		return fmt.Errorf("get rules: %w", err)
	}

	a.logger.Debug("syncing rules", "count", len(rules))

	// Build map of current rules
	ruleMap := make(map[uint]*Rule)
	for i := range rules {
		ruleMap[rules[i].ID] = &rules[i]
	}

	// Stop forwarders for removed rules
	a.forwardersMu.Lock()
	for ruleID, f := range a.forwarders {
		if _, exists := ruleMap[ruleID]; !exists {
			a.logger.Info("stopping forwarder for removed rule", "rule_id", ruleID)
			f.Stop()
			delete(a.forwarders, ruleID)
		}
	}
	a.forwardersMu.Unlock()

	// Start forwarders for new rules
	for _, rule := range rules {
		a.forwardersMu.RLock()
		_, exists := a.forwarders[rule.ID]
		a.forwardersMu.RUnlock()

		if !exists {
			r := rule // Copy to avoid loop variable capture
			if err := a.startForwarder(&r); err != nil {
				a.logger.Error("start forwarder failed", "rule_id", rule.ID, "error", err)
			}
		}
	}

	return nil
}

func (a *Agent) startForwarder(rule *Rule) error {
	var forwarder Forwarder

	switch rule.RuleType {
	case RuleTypeDirect:
		f := NewDirectForwarder(rule, a.logger)
		if err := f.Start(a.ctx); err != nil {
			return err
		}
		forwarder = f

	case RuleTypeEntry:
		// Get or create tunnel for exit agent
		tunnel, err := a.getOrCreateTunnel(rule.ExitAgentID)
		if err != nil {
			return fmt.Errorf("create tunnel: %w", err)
		}

		f := NewEntryForwarder(rule, tunnel, a.logger)
		tunnel.SetForwarder(f)
		if err := f.Start(a.ctx); err != nil {
			return err
		}
		forwarder = f

	case RuleTypeExit:
		// Get or create tunnel server
		if a.tunnelServer == nil {
			a.tunnelServer = NewTunnelServer(rule.WsListenPort, a.logger)
			if err := a.tunnelServer.Start(a.ctx); err != nil {
				return fmt.Errorf("start tunnel server: %w", err)
			}
		}

		// Create exit forwarder (tunnel sender will be set when entry connects)
		f := NewExitForwarder(rule, nil, a.logger)
		a.tunnelServer.AddForwarder(f)
		if err := f.Start(a.ctx); err != nil {
			return err
		}
		forwarder = f

	default:
		return fmt.Errorf("unknown rule type: %s", rule.RuleType)
	}

	a.forwardersMu.Lock()
	a.forwarders[rule.ID] = forwarder
	a.forwardersMu.Unlock()

	a.logger.Info("forwarder started", "rule_id", rule.ID, "rule_type", rule.RuleType)
	return nil
}

func (a *Agent) getOrCreateTunnel(exitAgentID uint) (*TunnelClient, error) {
	a.tunnelsMu.Lock()
	defer a.tunnelsMu.Unlock()

	if tunnel, exists := a.tunnels[exitAgentID]; exists {
		return tunnel, nil
	}

	// Get exit endpoint from API
	endpoint, err := a.client.GetExitEndpoint(a.ctx, exitAgentID)
	if err != nil {
		return nil, fmt.Errorf("get exit endpoint: %w", err)
	}

	tunnel := NewTunnelClient(endpoint, a.logger,
		WithReconnectInterval(a.reconnectInterval),
		WithHeartbeatInterval(a.heartbeatInterval),
	)

	if err := tunnel.Start(a.ctx); err != nil {
		return nil, fmt.Errorf("start tunnel: %w", err)
	}

	a.tunnels[exitAgentID] = tunnel
	return tunnel, nil
}

func (a *Agent) reportTraffic() {
	a.forwardersMu.RLock()
	items := make([]TrafficItem, 0, len(a.forwarders))
	for _, f := range a.forwarders {
		upload, download := f.Traffic().GetAndReset()
		if upload > 0 || download > 0 {
			items = append(items, TrafficItem{
				RuleID:        f.RuleID(),
				UploadBytes:   upload,
				DownloadBytes: download,
			})
		}
	}
	a.forwardersMu.RUnlock()

	if len(items) == 0 {
		return
	}

	result, err := a.client.ReportTraffic(a.ctx, items)
	if err != nil {
		a.logger.Error("report traffic failed", "error", err)
		return
	}

	a.logger.Debug("traffic reported", "updated", result.RulesUpdated, "failed", result.RulesFailed)
}

func (a *Agent) stopAll() {
	// Stop all forwarders
	a.forwardersMu.Lock()
	for _, f := range a.forwarders {
		f.Stop()
	}
	a.forwarders = make(map[uint]Forwarder)
	a.forwardersMu.Unlock()

	// Stop all tunnels
	a.tunnelsMu.Lock()
	for _, t := range a.tunnels {
		t.Stop()
	}
	a.tunnels = make(map[uint]*TunnelClient)
	a.tunnelsMu.Unlock()

	// Stop tunnel server
	if a.tunnelServer != nil {
		a.tunnelServer.Stop()
		a.tunnelServer = nil
	}
}
