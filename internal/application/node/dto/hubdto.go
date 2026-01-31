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

	// VLESS specific fields
	Flow        string `json:"flow,omitempty"`         // VLESS flow control (xtls-rprx-vision)
	Security    string `json:"security,omitempty"`     // Security type (none, tls, reality)
	Fingerprint string `json:"fingerprint,omitempty"`  // TLS fingerprint
	PrivateKey  string `json:"private_key,omitempty"`  // Reality private key
	PublicKey   string `json:"public_key,omitempty"`   // Reality public key
	ShortID     string `json:"short_id,omitempty"`     // Reality short ID
	SpiderX     string `json:"spider_x,omitempty"`     // Reality spider X

	// VMess specific fields
	AlterID      int  `json:"alter_id,omitempty"`      // VMess alter ID
	TLS          bool `json:"tls,omitempty"`           // Enable TLS (VMess)

	// Hysteria2 specific fields
	Password          string `json:"password,omitempty"`           // Hysteria2/TUIC password
	CongestionControl string `json:"congestion_control,omitempty"` // Congestion control algorithm
	Obfs              string `json:"obfs,omitempty"`               // Obfuscation type
	ObfsPassword      string `json:"obfs_password,omitempty"`      // Obfuscation password
	UpMbps            *int   `json:"up_mbps,omitempty"`            // Upstream bandwidth limit
	DownMbps          *int   `json:"down_mbps,omitempty"`          // Downstream bandwidth limit

	// TUIC specific fields
	UUID         string `json:"uuid,omitempty"`           // TUIC UUID
	UDPRelayMode string `json:"udp_relay_mode,omitempty"` // UDP relay mode (native, quic)
	ALPN         string `json:"alpn,omitempty"`           // ALPN protocols
	DisableSNI   bool   `json:"disable_sni,omitempty"`    // Disable SNI
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
	} else if n.Protocol().IsVLESS() {
		config.Protocol = "vless"

		// Extract VLESS-specific configuration
		if n.VLESSConfig() != nil {
			vc := n.VLESSConfig()
			config.TransportProtocol = vc.TransportType()
			config.Flow = vc.Flow()
			config.Security = vc.Security()
			config.SNI = vc.SNI()
			config.Fingerprint = vc.Fingerprint()
			config.AllowInsecure = vc.AllowInsecure()

			// Handle transport-specific fields
			switch vc.TransportType() {
			case "ws", "h2":
				config.Host = vc.Host()
				config.Path = vc.Path()
			case "grpc":
				config.ServiceName = vc.ServiceName()
			}

			// Handle Reality-specific fields
			if vc.Security() == vo.VLESSSecurityReality {
				config.PrivateKey = vc.PrivateKey()
				config.PublicKey = vc.PublicKey()
				config.ShortID = vc.ShortID()
				config.SpiderX = vc.SpiderX()
			}
		}
	} else if n.Protocol().IsVMess() {
		config.Protocol = "vmess"

		// Extract VMess-specific configuration
		if n.VMessConfig() != nil {
			vc := n.VMessConfig()
			config.TransportProtocol = vc.TransportType()
			config.AlterID = vc.AlterID()
			config.EncryptionMethod = vc.Security() // VMess security is encryption method
			config.TLS = vc.TLS()
			config.SNI = vc.SNI()
			config.AllowInsecure = vc.AllowInsecure()

			// Handle transport-specific fields
			switch vc.TransportType() {
			case "ws", "http":
				config.Host = vc.Host()
				config.Path = vc.Path()
			case "grpc":
				config.ServiceName = vc.ServiceName()
			}
		}
	} else if n.Protocol().IsHysteria2() {
		config.Protocol = "hysteria2"
		config.TransportProtocol = "udp" // Hysteria2 uses QUIC (UDP-based)

		// Extract Hysteria2-specific configuration
		if n.Hysteria2Config() != nil {
			hc := n.Hysteria2Config()
			config.Password = hc.Password()
			config.CongestionControl = hc.CongestionControl()
			config.Obfs = hc.Obfs()
			config.ObfsPassword = hc.ObfsPassword()
			config.UpMbps = hc.UpMbps()
			config.DownMbps = hc.DownMbps()
			config.SNI = hc.SNI()
			config.AllowInsecure = hc.AllowInsecure()
			config.Fingerprint = hc.Fingerprint()
		}
	} else if n.Protocol().IsTUIC() {
		config.Protocol = "tuic"
		config.TransportProtocol = "udp" // TUIC uses QUIC (UDP-based)

		// Extract TUIC-specific configuration
		if n.TUICConfig() != nil {
			tc := n.TUICConfig()
			config.UUID = tc.UUID()
			config.Password = tc.Password()
			config.CongestionControl = tc.CongestionControl()
			config.UDPRelayMode = tc.UDPRelayMode()
			config.ALPN = tc.ALPN()
			config.SNI = tc.SNI()
			config.AllowInsecure = tc.AllowInsecure()
			config.DisableSNI = tc.DisableSNI()
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

// Subscription sync change type constants.
const (
	SubscriptionChangeAdded   = "added"   // New subscription added
	SubscriptionChangeUpdated = "updated" // Subscription updated (status, expiry, etc.)
	SubscriptionChangeRemoved = "removed" // Subscription removed or expired
)

// SubscriptionSyncData represents subscription sync data for node agent.
type SubscriptionSyncData struct {
	ChangeType    string                 `json:"change_type"`             // added, updated, removed
	Subscriptions []NodeSubscriptionInfo `json:"subscriptions,omitempty"` // Affected subscriptions
	Timestamp     int64                  `json:"timestamp"`               // Unix timestamp
}
