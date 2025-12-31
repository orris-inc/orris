// Package dto provides data transfer objects for the node domain.
package dto

import (
	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
)

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
	NodeCmdActionUpdate       = "update" // Update node agent binary
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
	NodeSID           string          `json:"node_id"`
	Protocol          string          `json:"protocol"`
	ServerHost        string          `json:"server_host"`
	ServerPort        int             `json:"server_port"`
	EncryptionMethod  string          `json:"encryption_method,omitempty"`
	ServerKey         string          `json:"server_key,omitempty"`
	TransportProtocol string          `json:"transport_protocol"`
	Host              string          `json:"host,omitempty"`
	Path              string          `json:"path,omitempty"`
	ServiceName       string          `json:"service_name,omitempty"`
	SNI               string          `json:"sni,omitempty"`
	AllowInsecure     bool            `json:"allow_insecure"`
	Route             *RouteConfigDTO `json:"route,omitempty"`     // Routing configuration for traffic splitting
	Outbounds         []OutboundDTO   `json:"outbounds,omitempty"` // Outbound configs for nodes referenced in route rules
}

// ToNodeConfigData converts a domain node entity to NodeConfigData for Hub sync.
// This is used for WebSocket config sync messages to node agents.
// referencedNodes: nodes referenced by route rules (outbound: "node_xxx"), can be nil.
// serverKeyFunc: generates server key for each referenced node, can be nil.
func ToNodeConfigData(n *node.Node, referencedNodes []*node.Node, serverKeyFunc func(*node.Node) string) *NodeConfigData {
	if n == nil {
		return nil
	}

	config := &NodeConfigData{
		NodeSID:           n.SID(),
		ServerHost:        n.EffectiveServerAddress(),
		ServerPort:        int(n.AgentPort()),
		TransportProtocol: "tcp", // Default to TCP
		AllowInsecure:     false,
	}

	// Determine protocol type from node's protocol field
	if n.Protocol().IsShadowsocks() {
		config.Protocol = "shadowsocks"
		config.EncryptionMethod = n.EncryptionConfig().Method()

		// Generate server key for SS2022 methods
		config.ServerKey = vo.GenerateSS2022ServerKey(n.TokenHash(), config.EncryptionMethod)

		// Handle plugin configuration for Shadowsocks transport
		if n.PluginConfig() != nil {
			plugin := n.PluginConfig().Plugin()
			opts := n.PluginConfig().Opts()

			// Check if using obfs or v2ray-plugin with websocket
			if plugin == "v2ray-plugin" || plugin == "obfs" {
				if mode, ok := opts["mode"]; ok && mode == "websocket" {
					config.TransportProtocol = "ws"
				}
				if host, ok := opts["host"]; ok {
					config.Host = host
				}
				if path, ok := opts["path"]; ok {
					config.Path = path
				}
			}
		}
	} else if n.Protocol().IsTrojan() {
		config.Protocol = "trojan"

		// Extract Trojan-specific configuration
		if n.TrojanConfig() != nil {
			tc := n.TrojanConfig()
			config.TransportProtocol = tc.TransportProtocol()
			config.SNI = tc.SNI()
			config.AllowInsecure = tc.AllowInsecure()

			// Handle transport-specific fields
			switch tc.TransportProtocol() {
			case "ws":
				config.Host = tc.Host()
				config.Path = tc.Path()
			case "grpc":
				config.ServiceName = tc.Host() // In TrojanConfig, host is used as service name for gRPC
			}
		}
	}

	// Convert route configuration if present
	if n.RouteConfig() != nil {
		config.Route = ToRouteConfigDTO(n.RouteConfig())
	}

	// Convert referenced nodes to outbounds
	if len(referencedNodes) > 0 {
		config.Outbounds = ToOutboundDTOs(referencedNodes, serverKeyFunc)
	}

	return config
}
