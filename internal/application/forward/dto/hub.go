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
)

// HubMessage is the unified WebSocket message envelope.
type HubMessage struct {
	Type      string `json:"type"`
	AgentID   uint   `json:"agent_id,omitempty"`
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
