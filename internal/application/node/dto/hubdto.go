// Package dto provides data transfer objects for the node domain.
package dto

// Node Hub message type constants.
const (
	// Agent -> Server message types.
	NodeMsgTypeStatus    = "status"
	NodeMsgTypeHeartbeat = "heartbeat"
	NodeMsgTypeEvent     = "event"

	// Server -> Agent message types.
	NodeMsgTypeCommand    = "command"
	NodeMsgTypeConfigSync = "config_sync"
)

// NodeHubMessage is the unified WebSocket message envelope for node agents.
type NodeHubMessage struct {
	Type      string `json:"type"`
	NodeID    string `json:"node_id,omitempty"` // Stripe-style prefixed ID (e.g., "node_xK9mP2vL3nQ")
	Timestamp int64  `json:"timestamp"`
	Data      any    `json:"data,omitempty"`
}

// NodeCommandData represents a command to be sent to node agent.
type NodeCommandData struct {
	CommandID string `json:"command_id"`
	Action    string `json:"action"`
	Payload   any    `json:"payload,omitempty"`
}

// Node command action constants.
const (
	NodeCmdActionReloadConfig = "reload_config"
	NodeCmdActionRestart      = "restart"
	NodeCmdActionStop         = "stop"
)

// NodeEventData represents a node agent event payload.
type NodeEventData struct {
	EventType string `json:"event_type"`
	Message   string `json:"message,omitempty"`
	Extra     any    `json:"extra,omitempty"`
}

// Node event type constants.
const (
	NodeEventTypeConnected    = "connected"
	NodeEventTypeDisconnected = "disconnected"
	NodeEventTypeError        = "error"
	NodeEventTypeConfigChange = "config_change"
)

// NodeConfigSyncData represents configuration sync data for node agent.
type NodeConfigSyncData struct {
	Version   uint64          `json:"version"`
	FullSync  bool            `json:"full_sync"`
	Config    *NodeConfigData `json:"config,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// NodeConfigData represents the node configuration to sync.
type NodeConfigData struct {
	NodeSID           string `json:"node_id"`
	Protocol          string `json:"protocol"`
	ServerHost        string `json:"server_host"`
	ServerPort        int    `json:"server_port"`
	EncryptionMethod  string `json:"encryption_method,omitempty"`
	ServerKey         string `json:"server_key,omitempty"`
	TransportProtocol string `json:"transport_protocol"`
	Host              string `json:"host,omitempty"`
	Path              string `json:"path,omitempty"`
	ServiceName       string `json:"service_name,omitempty"`
	SNI               string `json:"sni,omitempty"`
	AllowInsecure     bool   `json:"allow_insecure"`
}
