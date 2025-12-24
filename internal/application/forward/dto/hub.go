// Package dto provides data transfer objects for the forward domain.
package dto

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
	CmdActionReloadConfig = "reload_config"
	CmdActionRestartRule  = "restart_rule"
	CmdActionStopRule     = "stop_rule"
	CmdActionProbe        = "probe"
)

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
	Version     uint64         `json:"version"`
	FullSync    bool           `json:"full_sync"`
	Added       []RuleSyncData `json:"added,omitempty"`
	Updated     []RuleSyncData `json:"updated,omitempty"`
	Removed     []string       `json:"removed,omitempty"`      // Rule IDs to remove (Stripe-style prefixed, e.g., "fr_xxx")
	ClientToken string         `json:"client_token,omitempty"` // Agent's token for tunnel handshake (full sync only)
	// Note: TokenSigningSecret has been removed for security reasons.
	// Agents should verify tokens via the server API, not using local HMAC verification.
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
	TunnelType             string   `json:"tunnel_type,omitempty"`               // Tunnel type: "ws" or "tls"
	ChainAgentIDs          []string `json:"chain_agent_ids,omitempty"`
	ChainPosition          int      `json:"chain_position,omitempty"`
	IsLastInChain          bool     `json:"is_last_in_chain,omitempty"`
}

// ConfigAckData represents agent acknowledgment of config sync.
type ConfigAckData struct {
	Version uint64 `json:"version"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
