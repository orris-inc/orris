// Package forward defines the WebSocket hub protocol types for forward agents.
// These types are shared between infrastructure (AgentHub) and application layers.
package forward

// Hub message type constants.
const (
	// Agent -> Server message types.
	MsgTypeStatus    = "status"
	MsgTypeHeartbeat = "heartbeat"
	MsgTypeEvent     = "event"

	// Server -> Agent message types.
	MsgTypeCommand = "command"

	// Probe message types (Forward domain specific, routed through AgentHub).
	MsgTypeProbeTask   = "probe_task"   // Server -> Agent
	MsgTypeProbeResult = "probe_result" // Agent -> Server

	// Config sync message types (Forward domain specific, routed through AgentHub).
	MsgTypeConfigSync = "config_sync" // Server -> Agent
	MsgTypeConfigAck  = "config_ack"  // Agent -> Server

	// Rule sync status message types (Forward domain specific, routed through AgentHub).
	MsgTypeRuleSyncStatus = "rule_sync_status" // Agent -> Server

	// Tunnel health report message types (Agent reports exit agent health status).
	MsgTypeTunnelHealthReport = "tunnel_health_report" // Agent -> Server
)

// HubMessage is the unified WebSocket message envelope.
type HubMessage struct {
	Type      string `json:"type"`
	AgentID   string `json:"agent_id,omitempty"` // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Timestamp int64  `json:"timestamp"`
	Data      any    `json:"data,omitempty"`
}

// CommandData represents a command to be sent to agent.
type CommandData struct {
	CommandID string `json:"command_id"`
	Action    string `json:"action"`
	Payload   any    `json:"payload,omitempty"`
}

// Command action constants.
const (
	CmdActionReloadConfig   = "reload_config"
	CmdActionRestartRule    = "restart_rule"
	CmdActionStopRule       = "stop_rule"
	CmdActionProbe          = "probe"
	CmdActionUpdate         = "update"          // Update agent binary
	CmdActionAPIURLChanged  = "api_url_changed" // API URL changed, agent should reconnect
	CmdActionConfigRelocate = "config_relocate" // Configuration relocated to new server
)

// APIURLChangedPayload contains the new API URL for agent reconnection.
type APIURLChangedPayload struct {
	NewURL string `json:"new_url"`
	Reason string `json:"reason,omitempty"`
}

// AgentEventData represents an agent event payload.
type AgentEventData struct {
	EventType string `json:"event_type"`
	Message   string `json:"message,omitempty"`
	Extra     any    `json:"extra,omitempty"`
}

// Agent event type constants.
const (
	EventTypeConnected    = "connected"
	EventTypeDisconnected = "disconnected"
	EventTypeError        = "error"
	EventTypeConfigChange = "config_change"
)

// ConfigSyncData represents incremental configuration sync data.
type ConfigSyncData struct {
	Version          uint64         `json:"version"`
	FullSync         bool           `json:"full_sync"`
	Added            []RuleSyncData `json:"added,omitempty"`
	Updated          []RuleSyncData `json:"updated,omitempty"`
	Removed          []string       `json:"removed,omitempty"`           // Rule IDs to remove (Stripe-style prefixed, e.g., "fr_xxx")
	ClientToken      string         `json:"client_token,omitempty"`      // Agent's token for tunnel handshake (full sync only)
	BlockedProtocols []string       `json:"blocked_protocols,omitempty"` // Agent-level blocked protocols (supports incremental sync)
	// Note: TokenSigningSecret has been removed for security reasons.
	// Agents should verify tokens via the server API, not using local HMAC verification.
}

// ExitAgentSyncData represents an exit agent with connection info for load balancing.
type ExitAgentSyncData struct {
	AgentID string `json:"agent_id"` // Stripe-style prefixed ID
	Weight  uint16 `json:"weight"`   // Load balancing weight (0-100, 0=backup)
	Address string `json:"address"`  // Exit agent public address
	WsPort  uint16 `json:"ws_port"`  // Exit agent WebSocket port
	TlsPort uint16 `json:"tls_port"` // Exit agent TLS port
	Online  bool   `json:"online"`   // Exit agent online status
}

