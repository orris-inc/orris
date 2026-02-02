// Package dto provides data transfer objects for the forward domain.
package dto

// ProbeTaskType represents the type of probe task.
type ProbeTaskType string

const (
	// ProbeTaskTypeTarget probes target reachability from agent.
	ProbeTaskTypeTarget ProbeTaskType = "target"
	// ProbeTaskTypeTunnel probes tunnel connectivity (entry to exit) via TCP connection test.
	ProbeTaskTypeTunnel ProbeTaskType = "tunnel"
	// ProbeTaskTypeTunnelPing measures tunnel latency by sending ping/pong through the tunnel.
	ProbeTaskTypeTunnelPing ProbeTaskType = "tunnel_ping"
)

// ProbeTask represents a probe task to be executed by the agent.
type ProbeTask struct {
	ID       string        `json:"id"`
	Type     ProbeTaskType `json:"type"`
	RuleID   string        `json:"rule_id"` // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	Target   string        `json:"target"`
	Port     uint16        `json:"port"`
	Protocol string        `json:"protocol"`
	Timeout  int           `json:"timeout"` // milliseconds

	// TunnelPing specific fields
	TunnelType        string `json:"tunnel_type,omitempty"`         // "ws" or "tls"
	TunnelToken       string `json:"tunnel_token,omitempty"`        // connection token for tunnel handshake
	PingCount         int    `json:"ping_count,omitempty"`          // number of pings (default: 3)
	PingIntervalMs    int    `json:"ping_interval_ms,omitempty"`    // interval between pings in ms (default: 200)
	TunnelConnTimeout int    `json:"tunnel_conn_timeout,omitempty"` // tunnel connection timeout in ms
}

// ProbeTaskResult represents the result of a single probe task.
type ProbeTaskResult struct {
	TaskID    string        `json:"task_id"`
	Type      ProbeTaskType `json:"type"`
	RuleID    string        `json:"rule_id"` // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	Success   bool          `json:"success"`
	LatencyMs int64         `json:"latency_ms"`
	Error     string        `json:"error,omitempty"`

	// TunnelPing specific results
	MinLatencyMs int64   `json:"min_latency_ms,omitempty"` // minimum RTT
	MaxLatencyMs int64   `json:"max_latency_ms,omitempty"` // maximum RTT
	AvgLatencyMs int64   `json:"avg_latency_ms,omitempty"` // average RTT
	PacketLoss   float64 `json:"packet_loss,omitempty"`    // packet loss percentage (0-100)
	PingsSent    int     `json:"pings_sent,omitempty"`     // number of pings sent
	PingsRecv    int     `json:"pings_recv,omitempty"`     // number of pongs received
}

// ChainHopLatency represents the latency of a single hop in the chain.
type ChainHopLatency struct {
	From      string `json:"from"`            // Stripe-style prefixed agent ID (e.g., "fa_xK9mP2vL3nQ")
	To        string `json:"to"`              // Stripe-style prefixed agent ID or "target"
	LatencyMs int64  `json:"latency_ms"`      // latency in milliseconds
	Success   bool   `json:"success"`         // whether this hop probe succeeded
	Error     string `json:"error,omitempty"` // error message if failed
	Online    bool   `json:"online"`          // whether the source agent is online
}

// ExitAgentProbeResult represents the probe result for a single exit agent in entry rules.
type ExitAgentProbeResult struct {
	AgentID            string   `json:"agent_id"`                        // Stripe-style prefixed ID
	Success            bool     `json:"success"`                         // whether probe succeeded
	Error              string   `json:"error,omitempty"`                 // error message if failed
	Online             bool     `json:"online"`                          // whether exit agent is online
	TunnelLatencyMs    *int64   `json:"tunnel_latency_ms,omitempty"`     // entry→exit (avg)
	TunnelMinLatencyMs *int64   `json:"tunnel_min_latency_ms,omitempty"` // minimum tunnel RTT
	TunnelMaxLatencyMs *int64   `json:"tunnel_max_latency_ms,omitempty"` // maximum tunnel RTT
	TunnelPacketLoss   *float64 `json:"tunnel_packet_loss,omitempty"`    // packet loss percentage
	TargetLatencyMs    *int64   `json:"target_latency_ms,omitempty"`     // exit→target
	TotalLatencyMs     *int64   `json:"total_latency_ms,omitempty"`      // total: tunnel + target
}

// RuleProbeResponse represents the probe result for a single rule.
// For direct rules: only targetLatencyMs is set.
// For entry rules: both tunnelLatencyMs and targetLatencyMs are set, plus exitAgentResults for multi-exit rules.
// For chain/direct_chain rules: chainLatencies contains per-hop latencies.
type RuleProbeResponse struct {
	RuleID          string             `json:"rule_id"`   // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	RuleType        string             `json:"rule_type"` // direct, entry, chain, direct_chain
	Success         bool               `json:"success"`
	TunnelLatencyMs *int64             `json:"tunnel_latency_ms,omitempty"` // entry only: entry→exit (avg, best exit agent)
	TargetLatencyMs *int64             `json:"target_latency_ms,omitempty"` // agent→target (best exit agent)
	ChainLatencies  []*ChainHopLatency `json:"chain_latencies,omitempty"`   // chain/direct_chain: per-hop latencies
	TotalLatencyMs  *int64             `json:"total_latency_ms,omitempty"`  // total round-trip (best exit agent)
	Error           string             `json:"error,omitempty"`

	// TunnelPing detailed results (when tunnel_ping probe is used, best exit agent)
	TunnelMinLatencyMs *int64   `json:"tunnel_min_latency_ms,omitempty"` // minimum tunnel RTT
	TunnelMaxLatencyMs *int64   `json:"tunnel_max_latency_ms,omitempty"` // maximum tunnel RTT
	TunnelPacketLoss   *float64 `json:"tunnel_packet_loss,omitempty"`    // tunnel packet loss percentage

	// Per-exit-agent results for entry rules with multiple exit agents
	ExitAgentResults []*ExitAgentProbeResult `json:"exit_agent_results,omitempty"` // entry: all exit agents probe results
}
