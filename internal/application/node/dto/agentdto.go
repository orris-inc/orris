package dto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	commondto "github.com/orris-inc/orris/internal/application/common/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/domain/subscription"
)

// AgentResponse represents the standard response format for agent API
type AgentResponse struct {
	Data interface{} `json:"data,omitempty"` // Response payload, can be any type based on endpoint
	Ret  int         `json:"ret,omitempty"`  // Return code: 1 = success, 0 = error
	Msg  string      `json:"msg,omitempty"`  // Message describing the result or error
}

// NodeConfigResponse represents node configuration data for node agents
// Compatible with sing-box inbound configuration
type NodeConfigResponse struct {
	NodeSID           string          `json:"node_id" binding:"required"`                                                  // Node SID (Stripe-style: node_xxx)
	Protocol          string          `json:"protocol" binding:"required,oneof=shadowsocks trojan vless vmess hysteria2 tuic"` // Protocol type
	ServerHost        string          `json:"server_host" binding:"required"`                                              // Server hostname or IP address
	ServerPort        int             `json:"server_port" binding:"required,min=1,max=65535"`                              // Server port number
	EncryptionMethod  string          `json:"encryption_method,omitempty"`                                                 // Encryption method for Shadowsocks
	ServerKey         string          `json:"server_key,omitempty"`                                                        // Server password for SS
	TransportProtocol string          `json:"transport_protocol,omitempty"`                                                // Transport protocol (tcp, ws, grpc, h2, http, quic)
	Host              string          `json:"host,omitempty"`                                                              // WebSocket/HTTP host header
	Path              string          `json:"path,omitempty"`                                                              // WebSocket/HTTP path
	ServiceName       string          `json:"service_name,omitempty"`                                                      // gRPC service name
	SNI               string          `json:"sni,omitempty"`                                                               // TLS Server Name Indication
	AllowInsecure     bool            `json:"allow_insecure"`                                                              // Allow insecure TLS connection
	EnableVless       bool            `json:"enable_vless"`                                                                // Enable VLESS protocol (deprecated, use Protocol=vless)
	EnableXTLS        bool            `json:"enable_xtls"`                                                                 // Enable XTLS (deprecated, use VLESSFlow)
	SpeedLimit        uint64          `json:"speed_limit"`                                                                 // Speed limit in Mbps, 0 = unlimited
	DeviceLimit       int             `json:"device_limit"`                                                                // Device connection limit, 0 = unlimited
	RuleListPath      string          `json:"rule_list_path,omitempty"`                                                    // Path to routing rule list file (deprecated, use Route)
	Route             *RouteConfigDTO `json:"route,omitempty"`                                                             // Routing configuration for traffic splitting
	Outbounds         []OutboundDTO   `json:"outbounds,omitempty"`                                                         // Outbound configs for nodes referenced in route rules

	// VLESS specific fields
	VLESSFlow             string `json:"vless_flow,omitempty"`               // VLESS flow control (xtls-rprx-vision)
	VLESSSecurity         string `json:"vless_security,omitempty"`           // VLESS security type (none, tls, reality)
	VLESSFingerprint      string `json:"vless_fingerprint,omitempty"`        // TLS fingerprint for VLESS
	VLESSRealityPublicKey string `json:"vless_reality_public_key,omitempty"` // Reality public key
	VLESSRealityShortID   string `json:"vless_reality_short_id,omitempty"`   // Reality short ID
	VLESSRealitySpiderX   string `json:"vless_reality_spider_x,omitempty"`   // Reality spider X parameter

	// VMess specific fields
	VMessAlterID  int    `json:"vmess_alter_id,omitempty"`  // VMess alter ID (usually 0)
	VMessSecurity string `json:"vmess_security,omitempty"`  // VMess security (auto, aes-128-gcm, chacha20-poly1305, none, zero)
	VMessTLS      bool   `json:"vmess_tls,omitempty"`       // VMess TLS enabled

	// Hysteria2 specific fields
	Hysteria2CongestionControl string `json:"hysteria2_congestion_control,omitempty"` // Congestion control (cubic, bbr, new_reno)
	Hysteria2Obfs              string `json:"hysteria2_obfs,omitempty"`               // Obfuscation type (salamander)
	Hysteria2ObfsPassword      string `json:"hysteria2_obfs_password,omitempty"`      // Obfuscation password
	Hysteria2UpMbps            *int   `json:"hysteria2_up_mbps,omitempty"`            // Upstream bandwidth limit in Mbps
	Hysteria2DownMbps          *int   `json:"hysteria2_down_mbps,omitempty"`          // Downstream bandwidth limit in Mbps
	Hysteria2Fingerprint       string `json:"hysteria2_fingerprint,omitempty"`        // TLS fingerprint for Hysteria2

	// TUIC specific fields
	TUICCongestionControl string `json:"tuic_congestion_control,omitempty"` // Congestion control (cubic, bbr, new_reno)
	TUICUDPRelayMode      string `json:"tuic_udp_relay_mode,omitempty"`     // UDP relay mode (native, quic)
	TUICAlpn              string `json:"tuic_alpn,omitempty"`               // ALPN protocols
	TUICDisableSNI        bool   `json:"tuic_disable_sni,omitempty"`        // Disable SNI
}