// HealthCheckConfig represents health check configuration for load balancing failover.
type HealthCheckConfig struct {
	UnhealthyThreshold uint32 `json:"unhealthy_threshold"` // Number of failures before marking unhealthy (default: 2)
	HealthyThreshold   uint32 `json:"healthy_threshold"`   // Number of successes before marking healthy (default: 1)
}

// RuleSyncData represents rule sync data for config sync.
type RuleSyncData struct {
	ID                     string   `json:"id"`       // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	ShortID                string   `json:"short_id"` // Deprecated: use ID instead
	RuleType               string   `json:"rule_type"`
	ListenPort             uint16   `json:"listen_port"`
	TargetAddress          string   `json:"target_address,omitempty"`
	TargetPort             uint16   `json:"target_port,omitempty"`
	BindIP                 string   `json:"bind_ip,omitempty"` // Bind IP address for outbound connections
	Protocol               string   `json:"protocol"`
	Role                   string   `json:"role,omitempty"`
	AgentID                string   `json:"agent_id,omitempty"` // Entry agent ID (for exit agents to verify handshake)
	NextHopAgentID         string   `json:"next_hop_agent_id,omitempty"`
	NextHopAddress         string   `json:"next_hop_address,omitempty"`
	NextHopWsPort          uint16   `json:"next_hop_ws_port,omitempty"`
	NextHopTlsPort         uint16   `json:"next_hop_tls_port,omitempty"`         // Next hop TLS listen port for tunnel connections
	NextHopPort            uint16   `json:"next_hop_port,omitempty"`             // Next hop listen port (for direct_chain type)
	NextHopConnectionToken string   `json:"next_hop_connection_token,omitempty"` // Short-term token for next hop authentication
	TunnelType             string   `json:"tunnel_type,omitempty"`               // Tunnel type: "ws", "tls", "ws_smux", or "tls_smux"
	TunnelHops             *int     `json:"tunnel_hops,omitempty"`               // Number of hops using tunnel (nil=full tunnel)
	HopMode                string   `json:"hop_mode,omitempty"`                  // Hop mode: "tunnel", "direct", or "boundary"
	InboundMode            string   `json:"inbound_mode,omitempty"`              // For boundary nodes: inbound mode
	OutboundMode           string   `json:"outbound_mode,omitempty"`             // For boundary nodes: outbound mode
	ChainAgentIDs          []string `json:"chain_agent_ids,omitempty"`
	ChainPosition          int      `json:"chain_position,omitempty"`
	IsLastInChain          bool     `json:"is_last_in_chain,omitempty"`
	// Multiple exit agents for load balancing (entry rules only, mutually exclusive with NextHop* fields)
	ExitAgents          []ExitAgentSyncData `json:"exit_agents,omitempty"`
	LoadBalanceStrategy string              `json:"load_balance_strategy,omitempty"` // Load balance strategy: "failover" (default), "weighted"
	HealthCheck         *HealthCheckConfig  `json:"health_check,omitempty"`          // Health check config for load balancing failover
}

// ConfigAckData represents agent acknowledgment of config sync.
type ConfigAckData struct {
	Version uint64 `json:"version"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// TunnelHealthReport represents a tunnel health status report from entry agent.
// Entry agents periodically check tunnel connectivity to exit agents and report failures.
type TunnelHealthReport struct {
	RuleID      string `json:"rule_id"`                // Rule ID (Stripe-style prefixed, e.g., "fr_xxx")
	ExitAgentID string `json:"exit_agent_id"`          // Exit agent ID (Stripe-style prefixed, e.g., "fa_xxx")
	Healthy     bool   `json:"healthy"`                // Whether the tunnel is healthy
	FailCount   int    `json:"fail_count,omitempty"`   // Consecutive failure count (when unhealthy)
	Error       string `json:"error,omitempty"`        // Error message (when unhealthy)
	LatencyMs   *int64 `json:"latency_ms,omitempty"`   // Last measured latency in milliseconds (when healthy)
	CheckedAt   int64  `json:"checked_at"`             // Health check timestamp (Unix seconds)
}
