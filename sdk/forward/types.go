// Package forward provides a Go SDK for interacting with the Orris Forward Agent API.
package forward

// RuleType represents the type of forward rule.
type RuleType string

const (
	// RuleTypeDirect forwards traffic directly to the target.
	RuleTypeDirect RuleType = "direct"
	// RuleTypeEntry is the entry point that forwards traffic to exit agent via WS tunnel.
	// The target information is configured on the entry rule and passed to the exit agent.
	RuleTypeEntry RuleType = "entry"
	// RuleTypeChain is a multi-hop chain forward rule.
	// Traffic flows through multiple agents: entry -> relay1 -> relay2 -> ... -> exit -> target.
	RuleTypeChain RuleType = "chain"
)

// Rule represents a forward rule returned by the API.
// Note: ws_listen_port field has been removed (exit type deprecated).
type Rule struct {
	ID            string   `json:"id"`       // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	AgentID       string   `json:"agent_id"` // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	RuleType      RuleType `json:"rule_type"`
	ExitAgentID   string   `json:"exit_agent_id,omitempty"` // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Name          string   `json:"name"`
	ListenPort    uint16   `json:"listen_port"`
	TargetAddress string   `json:"target_address,omitempty"`
	TargetPort    uint16   `json:"target_port,omitempty"`
	Protocol      string   `json:"protocol"`
	Status        string   `json:"status"`
	Remark        string   `json:"remark,omitempty"`
	UploadBytes   int64    `json:"upload_bytes"`
	DownloadBytes int64    `json:"download_bytes"`
	TotalBytes    int64    `json:"total_bytes"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`

	// Role indicates the requesting agent's role in this rule
	// Values: "entry" (needs to establish tunnel), "exit" (accepts tunnel connections), "relay" (chain middle node)
	Role string `json:"role,omitempty"`

	// Chain-specific fields (for chain rule type)
	ChainAgentIDs  []string `json:"chain_agent_ids,omitempty"`   // Ordered list of agent IDs in chain
	ChainPosition  int      `json:"chain_position,omitempty"`    // Agent's position in chain (0-indexed)
	IsLastInChain  bool     `json:"is_last_in_chain,omitempty"`  // True if agent is last in chain
	NextHopAgentID string   `json:"next_hop_agent_id,omitempty"` // Next agent in chain
	NextHopAddress string   `json:"next_hop_address,omitempty"`  // Next agent's public address
	NextHopWsPort  uint16   `json:"next_hop_ws_port,omitempty"`  // Next agent's WS port
}

// IsDirect returns true if this is a direct forward rule.
func (r *Rule) IsDirect() bool {
	return r.RuleType == RuleTypeDirect
}

// IsEntry returns true if this is an entry rule.
func (r *Rule) IsEntry() bool {
	return r.RuleType == RuleTypeEntry
}

// IsExit is deprecated (exit type has been removed).
func (r *Rule) IsExit() bool {
	return false
}

// IsChain returns true if this is a chain forward rule.
func (r *Rule) IsChain() bool {
	return r.RuleType == RuleTypeChain
}

// IsChainEntry returns true if this agent is the entry point of a chain rule.
func (r *Rule) IsChainEntry() bool {
	return r.RuleType == RuleTypeChain && r.ChainPosition == 0
}

// IsChainRelay returns true if this agent is a relay (middle) node in a chain rule.
func (r *Rule) IsChainRelay() bool {
	return r.RuleType == RuleTypeChain && !r.IsLastInChain && r.ChainPosition > 0
}

// IsChainExit returns true if this agent is the exit (last) node in a chain rule.
func (r *Rule) IsChainExit() bool {
	return r.RuleType == RuleTypeChain && r.IsLastInChain
}

// ExitEndpoint represents the connection information for an exit agent.
type ExitEndpoint struct {
	Address         string `json:"address"`
	WsPort          uint16 `json:"ws_port"`
	ConnectionToken string `json:"connection_token"` // Short-term JWT for agent-to-agent authentication
}

// TrafficItem represents traffic data for a single rule.
type TrafficItem struct {
	RuleID        string `json:"rule_id"` // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	UploadBytes   int64  `json:"upload_bytes"`
	DownloadBytes int64  `json:"download_bytes"`
}

// TrafficReportResult represents the result of a traffic report.
type TrafficReportResult struct {
	RulesUpdated int `json:"rules_updated"`
	RulesFailed  int `json:"rules_failed"`
	RulesDenied  int `json:"rules_denied"` // rules denied due to authorization failure
}

// apiResponse represents the standard API response structure.
type apiResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// AgentStatus represents the status data reported by a forward agent.
type AgentStatus struct {
	// System resources
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryUsed    uint64  `json:"memory_used"`
	MemoryTotal   uint64  `json:"memory_total"`
	DiskPercent   float64 `json:"disk_percent"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
	UptimeSeconds int64   `json:"uptime_seconds"`

	// Network connections
	TCPConnections int `json:"tcp_connections"`
	UDPConnections int `json:"udp_connections"`

	// Forward status
	ActiveRules       int                    `json:"active_rules"`
	ActiveConnections int                    `json:"active_connections"`
	TunnelStatus      map[string]TunnelState `json:"tunnel_status,omitempty"` // Key is Stripe-style rule ID (e.g., "fr_xK9mP2vL3nQ")

	// WebSocket tunnel configuration (for exit agent)
	// Note: WsListenPort is now reported directly by the agent in status updates,
	// not configured per-rule (exit rule type has been removed).
	WsListenPort uint16 `json:"ws_listen_port,omitempty"` // WebSocket listen port for tunnel connections
}

// TunnelState represents the connection state of a tunnel.
type TunnelState string

const (
	TunnelStateConnected    TunnelState = "connected"
	TunnelStateConnecting   TunnelState = "connecting"
	TunnelStateDisconnected TunnelState = "disconnected"
)

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
	Protocol string        `json:"protocol"` // always "tcp"
	Timeout  int           `json:"timeout"`  // milliseconds
}

// ProbeTaskResult represents the result of a probe task execution.
type ProbeTaskResult struct {
	TaskID    string        `json:"task_id"`
	Type      ProbeTaskType `json:"type"`
	RuleID    string        `json:"rule_id"` // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	Success   bool          `json:"success"`
	LatencyMs int64         `json:"latency_ms"`
	Error     string        `json:"error,omitempty"`
}

// ProbeMessage is the WebSocket message envelope for probe communication.
type ProbeMessage struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// ProbeMessageType constants for WebSocket message types.
const (
	ProbeMessageTypeTask   = "task"
	ProbeMessageTypeResult = "result"
)