// RouteConfigDTO represents the routing configuration for sing-box
type RouteConfigDTO struct {
	Rules []RouteRuleDTO `json:"rules,omitempty"` // Ordered list of routing rules
	Final string         `json:"final"`           // Default outbound when no rules match (direct/block/proxy/node_xxx)
}

// RouteRuleDTO represents a single routing rule, compatible with sing-box route rule
type RouteRuleDTO struct {
	// Domain matching
	Domain        []string `json:"domain,omitempty"`         // Exact domain match
	DomainSuffix  []string `json:"domain_suffix,omitempty"`  // Domain suffix match
	DomainKeyword []string `json:"domain_keyword,omitempty"` // Domain keyword match
	DomainRegex   []string `json:"domain_regex,omitempty"`   // Domain regex match

	// IP matching
	IPCIDR       []string `json:"ip_cidr,omitempty"`        // Destination IP CIDR match
	SourceIPCIDR []string `json:"source_ip_cidr,omitempty"` // Source IP CIDR match
	IPIsPrivate  bool     `json:"ip_is_private,omitempty"`  // Match private/LAN IP addresses

	// GeoIP/GeoSite matching
	GeoIP   []string `json:"geoip,omitempty"`   // GeoIP country codes (cn, us, etc.)
	GeoSite []string `json:"geosite,omitempty"` // GeoSite categories (cn, google, etc.)

	// Port matching (using int instead of uint16 for JSON compatibility with sing-box)
	Port       []int `json:"port,omitempty"`        // Destination port match
	SourcePort []int `json:"source_port,omitempty"` // Source port match

	// Protocol/Network matching
	Protocol []string `json:"protocol,omitempty"` // Sniffed protocol match (http, tls, etc.)
	Network  []string `json:"network,omitempty"`  // Network type match (tcp, udp)

	// Rule set reference
	RuleSet []string `json:"rule_set,omitempty"` // Rule set references

	// Action
	Outbound string `json:"outbound"` // Action: direct/block/proxy or node SID (node_xxx)
}

// OutboundDTO represents a sing-box outbound configuration.
// Used when route rules reference other nodes as outbounds.
type OutboundDTO struct {
	Type   string `json:"type"`   // Protocol type: shadowsocks, trojan, vless, vmess, hysteria2, tuic, direct, block
	Tag    string `json:"tag"`    // Unique identifier for the outbound (node SID)
	Server string `json:"server"` // Server hostname or IP address
	Port   int    `json:"server_port"`

	// Shadowsocks specific fields
	Method     string `json:"method,omitempty"`      // Encryption method for SS
	Password   string `json:"password,omitempty"`    // Password for SS/Trojan/Hysteria2/TUIC
	Plugin     string `json:"plugin,omitempty"`      // SIP003 plugin name
	PluginOpts string `json:"plugin_opts,omitempty"` // Plugin options string

	// UUID field (for VLESS/VMess/TUIC)
	UUID string `json:"uuid,omitempty"` // User UUID for VLESS/VMess/TUIC

	// TLS fields (for Trojan/VLESS/VMess)
	TLS *OutboundTLSDTO `json:"tls,omitempty"` // TLS configuration

	// Transport fields (for Trojan/VLESS/VMess ws/grpc/h2)
	Transport *OutboundTransportDTO `json:"transport,omitempty"` // Transport configuration

	// VLESS specific fields
	VLESSFlow string `json:"flow,omitempty"` // VLESS flow control (xtls-rprx-vision)

	// VMess specific fields
	VMessAlterID  int    `json:"alter_id,omitempty"` // VMess alter ID
	VMessSecurity string `json:"security,omitempty"` // VMess encryption method

	// Hysteria2 specific fields
	Hysteria2Obfs         string `json:"obfs,omitempty"`          // Obfuscation type
	Hysteria2ObfsPassword string `json:"obfs_password,omitempty"` // Obfuscation password
	Hysteria2UpMbps       *int   `json:"up_mbps,omitempty"`       // Upstream bandwidth limit
	Hysteria2DownMbps     *int   `json:"down_mbps,omitempty"`     // Downstream bandwidth limit

	// TUIC specific fields
	TUICCongestionControl string `json:"congestion_control,omitempty"` // Congestion control algorithm
	TUICUDPRelayMode      string `json:"udp_relay_mode,omitempty"`     // UDP relay mode
}

