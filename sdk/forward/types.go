// Package forward provides a Go SDK for interacting with the Orris Forward Agent API.
package forward

// RuleType represents the type of forward rule.
type RuleType string

const (
	// RuleTypeDirect forwards traffic directly to the target.
	RuleTypeDirect RuleType = "direct"
	// RuleTypeEntry is the entry point that forwards traffic to exit agent via WS tunnel.
	RuleTypeEntry RuleType = "entry"
	// RuleTypeExit receives traffic from entry agent and forwards to the target.
	RuleTypeExit RuleType = "exit"
)

// Rule represents a forward rule returned by the API.
type Rule struct {
	ID            uint     `json:"id"`
	AgentID       uint     `json:"agent_id"`
	RuleType      RuleType `json:"rule_type"`
	ExitAgentID   uint     `json:"exit_agent_id,omitempty"`
	WsListenPort  uint16   `json:"ws_listen_port,omitempty"`
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
}

// IsDirect returns true if this is a direct forward rule.
func (r *Rule) IsDirect() bool {
	return r.RuleType == RuleTypeDirect
}

// IsEntry returns true if this is an entry rule.
func (r *Rule) IsEntry() bool {
	return r.RuleType == RuleTypeEntry
}

// IsExit returns true if this is an exit rule.
func (r *Rule) IsExit() bool {
	return r.RuleType == RuleTypeExit
}

// ExitEndpoint represents the connection information for an exit agent.
type ExitEndpoint struct {
	Address string `json:"address"`
	WsPort  uint16 `json:"ws_port"`
}

// TrafficItem represents traffic data for a single rule.
type TrafficItem struct {
	RuleID        uint  `json:"rule_id"`
	UploadBytes   int64 `json:"upload_bytes"`
	DownloadBytes int64 `json:"download_bytes"`
}

// TrafficReportResult represents the result of a traffic report.
type TrafficReportResult struct {
	RulesUpdated int `json:"rules_updated"`
	RulesFailed  int `json:"rules_failed"`
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
	ActiveRules       int                  `json:"active_rules"`
	ActiveConnections int                  `json:"active_connections"`
	TunnelStatus      map[uint]TunnelState `json:"tunnel_status,omitempty"`
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
	RuleID   uint          `json:"rule_id"`
	Target   string        `json:"target"`
	Port     uint16        `json:"port"`
	Protocol string        `json:"protocol"` // always "tcp"
	Timeout  int           `json:"timeout"`  // milliseconds
}

// ProbeTaskResult represents the result of a probe task execution.
type ProbeTaskResult struct {
	TaskID    string        `json:"task_id"`
	Type      ProbeTaskType `json:"type"`
	RuleID    uint          `json:"rule_id"`
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
