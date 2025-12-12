// Package dto provides data transfer objects for the forward domain.
package dto

// ProbeTaskType represents the type of probe task.
type ProbeTaskType string

const (
	// ProbeTaskTypeTarget probes target reachability from agent.
	ProbeTaskTypeTarget ProbeTaskType = "target"
	// ProbeTaskTypeTunnel probes tunnel connectivity (entry to exit).
	ProbeTaskTypeTunnel ProbeTaskType = "tunnel"
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
}

// ProbeTaskResult represents the result of a single probe task.
type ProbeTaskResult struct {
	TaskID    string        `json:"task_id"`
	Type      ProbeTaskType `json:"type"`
	RuleID    string        `json:"rule_id"` // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	Success   bool          `json:"success"`
	LatencyMs int64         `json:"latency_ms"`
	Error     string        `json:"error,omitempty"`
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

// RuleProbeResponse represents the probe result for a single rule.
// For direct rules: only targetLatencyMs is set.
// For entry rules: both tunnelLatencyMs and targetLatencyMs are set.
// For chain/direct_chain rules: chainLatencies contains per-hop latencies.
type RuleProbeResponse struct {
	RuleID          string             `json:"rule_id"`   // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	RuleType        string             `json:"rule_type"` // direct, entry, chain, direct_chain
	Success         bool               `json:"success"`
	TunnelLatencyMs *int64             `json:"tunnel_latency_ms,omitempty"` // entry only: entry→exit
	TargetLatencyMs *int64             `json:"target_latency_ms,omitempty"` // agent→target
	ChainLatencies  []*ChainHopLatency `json:"chain_latencies,omitempty"`   // chain/direct_chain: per-hop latencies
	TotalLatencyMs  *int64             `json:"total_latency_ms,omitempty"`  // total round-trip
	Error           string             `json:"error,omitempty"`
}