// OutboundTLSDTO represents TLS configuration for outbound.
type OutboundTLSDTO struct {
	Enabled    bool     `json:"enabled"`               // Enable TLS
	ServerName string   `json:"server_name,omitempty"` // SNI
	Insecure   bool     `json:"insecure,omitempty"`    // Allow insecure TLS
	DisableSNI bool     `json:"disable_sni,omitempty"` // Disable SNI
	ALPN       []string `json:"alpn,omitempty"`        // ALPN protocols

	// Reality specific fields (for VLESS)
	Reality *OutboundRealityDTO `json:"reality,omitempty"` // Reality configuration
}

// OutboundRealityDTO represents Reality configuration for outbound TLS.
type OutboundRealityDTO struct {
	Enabled   bool   `json:"enabled"`              // Enable Reality
	PublicKey string `json:"public_key,omitempty"` // Reality public key
	ShortID   string `json:"short_id,omitempty"`   // Reality short ID
}

// OutboundTransportDTO represents transport configuration for outbound.
type OutboundTransportDTO struct {
	Type        string            `json:"type"`                   // Transport type: ws, grpc
	Path        string            `json:"path,omitempty"`         // WebSocket path
	Headers     map[string]string `json:"headers,omitempty"`      // Custom headers for WS
	ServiceName string            `json:"service_name,omitempty"` // gRPC service name
}

// NodeSubscriptionInfo represents individual subscription information for node access
type NodeSubscriptionInfo struct {
	SubscriptionSID string `json:"subscription_id" binding:"required"` // Subscription SID (Stripe-style: sub_xxx)
	Password        string `json:"password" binding:"required"`        // HMAC-SHA256 signed password derived from subscription UUID
	Name            string `json:"name" binding:"required"`            // User identifier for logging (sing-box compatible)
	SpeedLimit      uint64 `json:"speed_limit"`                        // Speed limit in bps (0 = unlimited)
	DeviceLimit     int    `json:"device_limit"`                       // Device connection limit (0 = unlimited)
	ExpireTime      int64  `json:"expire_time"`                        // Unix timestamp of expiration date
}

// NodeSubscriptionsResponse represents the subscription list response for a node
type NodeSubscriptionsResponse struct {
	Subscriptions []NodeSubscriptionInfo `json:"subscriptions" binding:"required"` // List of subscriptions authorized for this node
}

// SubscriptionUsageItem represents usage data for a single subscription
type SubscriptionUsageItem struct {
	SubscriptionSID string `json:"subscription_id" binding:"required"` // Subscription SID (Stripe-style: sub_xxx)
	Upload          int64  `json:"upload" binding:"min=0"`             // Upload usage in bytes
	Download        int64  `json:"download" binding:"min=0"`           // Download usage in bytes
}

// ReportSubscriptionUsageRequest represents subscription usage report request
type ReportSubscriptionUsageRequest struct {
	Subscriptions []SubscriptionUsageItem `json:"subscriptions" binding:"required,dive"` // Array of subscription usage data
}

// ReportNodeStatusRequest represents node status report request with procfs metrics.
// Uses embedded SystemStatus for common system metrics shared with Forward Agent.
type ReportNodeStatusRequest struct {
	commondto.SystemStatus
}

