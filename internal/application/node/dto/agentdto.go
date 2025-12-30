package dto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

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
	NodeSID           string          `json:"node_id" binding:"required"`                              // Node SID (Stripe-style: node_xxx)
	Protocol          string          `json:"protocol" binding:"required,oneof=shadowsocks trojan"`    // Protocol type
	ServerHost        string          `json:"server_host" binding:"required"`                          // Server hostname or IP address
	ServerPort        int             `json:"server_port" binding:"required,min=1,max=65535"`          // Server port number
	EncryptionMethod  string          `json:"encryption_method,omitempty"`                             // Encryption method for Shadowsocks
	ServerKey         string          `json:"server_key,omitempty"`                                    // Server password for SS
	TransportProtocol string          `json:"transport_protocol" binding:"required,oneof=tcp ws grpc"` // Transport protocol (tcp, ws, grpc)
	Host              string          `json:"host,omitempty"`                                          // WebSocket host header
	Path              string          `json:"path,omitempty"`                                          // WebSocket path
	ServiceName       string          `json:"service_name,omitempty"`                                  // gRPC service name
	SNI               string          `json:"sni,omitempty"`                                           // TLS Server Name Indication
	AllowInsecure     bool            `json:"allow_insecure"`                                          // Allow insecure TLS connection
	EnableVless       bool            `json:"enable_vless"`                                            // Enable VLESS protocol
	EnableXTLS        bool            `json:"enable_xtls"`                                             // Enable XTLS
	SpeedLimit        uint64          `json:"speed_limit"`                                             // Speed limit in Mbps, 0 = unlimited
	DeviceLimit       int             `json:"device_limit"`                                            // Device connection limit, 0 = unlimited
	RuleListPath      string          `json:"rule_list_path,omitempty"`                                // Path to routing rule list file (deprecated, use Route)
	Route             *RouteConfigDTO `json:"route,omitempty"`                                         // Routing configuration for traffic splitting
	Outbounds         []OutboundDTO   `json:"outbounds,omitempty"`                                     // Outbound configs for nodes referenced in route rules
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
	Type   string `json:"type"`   // Protocol type: shadowsocks, trojan, direct, block
	Tag    string `json:"tag"`    // Unique identifier for the outbound (node SID)
	Server string `json:"server"` // Server hostname or IP address
	Port   int    `json:"server_port"`

	// Shadowsocks specific fields
	Method     string `json:"method,omitempty"`      // Encryption method for SS
	Password   string `json:"password,omitempty"`    // Password for SS/Trojan
	Plugin     string `json:"plugin,omitempty"`      // SIP003 plugin name
	PluginOpts string `json:"plugin_opts,omitempty"` // Plugin options string

	// TLS fields (for Trojan)
	TLS *OutboundTLSDTO `json:"tls,omitempty"` // TLS configuration

	// Transport fields (for Trojan ws/grpc)
	Transport *OutboundTransportDTO `json:"transport,omitempty"` // Transport configuration
}

// OutboundTLSDTO represents TLS configuration for outbound.
type OutboundTLSDTO struct {
	Enabled    bool     `json:"enabled"`               // Enable TLS
	ServerName string   `json:"server_name,omitempty"` // SNI
	Insecure   bool     `json:"insecure,omitempty"`    // Allow insecure TLS
	DisableSNI bool     `json:"disable_sni,omitempty"` // Disable SNI
	ALPN       []string `json:"alpn,omitempty"`        // ALPN protocols
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

// ReportNodeStatusRequest represents node status report request with procfs metrics
type ReportNodeStatusRequest struct {
	// System resources
	CPUPercent    float64 `json:"cpu_percent" binding:"min=0,max=100"`    // CPU usage percentage (0-100)
	MemoryPercent float64 `json:"memory_percent" binding:"min=0,max=100"` // Memory usage percentage (0-100)
	MemoryUsed    uint64  `json:"memory_used"`                            // Memory used in bytes
	MemoryTotal   uint64  `json:"memory_total"`                           // Total memory in bytes
	MemoryAvail   uint64  `json:"memory_avail"`                           // Available memory in bytes (from procfs Meminfo)
	DiskPercent   float64 `json:"disk_percent" binding:"min=0,max=100"`   // Disk usage percentage (0-100)
	DiskUsed      uint64  `json:"disk_used"`                              // Disk used in bytes
	DiskTotal     uint64  `json:"disk_total"`                             // Total disk in bytes
	UptimeSeconds int64   `json:"uptime_seconds" binding:"min=0"`         // System uptime in seconds

	// System load (from procfs LoadAvg)
	LoadAvg1  float64 `json:"load_avg_1"`  // 1-minute load average
	LoadAvg5  float64 `json:"load_avg_5"`  // 5-minute load average
	LoadAvg15 float64 `json:"load_avg_15"` // 15-minute load average

	// Network statistics (from procfs NetDev)
	NetworkRxBytes uint64 `json:"network_rx_bytes"` // Total received bytes across all interfaces
	NetworkTxBytes uint64 `json:"network_tx_bytes"` // Total transmitted bytes across all interfaces

	// Network bandwidth (calculated by agent)
	NetworkRxRate uint64 `json:"network_rx_rate"` // Current receive rate in bytes per second
	NetworkTxRate uint64 `json:"network_tx_rate"` // Current transmit rate in bytes per second

	// Connection statistics
	TCPConnections int `json:"tcp_connections" binding:"min=0"` // Number of TCP connections
	UDPConnections int `json:"udp_connections" binding:"min=0"` // Number of UDP connections

	// Network info
	PublicIPv4 string `json:"public_ipv4,omitempty"` // Public IPv4 address reported by agent
	PublicIPv6 string `json:"public_ipv6,omitempty"` // Public IPv6 address reported by agent

	// Agent info
	AgentVersion string `json:"agent_version,omitempty"` // Agent software version (e.g., "1.2.3")
	Platform     string `json:"platform,omitempty"`      // OS platform (linux, darwin, windows)
	Arch         string `json:"arch,omitempty"`          // CPU architecture (amd64, arm64, arm, 386)
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
// Supports both Shadowsocks and Trojan protocols with sing-box compatible configuration.
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
// The serverKey is used for Shadowsocks server password (pre-generated).
func ToOutboundDTO(n *node.Node, serverKey string) *OutboundDTO {
	if n == nil {
		return nil
	}

	dto := &OutboundDTO{
		Tag:    n.SID(),
		Server: n.EffectiveServerAddress(),
		Port:   int(n.EffectiveSubscriptionPort()),
	}

	if n.Protocol().IsShadowsocks() {
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
	} else if n.Protocol().IsTrojan() {
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
