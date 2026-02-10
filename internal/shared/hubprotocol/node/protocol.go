// Package node defines the WebSocket hub protocol types for node agents.
// These types are shared between infrastructure (AgentHub) and application layers.
package node

// Node Hub message type constants.
const (
	// Agent -> Server message types.
	NodeMsgTypeStatus    = "status"
	NodeMsgTypeHeartbeat = "heartbeat"
	NodeMsgTypeEvent     = "event"

	// Server -> Agent message types.
	NodeMsgTypeCommand          = "command"
	NodeMsgTypeConfigSync       = "config_sync"
	NodeMsgTypeSubscriptionSync = "subscription_sync"
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
	NodeCmdActionReloadConfig   = "reload_config"
	NodeCmdActionRestart        = "restart"
	NodeCmdActionStop           = "stop"
	NodeCmdActionUpdate         = "update"          // Update node agent binary
	NodeCmdActionAPIURLChanged  = "api_url_changed" // API URL changed, node should reconnect
	NodeCmdActionConfigRelocate = "config_relocate" // Configuration relocated to new server
)

// NodeAPIURLChangedPayload contains the new API URL for node reconnection.
type NodeAPIURLChangedPayload struct {
	NewURL string `json:"new_url"`
	Reason string `json:"reason,omitempty"`
}

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