// OnlineSubscriptionItem represents a single online subscription connection
type OnlineSubscriptionItem struct {
	SubscriptionSID string `json:"subscription_id" binding:"required"` // Subscription SID (Stripe-style: sub_xxx)
	IP              string `json:"ip" binding:"required"`              // Connection IP address
}

// ReportOnlineSubscriptionsRequest represents online subscriptions report request
type ReportOnlineSubscriptionsRequest struct {
	Subscriptions []OnlineSubscriptionItem `json:"subscriptions" binding:"required,dive"` // Array of currently online subscriptions
}

// ToNodeConfigResponse converts a domain node entity to agent node config response.
// Supports Shadowsocks, Trojan, VLESS, VMess, Hysteria2, and TUIC protocols with sing-box compatible configuration.
// referencedNodes: nodes referenced by route rules (outbound: "node_xxx"), can be nil.
// serverKeyFunc: generates server key for each referenced node, can be nil.
func ToNodeConfigResponse(n *node.Node, referencedNodes []*node.Node, serverKeyFunc func(*node.Node) string) *NodeConfigResponse {
	if n == nil {
		return nil
	}

	config := &NodeConfigResponse{
		NodeSID:           n.SID(),
		ServerHost:        n.EffectiveServerAddress(),
		ServerPort:        int(n.AgentPort()),
		TransportProtocol: "tcp", // Default to TCP
		EnableVless:       false,
		EnableXTLS:        false,
		AllowInsecure:     false,
		SpeedLimit:        0, // 0 = unlimited, can be set from node metadata
		DeviceLimit:       0, // 0 = unlimited, can be set from node metadata
		RuleListPath:      "",
	}

	// Determine protocol type from node's protocol field
	switch {
	case n.Protocol().IsShadowsocks():
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

	case n.Protocol().IsTrojan():
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

	case n.Protocol().IsVLESS():
		config.Protocol = "vless"

		// Extract VLESS-specific configuration
		if n.VLESSConfig() != nil {
			vc := n.VLESSConfig()
			config.TransportProtocol = vc.TransportType()
			config.SNI = vc.SNI()
			config.AllowInsecure = vc.AllowInsecure()

			// VLESS specific fields
			config.VLESSFlow = vc.Flow()
			config.VLESSSecurity = vc.Security()
			config.VLESSFingerprint = vc.Fingerprint()

			// Reality specific fields
			config.VLESSRealityPublicKey = vc.PublicKey()
			config.VLESSRealityShortID = vc.ShortID()
			config.VLESSRealitySpiderX = vc.SpiderX()

			// Handle transport-specific fields
			switch vc.TransportType() {
			case "ws", "h2":
				config.Host = vc.Host()
				config.Path = vc.Path()
			case "grpc":
				config.ServiceName = vc.ServiceName()
			}
		}

	case n.Protocol().IsVMess():
		config.Protocol = "vmess"

		// Extract VMess-specific configuration
		if n.VMessConfig() != nil {
			vc := n.VMessConfig()
			config.TransportProtocol = vc.TransportType()
			config.SNI = vc.SNI()
			config.AllowInsecure = vc.AllowInsecure()

			// VMess specific fields
			config.VMessAlterID = vc.AlterID()
			config.VMessSecurity = vc.Security()
			config.VMessTLS = vc.TLS()

			// Handle transport-specific fields
			switch vc.TransportType() {
			case "ws", "http":
				config.Host = vc.Host()
				config.Path = vc.Path()
			case "grpc":
				config.ServiceName = vc.ServiceName()
			}
		}

	case n.Protocol().IsHysteria2():
		config.Protocol = "hysteria2"

		// Extract Hysteria2-specific configuration
		if n.Hysteria2Config() != nil {
			hc := n.Hysteria2Config()
			config.SNI = hc.SNI()
			config.AllowInsecure = hc.AllowInsecure()

			// Hysteria2 specific fields
			config.Hysteria2CongestionControl = hc.CongestionControl()
			config.Hysteria2Obfs = hc.Obfs()
			config.Hysteria2ObfsPassword = hc.ObfsPassword()
			config.Hysteria2UpMbps = hc.UpMbps()
			config.Hysteria2DownMbps = hc.DownMbps()
			config.Hysteria2Fingerprint = hc.Fingerprint()

			// Hysteria2 uses QUIC transport implicitly
			config.TransportProtocol = "quic"
		}

	case n.Protocol().IsTUIC():
		config.Protocol = "tuic"

		// Extract TUIC-specific configuration
		if n.TUICConfig() != nil {
			tc := n.TUICConfig()
			config.SNI = tc.SNI()
			config.AllowInsecure = tc.AllowInsecure()

			// TUIC specific fields
			config.TUICCongestionControl = tc.CongestionControl()
			config.TUICUDPRelayMode = tc.UDPRelayMode()
			config.TUICAlpn = tc.ALPN()
			config.TUICDisableSNI = tc.DisableSNI()

			// TUIC uses QUIC transport implicitly
			config.TransportProtocol = "quic"
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

// ToRouteConfigDTO converts domain RouteConfig to DTO
func ToRouteConfigDTO(rc *vo.RouteConfig) *RouteConfigDTO {
	if rc == nil {
		return nil
	}

	rules := make([]RouteRuleDTO, 0, len(rc.Rules()))
	for _, rule := range rc.Rules() {
		rules = append(rules, ToRouteRuleDTO(&rule))
	}

	return &RouteConfigDTO{
		Rules: rules,
		Final: rc.FinalAction().String(),
	}
}

// ToRouteRuleDTO converts domain RouteRule to DTO
func ToRouteRuleDTO(rule *vo.RouteRule) RouteRuleDTO {
	dto := RouteRuleDTO{
		Domain:        rule.Domain(),
		DomainSuffix:  rule.DomainSuffix(),
		DomainKeyword: rule.DomainKeyword(),
		DomainRegex:   rule.DomainRegex(),
		IPCIDR:        rule.IPCIDR(),
		SourceIPCIDR:  rule.SourceIPCIDR(),
		IPIsPrivate:   rule.IPIsPrivate(),
		GeoIP:         rule.GeoIP(),
		GeoSite:       rule.GeoSite(),
		Protocol:      rule.Protocol(),
		Network:       rule.Network(),
		RuleSet:       rule.RuleSet(),
		Outbound:      rule.Outbound().String(),
	}

	// Convert uint16 ports to int
	if len(rule.Port()) > 0 {
		dto.Port = make([]int, len(rule.Port()))
		for i, p := range rule.Port() {
			dto.Port[i] = int(p)
		}
	}
	if len(rule.SourcePort()) > 0 {
		dto.SourcePort = make([]int, len(rule.SourcePort()))
		for i, p := range rule.SourcePort() {
			dto.SourcePort[i] = int(p)
		}
	}

	return dto
}

// ToOutboundDTO converts a node entity to an OutboundDTO for sing-box outbound configuration.
// The serverKey is used for Shadowsocks server password (pre-generated), or UUID for VLESS/VMess/TUIC.
func ToOutboundDTO(n *node.Node, serverKey string) *OutboundDTO {
	if n == nil {
		return nil
	}

	dto := &OutboundDTO{
		Tag:    n.SID(),
		Server: n.EffectiveServerAddress(),
		Port:   int(n.EffectiveSubscriptionPort()),
	}

	switch {
	case n.Protocol().IsShadowsocks():
		dto.Type = "shadowsocks"
		dto.Method = n.EncryptionConfig().Method()
		dto.Password = serverKey

		// Handle plugin configuration
		if n.PluginConfig() != nil {
			dto.Plugin = n.PluginConfig().Plugin()
			// Convert plugin opts map to string format
			opts := n.PluginConfig().Opts()
			if len(opts) > 0 {
				var optsStr string
				for k, v := range opts {
					if optsStr != "" {
						optsStr += ";"
					}
					optsStr += k + "=" + v
				}
				dto.PluginOpts = optsStr
			}
		}

	case n.Protocol().IsTrojan():
		dto.Type = "trojan"
		dto.Password = serverKey

		// TLS configuration (Trojan always uses TLS)
		dto.TLS = &OutboundTLSDTO{
			Enabled: true,
		}

		if n.TrojanConfig() != nil {
			tc := n.TrojanConfig()
			dto.TLS.ServerName = tc.SNI()
			dto.TLS.Insecure = tc.AllowInsecure()

			// Transport configuration for ws/grpc
			switch tc.TransportProtocol() {
			case "ws":
				dto.Transport = &OutboundTransportDTO{
					Type: "ws",
					Path: tc.Path(),
				}
				if tc.Host() != "" {
					dto.Transport.Headers = map[string]string{"Host": tc.Host()}
				}
			case "grpc":
				dto.Transport = &OutboundTransportDTO{
					Type:        "grpc",
					ServiceName: tc.Host(),
				}
			}
		}

	case n.Protocol().IsVLESS():
		dto.Type = "vless"
		dto.UUID = serverKey // For VLESS, serverKey contains the user UUID

		if n.VLESSConfig() != nil {
			vc := n.VLESSConfig()

			// VLESS flow control
			dto.VLESSFlow = vc.Flow()

			// TLS/Reality configuration
			if vc.Security() == "reality" {
				dto.TLS = &OutboundTLSDTO{
					Enabled:    true,
					ServerName: vc.SNI(),
					Reality: &OutboundRealityDTO{
						Enabled:   true,
						PublicKey: vc.PublicKey(),
						ShortID:   vc.ShortID(),
					},
				}
			} else if vc.Security() == "tls" {
				dto.TLS = &OutboundTLSDTO{
					Enabled:    true,
					ServerName: vc.SNI(),
					Insecure:   vc.AllowInsecure(),
				}
			}

			// Transport configuration for ws/grpc/h2
			switch vc.TransportType() {
			case "ws":
				dto.Transport = &OutboundTransportDTO{
					Type: "ws",
					Path: vc.Path(),
				}
				if vc.Host() != "" {
					dto.Transport.Headers = map[string]string{"Host": vc.Host()}
				}
			case "grpc":
				dto.Transport = &OutboundTransportDTO{
					Type:        "grpc",
					ServiceName: vc.ServiceName(),
				}
			case "h2":
				dto.Transport = &OutboundTransportDTO{
					Type: "http",
					Path: vc.Path(),
				}
				if vc.Host() != "" {
					dto.Transport.Headers = map[string]string{"Host": vc.Host()}
				}
			}
		}

	case n.Protocol().IsVMess():
		dto.Type = "vmess"
		dto.UUID = serverKey // For VMess, serverKey contains the user UUID

		if n.VMessConfig() != nil {
			vc := n.VMessConfig()

			// VMess specific fields
			dto.VMessAlterID = vc.AlterID()
			dto.VMessSecurity = vc.Security()

			// TLS configuration
			if vc.TLS() {
				dto.TLS = &OutboundTLSDTO{
					Enabled:    true,
					ServerName: vc.SNI(),
					Insecure:   vc.AllowInsecure(),
				}
			}

			// Transport configuration for ws/grpc/http
			switch vc.TransportType() {
			case "ws":
				dto.Transport = &OutboundTransportDTO{
					Type: "ws",
					Path: vc.Path(),
				}
				if vc.Host() != "" {
					dto.Transport.Headers = map[string]string{"Host": vc.Host()}
				}
			case "grpc":
				dto.Transport = &OutboundTransportDTO{
					Type:        "grpc",
					ServiceName: vc.ServiceName(),
				}
			case "http":
				dto.Transport = &OutboundTransportDTO{
					Type: "http",
					Path: vc.Path(),
				}
				if vc.Host() != "" {
					dto.Transport.Headers = map[string]string{"Host": vc.Host()}
				}
			}
		}

	case n.Protocol().IsHysteria2():
		dto.Type = "hysteria2"
		dto.Password = serverKey // For Hysteria2, serverKey is the password

		if n.Hysteria2Config() != nil {
			hc := n.Hysteria2Config()

			// Hysteria2 specific fields
			dto.Hysteria2Obfs = hc.Obfs()
			dto.Hysteria2ObfsPassword = hc.ObfsPassword()
			dto.Hysteria2UpMbps = hc.UpMbps()
			dto.Hysteria2DownMbps = hc.DownMbps()

			// TLS configuration
			dto.TLS = &OutboundTLSDTO{
				Enabled:    true,
				ServerName: hc.SNI(),
				Insecure:   hc.AllowInsecure(),
			}
		}

	case n.Protocol().IsTUIC():
		dto.Type = "tuic"
		dto.UUID = serverKey // For TUIC, serverKey contains the UUID

		if n.TUICConfig() != nil {
			tc := n.TUICConfig()

			// TUIC password (separate from UUID)
			dto.Password = tc.Password()

			// TUIC specific fields
			dto.TUICCongestionControl = tc.CongestionControl()
			dto.TUICUDPRelayMode = tc.UDPRelayMode()

			// TLS configuration
			dto.TLS = &OutboundTLSDTO{
				Enabled:    true,
				ServerName: tc.SNI(),
				Insecure:   tc.AllowInsecure(),
				DisableSNI: tc.DisableSNI(),
			}
			if tc.ALPN() != "" {
				dto.TLS.ALPN = []string{tc.ALPN()}
			}
		}
	}

	return dto
}

// ToOutboundDTOs converts a list of nodes to OutboundDTOs.
// The serverKeyFunc generates the server key for each node based on its encryption method.
func ToOutboundDTOs(nodes []*node.Node, serverKeyFunc func(n *node.Node) string) []OutboundDTO {
	if len(nodes) == 0 {
		return nil
	}

	dtos := make([]OutboundDTO, 0, len(nodes))
	for _, n := range nodes {
		if n == nil {
			continue
		}
		serverKey := ""
		if serverKeyFunc != nil {
			serverKey = serverKeyFunc(n)
		}
		if dto := ToOutboundDTO(n, serverKey); dto != nil {
			dtos = append(dtos, *dto)
		}
	}
	return dtos
}

// ToNodeSubscriptionsResponse converts subscription list to agent subscriptions response
// The hmacSecret is used to generate HMAC-signed passwords from subscription UUIDs
// The encryptionMethod parameter determines the password encoding format (hex for traditional SS, base64 for SS2022)
func ToNodeSubscriptionsResponse(subscriptions []*subscription.Subscription, hmacSecret string, encryptionMethod string) *NodeSubscriptionsResponse {
	if subscriptions == nil {
		return &NodeSubscriptionsResponse{
			Subscriptions: []NodeSubscriptionInfo{},
		}
	}

	subscriptionInfos := make([]NodeSubscriptionInfo, 0, len(subscriptions))
	for _, sub := range subscriptions {
		if sub == nil {
			continue
		}

		// Skip inactive subscriptions
		if !sub.IsActive() {
			continue
		}

		subscriptionInfo := NodeSubscriptionInfo{
			SubscriptionSID: sub.SID(), // Using subscription SID for traffic tracking
			Password:        generatePasswordForEncryptionMethod(sub, hmacSecret, encryptionMethod),
			Name:            generateSubscriptionName(sub),
			SpeedLimit:      0, // 0 = unlimited, can be set from subscription plan limits
			DeviceLimit:     0, // 0 = unlimited, can be set from subscription plan limits
			ExpireTime:      sub.EndDate().Unix(),
		}

		subscriptionInfos = append(subscriptionInfos, subscriptionInfo)
	}

	return &NodeSubscriptionsResponse{
		Subscriptions: subscriptionInfos,
	}
}

// NewSuccessResponse creates a success response for agent API
func NewSuccessResponse(data interface{}) *AgentResponse {
	return &AgentResponse{
		Data: data,
		Ret:  1,
		Msg:  "success",
	}
}

// NewErrorResponse creates an error response for agent API
func NewErrorResponse(msg string) *AgentResponse {
	return &AgentResponse{
		Data: nil,
		Ret:  0,
		Msg:  msg,
	}
}

// Helper functions

// generateSubscriptionName generates name identifier for subscription (sing-box compatible)
// Format: user{userId}-sub{subscriptionId}
func generateSubscriptionName(sub *subscription.Subscription) string {
	if sub == nil {
		return ""
	}
	return fmt.Sprintf("user%d-sub%d", sub.UserID(), sub.ID())
}

// FromRouteConfigDTO converts RouteConfigDTO to domain RouteConfig
func FromRouteConfigDTO(dto *RouteConfigDTO) (*vo.RouteConfig, error) {
	if dto == nil {
		return nil, nil
	}

	// Default final action to "direct" if not specified
	finalStr := dto.Final
	if finalStr == "" {
		finalStr = "direct"
	}

	// Parse and validate final action
	finalAction, err := vo.ParseOutboundType(finalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid final action: %w", err)
	}

	// Create route config
	config, err := vo.NewRouteConfig(finalAction)
	if err != nil {
		return nil, fmt.Errorf("failed to create route config: %w", err)
	}

	// Convert and add rules
	for i, ruleDTO := range dto.Rules {
		rule, err := FromRouteRuleDTO(&ruleDTO)
		if err != nil {
			return nil, fmt.Errorf("invalid rule at index %d: %w", i, err)
		}
		if err := config.AddRule(*rule); err != nil {
			return nil, fmt.Errorf("failed to add rule at index %d: %w", i, err)
		}
	}

	return config, nil
}

// FromRouteRuleDTO converts RouteRuleDTO to domain RouteRule
func FromRouteRuleDTO(dto *RouteRuleDTO) (*vo.RouteRule, error) {
	if dto == nil {
		return nil, nil
	}

	// Parse and validate outbound type
	outbound, err := vo.ParseOutboundType(dto.Outbound)
	if err != nil {
		return nil, fmt.Errorf("invalid outbound type: %w", err)
	}

	// Create rule with outbound action
	rule, err := vo.NewRouteRule(outbound)
	if err != nil {
		return nil, err
	}

	// Apply all conditions using builder pattern
	if len(dto.Domain) > 0 {
		rule.WithDomain(dto.Domain...)
	}
	if len(dto.DomainSuffix) > 0 {
		rule.WithDomainSuffix(dto.DomainSuffix...)
	}
	if len(dto.DomainKeyword) > 0 {
		rule.WithDomainKeyword(dto.DomainKeyword...)
	}
	if len(dto.DomainRegex) > 0 {
		rule.WithDomainRegex(dto.DomainRegex...)
	}
	if len(dto.IPCIDR) > 0 {
		rule.WithIPCIDR(dto.IPCIDR...)
	}
	if len(dto.SourceIPCIDR) > 0 {
		rule.WithSourceIPCIDR(dto.SourceIPCIDR...)
	}
	if dto.IPIsPrivate {
		rule.WithIPIsPrivate(true)
	}
	if len(dto.GeoIP) > 0 {
		rule.WithGeoIP(dto.GeoIP...)
	}
	if len(dto.GeoSite) > 0 {
		rule.WithGeoSite(dto.GeoSite...)
	}
	// Convert int ports to uint16
	if len(dto.Port) > 0 {
		ports := make([]uint16, len(dto.Port))
		for i, p := range dto.Port {
			if p < 1 || p > 65535 {
				return nil, fmt.Errorf("invalid port number: %d (must be 1-65535)", p)
			}
			ports[i] = uint16(p)
		}
		rule.WithPort(ports...)
	}
	if len(dto.SourcePort) > 0 {
		ports := make([]uint16, len(dto.SourcePort))
		for i, p := range dto.SourcePort {
			if p < 1 || p > 65535 {
				return nil, fmt.Errorf("invalid source port number: %d (must be 1-65535)", p)
			}
			ports[i] = uint16(p)
		}
		rule.WithSourcePort(ports...)
	}
	if len(dto.Protocol) > 0 {
		rule.WithProtocol(dto.Protocol...)
	}
	if len(dto.Network) > 0 {
		rule.WithNetwork(dto.Network...)
	}
	if len(dto.RuleSet) > 0 {
		rule.WithRuleSet(dto.RuleSet...)
	}

	return rule, nil
}

// generatePasswordForEncryptionMethod generates password based on encryption method type
// SS2022 methods use base64-encoded fixed-length keys
// Traditional SS methods use hex-encoded keys (backward compatible)
func generatePasswordForEncryptionMethod(sub *subscription.Subscription, secret string, method string) string {
	if sub == nil || sub.UUID() == "" {
		return ""
	}

	// Use HMAC-SHA256 to derive key material from subscription UUID
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sub.UUID()))
	keyMaterial := mac.Sum(nil) // 32 bytes

	// SS2022 requires base64-encoded fixed-length key
	if vo.IsSS2022Method(method) {
		keySize := vo.GetSS2022KeySize(method)
		if keySize > 0 && keySize <= len(keyMaterial) {
			return base64.StdEncoding.EncodeToString(keyMaterial[:keySize])
		}
	}

	// Traditional SS: hex-encoded (backward compatible)
	return hex.EncodeToString(keyMaterial)
}
