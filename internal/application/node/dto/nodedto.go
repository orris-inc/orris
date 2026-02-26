package dto

import (
	"time"

	commondto "github.com/orris-inc/orris/internal/application/common/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/mapper"
)

type NodeDTO struct {
	ID               string            `json:"id" example:"node_xK9mP2vL3nQ" description:"Unique identifier for the node (Stripe-style prefixed ID)"`
	Name             string            `json:"name" example:"US-Node-01" description:"Display name of the node"`
	ServerAddress    string            `json:"server_address" example:"proxy.example.com" description:"Server hostname or IP address"`
	AgentPort        uint16            `json:"agent_port" example:"8388" description:"Port for agent connections"`
	SubscriptionPort *uint16           `json:"subscription_port,omitempty" example:"8389" description:"Port for client subscriptions (if null, uses agent_port)"`
	Protocol         string            `json:"protocol" example:"shadowsocks" enums:"shadowsocks,trojan,vless,vmess,hysteria2,tuic,anytls" description:"Proxy protocol type"`
	EncryptionMethod string            `json:"encryption_method" example:"aes-256-gcm" enums:"aes-256-gcm,aes-128-gcm,chacha20-ietf-poly1305" description:"Encryption method for the proxy connection"`
	Plugin           string            `json:"plugin,omitempty" example:"obfs-local" description:"Optional plugin name"`
	PluginOpts       map[string]string `json:"plugin_opts,omitempty" example:"obfs:http,obfs-host:example.com" description:"Plugin configuration options"`
	Status           string            `json:"status" example:"active" enums:"active,inactive,maintenance" description:"Current operational status of the node"`
	Region           string            `json:"region,omitempty" example:"us-west" description:"Geographic region or location identifier"`
	Tags             []string          `json:"tags,omitempty" example:"premium,fast" description:"Custom tags for categorization"`
	SortOrder        int               `json:"sort_order" example:"100" description:"Display order for sorting nodes"`
	MuteNotification bool              `json:"mute_notification" example:"false" description:"Mute online/offline notifications for this node"`
	// Trojan specific fields
	TransportProtocol string  `json:"transport_protocol,omitempty" example:"tcp" enums:"tcp,ws,grpc" description:"Transport protocol for Trojan (tcp, ws, grpc)"`
	Host              string  `json:"host,omitempty" example:"cdn.example.com" description:"WebSocket host header or gRPC service name"`
	Path              string  `json:"path,omitempty" example:"/trojan" description:"WebSocket path"`
	SNI               string  `json:"sni,omitempty" example:"example.com" description:"TLS Server Name Indication"`
	AllowInsecure     bool    `json:"allow_insecure,omitempty" example:"true" description:"Allow insecure TLS connection"`
	MaintenanceReason *string `json:"maintenance_reason,omitempty" example:"Scheduled maintenance" description:"Reason for maintenance status (only when status is maintenance)"`

	// VLESS specific fields
	VLESSTransportType    string `json:"vless_transport_type,omitempty" example:"tcp" enums:"tcp,ws,grpc,h2" description:"VLESS transport type"`
	VLESSFlow             string `json:"vless_flow,omitempty" example:"xtls-rprx-vision" description:"VLESS flow control"`
	VLESSSecurity         string `json:"vless_security,omitempty" example:"tls" enums:"none,tls,reality" description:"VLESS security type"`
	VLESSSni              string `json:"vless_sni,omitempty" example:"example.com" description:"VLESS TLS SNI"`
	VLESSFingerprint      string `json:"vless_fingerprint,omitempty" example:"chrome" description:"VLESS TLS fingerprint"`
	VLESSAllowInsecure    bool   `json:"vless_allow_insecure,omitempty" description:"VLESS allow insecure TLS"`
	VLESSHost             string `json:"vless_host,omitempty" description:"VLESS WS/H2 host header"`
	VLESSPath             string `json:"vless_path,omitempty" description:"VLESS WS/H2 path"`
	VLESSServiceName      string `json:"vless_service_name,omitempty" description:"VLESS gRPC service name"`
	VLESSRealityPublicKey string `json:"vless_reality_public_key,omitempty" description:"VLESS Reality public key"`
	VLESSRealityShortID   string `json:"vless_reality_short_id,omitempty" description:"VLESS Reality short ID"`
	VLESSRealitySpiderX   string `json:"vless_reality_spider_x,omitempty" description:"VLESS Reality spider X"`

	// VMess specific fields
	VMessAlterID       int    `json:"vmess_alter_id,omitempty" example:"0" description:"VMess alter ID"`
	VMessSecurity      string `json:"vmess_security,omitempty" example:"auto" enums:"auto,aes-128-gcm,chacha20-poly1305,none,zero" description:"VMess security"`
	VMessTransportType string `json:"vmess_transport_type,omitempty" example:"tcp" enums:"tcp,ws,grpc,http,quic" description:"VMess transport type"`
	VMessHost          string `json:"vmess_host,omitempty" description:"VMess WS/HTTP host header"`
	VMessPath          string `json:"vmess_path,omitempty" description:"VMess WS/HTTP path"`
	VMessServiceName   string `json:"vmess_service_name,omitempty" description:"VMess gRPC service name"`
	VMessTLS           bool   `json:"vmess_tls,omitempty" description:"VMess TLS enabled"`
	VMessSni           string `json:"vmess_sni,omitempty" description:"VMess TLS SNI"`
	VMessAllowInsecure bool   `json:"vmess_allow_insecure,omitempty" description:"VMess allow insecure TLS"`

	// Hysteria2 specific fields
	Hysteria2CongestionControl string `json:"hysteria2_congestion_control,omitempty" example:"bbr" enums:"cubic,bbr,new_reno" description:"Hysteria2 congestion control"`
	Hysteria2Obfs              string `json:"hysteria2_obfs,omitempty" example:"salamander" enums:"salamander" description:"Hysteria2 obfuscation type"`
	Hysteria2ObfsPassword      string `json:"hysteria2_obfs_password,omitempty" description:"Hysteria2 obfuscation password"`
	Hysteria2UpMbps            *int   `json:"hysteria2_up_mbps,omitempty" description:"Hysteria2 upstream bandwidth limit"`
	Hysteria2DownMbps          *int   `json:"hysteria2_down_mbps,omitempty" description:"Hysteria2 downstream bandwidth limit"`
	Hysteria2Sni               string `json:"hysteria2_sni,omitempty" description:"Hysteria2 TLS SNI"`
	Hysteria2AllowInsecure     bool   `json:"hysteria2_allow_insecure,omitempty" description:"Hysteria2 allow insecure TLS"`
	Hysteria2Fingerprint       string `json:"hysteria2_fingerprint,omitempty" description:"Hysteria2 TLS fingerprint"`

	// TUIC specific fields
	TUICCongestionControl string     `json:"tuic_congestion_control,omitempty" example:"bbr" enums:"cubic,bbr,new_reno" description:"TUIC congestion control"`
	TUICUDPRelayMode      string     `json:"tuic_udp_relay_mode,omitempty" example:"native" enums:"native,quic" description:"TUIC UDP relay mode"`
	TUICAlpn              string     `json:"tuic_alpn,omitempty" description:"TUIC ALPN protocols"`
	TUICSni               string     `json:"tuic_sni,omitempty" description:"TUIC TLS SNI"`
	TUICAllowInsecure     bool   `json:"tuic_allow_insecure,omitempty" description:"TUIC allow insecure TLS"`
	TUICDisableSNI        bool   `json:"tuic_disable_sni,omitempty" description:"TUIC disable SNI"`

	// AnyTLS specific fields
	AnyTLSSni                      string `json:"anytls_sni,omitempty" description:"AnyTLS TLS SNI"`
	AnyTLSAllowInsecure            bool   `json:"anytls_allow_insecure,omitempty" description:"AnyTLS allow insecure TLS"`
	AnyTLSFingerprint              string `json:"anytls_fingerprint,omitempty" description:"AnyTLS TLS fingerprint"`
	AnyTLSIdleSessionCheckInterval string `json:"anytls_idle_session_check_interval,omitempty" description:"AnyTLS idle session check interval"`
	AnyTLSIdleSessionTimeout       string `json:"anytls_idle_session_timeout,omitempty" description:"AnyTLS idle session timeout"`
	AnyTLSMinIdleSession           int    `json:"anytls_min_idle_session,omitempty" description:"AnyTLS minimum idle sessions"`

	IsOnline                bool `json:"is_online" example:"true" description:"Indicates if the node agent is online (reported within 5 minutes)"`
	OnlineSubscriptionCount int  `json:"online_subscription_count" description:"Number of online subscriptions on this node"`
	LastSeenAt            *time.Time `json:"last_seen_at,omitempty" example:"2024-01-15T14:20:00Z" description:"Last time the node agent reported status"`
	ExpiresAt             *string    `json:"expires_at,omitempty" example:"2025-12-31T23:59:59Z" description:"Expiration time in ISO8601 format (null = never expires)"`
	CostLabel             string     `json:"cost_label,omitempty" example:"35$/m" description:"Cost label for display (e.g., '35$/m', '35¥/y')"`
	IsExpired             bool       `json:"is_expired" example:"false" description:"True if node has expired"`
	AgentVersion          string     `json:"agent_version,omitempty" example:"1.2.0" description:"Agent software version, extracted from system_status for easy display"`
	Platform              string     `json:"platform,omitempty" example:"linux" description:"OS platform (linux, darwin, windows)"`
	Arch                  string     `json:"arch,omitempty" example:"amd64" description:"CPU architecture (amd64, arm64, arm, 386)"`
	HasUpdate             bool       `json:"has_update" example:"true" description:"True if a newer agent version is available"`
	GroupSIDs             []string   `json:"group_sids,omitempty" example:"[\"rg_xK9mP2vL3nQ\"]" description:"Resource group SIDs this node belongs to"`
	Version               int        `json:"version" example:"1" description:"Version number for optimistic locking"`
	CreatedAt             time.Time  `json:"created_at" example:"2024-01-15T10:30:00Z" description:"Timestamp when the node was created"`
	UpdatedAt             time.Time  `json:"updated_at" example:"2024-01-15T14:20:00Z" description:"Timestamp when the node was last updated"`
	// System status fields (from Redis)
	SystemStatus *NodeSystemStatusDTO `json:"system_status,omitempty" description:"Real-time system metrics from monitoring"`
	// Owner information (for user-created nodes)
	Owner *NodeOwnerDTO `json:"owner,omitempty" description:"Owner information for user-created nodes"`
	// Route configuration for traffic splitting (sing-box compatible)
	Route *RouteConfigDTO `json:"route,omitempty" description:"Routing configuration for traffic splitting"`
	// DNS configuration for DNS-based unlocking (sing-box compatible)
	DNS *DnsConfigDTO `json:"dns,omitempty" description:"DNS configuration for DNS-based unlocking"`
}

// NodeOwnerDTO represents the owner information for a user-created node
type NodeOwnerDTO struct {
	ID    string `json:"id" example:"user_xK9mP2vL3nQ" description:"User's Stripe-style ID"`
	Email string `json:"email" example:"user@example.com" description:"User's email address"`
	Name  string `json:"name" example:"John Doe" description:"User's display name"`
}

// NodeSystemStatusDTO represents real-time system status metrics retrieved from Redis cache.
// Embeds common SystemStatus for shared fields across all agent types.
type NodeSystemStatusDTO struct {
	commondto.SystemStatus
}

type CreateNodeDTO struct {
	Name             string            `json:"name" binding:"required,min=2,max=100" example:"US-Node-01" description:"Display name of the node (2-100 characters)"`
	ServerAddress    string            `json:"server_address,omitempty" example:"proxy.example.com" description:"Server hostname or IP address (optional, can be auto-detected from agent)"`
	AgentPort        uint16            `json:"agent_port" binding:"required,min=1,max=65535" example:"8388" description:"Port for agent connections (1-65535)"`
	SubscriptionPort *uint16           `json:"subscription_port,omitempty" binding:"omitempty,min=1,max=65535" example:"8389" description:"Port for client subscriptions (if null, uses agent_port)"`
	EncryptionMethod string            `json:"encryption_method" binding:"required" example:"aes-256-gcm" enums:"aes-256-gcm,aes-128-gcm,chacha20-ietf-poly1305" description:"Encryption method for the proxy connection"`
	Password         string            `json:"password" binding:"required" example:"mySecurePassword123" description:"Authentication password"`
	Plugin           string            `json:"plugin,omitempty" example:"obfs-local" description:"Optional plugin name"`
	PluginOpts       map[string]string `json:"plugin_opts,omitempty" example:"obfs:http,obfs-host:example.com" description:"Plugin configuration options"`
	Region           string            `json:"region,omitempty" example:"us-west" description:"Geographic region or location identifier"`
	Tags             []string          `json:"tags,omitempty" example:"premium,fast" description:"Custom tags for categorization"`
	SortOrder        int               `json:"sort_order" example:"100" description:"Display order for sorting nodes"`
	Route            *RouteConfigDTO   `json:"route,omitempty" description:"Routing configuration for traffic splitting (sing-box compatible)"`
	DNS              *DnsConfigDTO     `json:"dns,omitempty" description:"DNS configuration for DNS-based unlocking (sing-box compatible)"`
}

type NodeListDTO struct {
	Nodes      []*NodeDTO         `json:"nodes" description:"List of node objects"`
	Pagination PaginationResponse `json:"pagination" description:"Pagination metadata"`
}

type PaginationResponse struct {
	Page       int `json:"page" example:"1" description:"Current page number"`
	PageSize   int `json:"page_size" example:"20" description:"Number of items per page"`
	Total      int `json:"total" example:"100" description:"Total number of items"`
	TotalPages int `json:"total_pages" example:"5" description:"Total number of pages"`
}

type ListNodesRequest struct {
	Page     int      `json:"page" form:"page" example:"1" description:"Page number for pagination"`
	PageSize int      `json:"page_size" form:"page_size" example:"20" description:"Number of items per page"`
	Status   string   `json:"status,omitempty" form:"status" example:"active" enums:"active,inactive,maintenance" description:"Filter by node status"`
	Region   string   `json:"region,omitempty" form:"region" example:"us-west" description:"Filter by geographic region"`
	Tags     []string `json:"tags,omitempty" form:"tags" example:"premium,fast" description:"Filter by tags"`
	OrderBy  string   `json:"order_by,omitempty" form:"order_by" example:"created_at" description:"Field to order by"`
	Order    string   `json:"order,omitempty" form:"order" binding:"omitempty,oneof=asc desc" example:"desc" enums:"asc,desc" description:"Sort order (ascending or descending)"`
}

func ToNodeDTO(n *node.Node) *NodeDTO {
	if n == nil {
		return nil
	}

	// Format expires_at as ISO8601 string
	var expiresAtStr *string
	if n.ExpiresAt() != nil {
		s := n.ExpiresAt().Format("2006-01-02T15:04:05Z07:00")
		expiresAtStr = &s
	}

	dto := &NodeDTO{
		ID:                n.SID(),
		Name:              n.Name(),
		ServerAddress:     n.ServerAddress().Value(),
		AgentPort:         n.AgentPort(),
		SubscriptionPort:  n.SubscriptionPort(),
		Protocol:          n.Protocol().String(),
		EncryptionMethod:  n.EncryptionConfig().Method(),
		Status:            n.Status().String(),
		SortOrder:         n.SortOrder(),
		MuteNotification:  n.MuteNotification(),
		MaintenanceReason: n.MaintenanceReason(),
		IsOnline:          n.IsOnline(),
		LastSeenAt:        n.LastSeenAt(),
		ExpiresAt:         expiresAtStr,
		IsExpired:         n.IsExpired(),
		Version:           n.Version(),
		CreatedAt:         n.CreatedAt(),
		UpdatedAt:         n.UpdatedAt(),
	}

	// Map cost label
	if n.CostLabel() != nil {
		dto.CostLabel = *n.CostLabel()
	}

	// Map agent info fields
	if n.AgentVersion() != nil {
		dto.AgentVersion = *n.AgentVersion()
	}
	if n.AgentPlatform() != nil {
		dto.Platform = *n.AgentPlatform()
	}
	if n.AgentArch() != nil {
		dto.Arch = *n.AgentArch()
	}

	if n.PluginConfig() != nil {
		dto.Plugin = n.PluginConfig().Plugin()
		dto.PluginOpts = n.PluginConfig().Opts()
	}

	// Map Trojan specific fields
	if n.TrojanConfig() != nil {
		dto.TransportProtocol = n.TrojanConfig().TransportProtocol()
		dto.Host = n.TrojanConfig().Host()
		dto.Path = n.TrojanConfig().Path()
		dto.SNI = n.TrojanConfig().SNI()
		dto.AllowInsecure = n.TrojanConfig().AllowInsecure()
	}

	// Map VLESS specific fields
	if n.VLESSConfig() != nil {
		dto.VLESSTransportType = n.VLESSConfig().TransportType()
		dto.VLESSFlow = n.VLESSConfig().Flow()
		dto.VLESSSecurity = n.VLESSConfig().Security()
		dto.VLESSSni = n.VLESSConfig().SNI()
		dto.VLESSFingerprint = n.VLESSConfig().Fingerprint()
		dto.VLESSAllowInsecure = n.VLESSConfig().AllowInsecure()
		dto.VLESSHost = n.VLESSConfig().Host()
		dto.VLESSPath = n.VLESSConfig().Path()
		dto.VLESSServiceName = n.VLESSConfig().ServiceName()
		dto.VLESSRealityPublicKey = n.VLESSConfig().PublicKey()
		dto.VLESSRealityShortID = n.VLESSConfig().ShortID()
		dto.VLESSRealitySpiderX = n.VLESSConfig().SpiderX()
	}

	// Map VMess specific fields
	if n.VMessConfig() != nil {
		dto.VMessAlterID = n.VMessConfig().AlterID()
		dto.VMessSecurity = n.VMessConfig().Security()
		dto.VMessTransportType = n.VMessConfig().TransportType()
		dto.VMessHost = n.VMessConfig().Host()
		dto.VMessPath = n.VMessConfig().Path()
		dto.VMessServiceName = n.VMessConfig().ServiceName()
		dto.VMessTLS = n.VMessConfig().TLS()
		dto.VMessSni = n.VMessConfig().SNI()
		dto.VMessAllowInsecure = n.VMessConfig().AllowInsecure()
	}

	// Map Hysteria2 specific fields
	if n.Hysteria2Config() != nil {
		dto.Hysteria2CongestionControl = n.Hysteria2Config().CongestionControl()
		dto.Hysteria2Obfs = n.Hysteria2Config().Obfs()
		dto.Hysteria2ObfsPassword = n.Hysteria2Config().ObfsPassword()
		dto.Hysteria2UpMbps = n.Hysteria2Config().UpMbps()
		dto.Hysteria2DownMbps = n.Hysteria2Config().DownMbps()
		dto.Hysteria2Sni = n.Hysteria2Config().SNI()
		dto.Hysteria2AllowInsecure = n.Hysteria2Config().AllowInsecure()
		dto.Hysteria2Fingerprint = n.Hysteria2Config().Fingerprint()
	}

	// Map TUIC specific fields
	if n.TUICConfig() != nil {
		dto.TUICCongestionControl = n.TUICConfig().CongestionControl()
		dto.TUICUDPRelayMode = n.TUICConfig().UDPRelayMode()
		dto.TUICAlpn = n.TUICConfig().ALPN()
		dto.TUICSni = n.TUICConfig().SNI()
		dto.TUICAllowInsecure = n.TUICConfig().AllowInsecure()
		dto.TUICDisableSNI = n.TUICConfig().DisableSNI()
	}

	// Map AnyTLS specific fields
	if n.AnyTLSConfig() != nil {
		dto.AnyTLSSni = n.AnyTLSConfig().SNI()
		dto.AnyTLSAllowInsecure = n.AnyTLSConfig().AllowInsecure()
		dto.AnyTLSFingerprint = n.AnyTLSConfig().Fingerprint()
		dto.AnyTLSIdleSessionCheckInterval = n.AnyTLSConfig().IdleSessionCheckInterval()
		dto.AnyTLSIdleSessionTimeout = n.AnyTLSConfig().IdleSessionTimeout()
		dto.AnyTLSMinIdleSession = n.AnyTLSConfig().MinIdleSession()
	}

	metadata := n.Metadata()
	if metadata.Region() != "" {
		dto.Region = metadata.Region()
	}
	if len(metadata.Tags()) > 0 {
		dto.Tags = metadata.Tags()
	}

	// Map route configuration if present
	if n.RouteConfig() != nil {
		dto.Route = ToRouteConfigDTO(n.RouteConfig())
	}

	// Map DNS configuration if present
	if n.DnsConfig() != nil {
		dto.DNS = ToDnsConfigDTO(n.DnsConfig())
	}

	return dto
}

func ToNodeDTOList(nodes []*node.Node) []*NodeDTO {
	return mapper.MapSlicePtrSkipNil(nodes, ToNodeDTO)
}

// UserNodeDTO represents a user-owned node for API responses
// It contains fewer fields than NodeDTO, hiding admin-specific information
type UserNodeDTO struct {
	ID               string     `json:"id" example:"node_xK9mP2vL3nQ" description:"Unique identifier for the node"`
	Name             string     `json:"name" example:"My-Node-01" description:"Display name of the node"`
	ServerAddress    string     `json:"server_address" example:"proxy.example.com" description:"Server hostname or IP address"`
	AgentPort        uint16     `json:"agent_port" example:"8388" description:"Port for agent connections"`
	SubscriptionPort *uint16    `json:"subscription_port,omitempty" example:"8389" description:"Port for client subscriptions"`
	Protocol         string     `json:"protocol" example:"shadowsocks" enums:"shadowsocks,trojan,vless,vmess,hysteria2,tuic,anytls" description:"Proxy protocol type"`
	EncryptionMethod string     `json:"encryption_method,omitempty" example:"aes-256-gcm" description:"Encryption method (Shadowsocks only)"`
	Status           string     `json:"status" example:"active" enums:"active,inactive,maintenance" description:"Current operational status"`
	IsOnline         bool       `json:"is_online" example:"true" description:"Indicates if the node agent is online"`
	LastSeenAt       *time.Time `json:"last_seen_at,omitempty" example:"2024-01-15T14:20:00Z" description:"Last time the node agent reported status"`
	// Trojan specific fields
	TransportProtocol string `json:"transport_protocol,omitempty" example:"tcp" description:"Transport protocol for Trojan"`
	Host              string `json:"host,omitempty" example:"cdn.example.com" description:"WebSocket host or gRPC service name"`
	Path              string `json:"path,omitempty" example:"/trojan" description:"WebSocket path"`
	SNI               string `json:"sni,omitempty" example:"example.com" description:"TLS SNI"`
	AllowInsecure     bool   `json:"allow_insecure,omitempty" example:"false" description:"Allow insecure TLS"`

	// VLESS specific fields
	VLESSTransportType    string `json:"vless_transport_type,omitempty" description:"VLESS transport type"`
	VLESSFlow             string `json:"vless_flow,omitempty" description:"VLESS flow control"`
	VLESSSecurity         string `json:"vless_security,omitempty" description:"VLESS security type"`
	VLESSSni              string `json:"vless_sni,omitempty" description:"VLESS TLS SNI"`
	VLESSFingerprint      string `json:"vless_fingerprint,omitempty" description:"VLESS TLS fingerprint"`
	VLESSAllowInsecure    bool   `json:"vless_allow_insecure,omitempty" description:"VLESS allow insecure TLS"`
	VLESSHost             string `json:"vless_host,omitempty" description:"VLESS WS/H2 host header"`
	VLESSPath             string `json:"vless_path,omitempty" description:"VLESS WS/H2 path"`
	VLESSServiceName      string `json:"vless_service_name,omitempty" description:"VLESS gRPC service name"`
	VLESSRealityPublicKey string `json:"vless_reality_public_key,omitempty" description:"VLESS Reality public key"`
	VLESSRealityShortID   string `json:"vless_reality_short_id,omitempty" description:"VLESS Reality short ID"`
	VLESSRealitySpiderX   string `json:"vless_reality_spider_x,omitempty" description:"VLESS Reality spider X"`

	// VMess specific fields
	VMessAlterID       int    `json:"vmess_alter_id,omitempty" description:"VMess alter ID"`
	VMessSecurity      string `json:"vmess_security,omitempty" description:"VMess security"`
	VMessTransportType string `json:"vmess_transport_type,omitempty" description:"VMess transport type"`
	VMessHost          string `json:"vmess_host,omitempty" description:"VMess WS/HTTP host header"`
	VMessPath          string `json:"vmess_path,omitempty" description:"VMess WS/HTTP path"`
	VMessServiceName   string `json:"vmess_service_name,omitempty" description:"VMess gRPC service name"`
	VMessTLS           bool   `json:"vmess_tls,omitempty" description:"VMess TLS enabled"`
	VMessSni           string `json:"vmess_sni,omitempty" description:"VMess TLS SNI"`
	VMessAllowInsecure bool   `json:"vmess_allow_insecure,omitempty" description:"VMess allow insecure TLS"`

	// Hysteria2 specific fields
	Hysteria2CongestionControl string `json:"hysteria2_congestion_control,omitempty" description:"Hysteria2 congestion control"`
	Hysteria2Obfs              string `json:"hysteria2_obfs,omitempty" description:"Hysteria2 obfuscation type"`
	Hysteria2ObfsPassword      string `json:"hysteria2_obfs_password,omitempty" description:"Hysteria2 obfuscation password"`
	Hysteria2UpMbps            *int   `json:"hysteria2_up_mbps,omitempty" description:"Hysteria2 upstream bandwidth limit"`
	Hysteria2DownMbps          *int   `json:"hysteria2_down_mbps,omitempty" description:"Hysteria2 downstream bandwidth limit"`
	Hysteria2Sni               string `json:"hysteria2_sni,omitempty" description:"Hysteria2 TLS SNI"`
	Hysteria2AllowInsecure     bool   `json:"hysteria2_allow_insecure,omitempty" description:"Hysteria2 allow insecure TLS"`
	Hysteria2Fingerprint       string `json:"hysteria2_fingerprint,omitempty" description:"Hysteria2 TLS fingerprint"`

	// TUIC specific fields
	TUICCongestionControl string `json:"tuic_congestion_control,omitempty" description:"TUIC congestion control"`
	TUICUDPRelayMode      string `json:"tuic_udp_relay_mode,omitempty" description:"TUIC UDP relay mode"`
	TUICAlpn              string `json:"tuic_alpn,omitempty" description:"TUIC ALPN protocols"`
	TUICSni               string `json:"tuic_sni,omitempty" description:"TUIC TLS SNI"`
	TUICAllowInsecure     bool   `json:"tuic_allow_insecure,omitempty" description:"TUIC allow insecure TLS"`
	TUICDisableSNI        bool   `json:"tuic_disable_sni,omitempty" description:"TUIC disable SNI"`

	// AnyTLS specific fields
	AnyTLSSni                      string `json:"anytls_sni,omitempty" description:"AnyTLS TLS SNI"`
	AnyTLSAllowInsecure            bool   `json:"anytls_allow_insecure,omitempty" description:"AnyTLS allow insecure TLS"`
	AnyTLSFingerprint              string `json:"anytls_fingerprint,omitempty" description:"AnyTLS TLS fingerprint"`
	AnyTLSIdleSessionCheckInterval string `json:"anytls_idle_session_check_interval,omitempty" description:"AnyTLS idle session check interval"`
	AnyTLSIdleSessionTimeout       string `json:"anytls_idle_session_timeout,omitempty" description:"AnyTLS idle session timeout"`
	AnyTLSMinIdleSession           int    `json:"anytls_min_idle_session,omitempty" description:"AnyTLS minimum idle sessions"`

	CreatedAt time.Time `json:"created_at" example:"2024-01-15T10:30:00Z" description:"Timestamp when the node was created"`
	UpdatedAt time.Time `json:"updated_at" example:"2024-01-15T14:20:00Z" description:"Timestamp when the node was last updated"`
}

// ToUserNodeDTO converts a node entity to a user node DTO
func ToUserNodeDTO(n *node.Node) *UserNodeDTO {
	if n == nil {
		return nil
	}

	dto := &UserNodeDTO{
		ID:               n.SID(),
		Name:             n.Name(),
		ServerAddress:    n.ServerAddress().Value(),
		AgentPort:        n.AgentPort(),
		SubscriptionPort: n.SubscriptionPort(),
		Protocol:         n.Protocol().String(),
		EncryptionMethod: n.EncryptionConfig().Method(),
		Status:           n.Status().String(),
		IsOnline:         n.IsOnline(),
		LastSeenAt:       n.LastSeenAt(),
		CreatedAt:        n.CreatedAt(),
		UpdatedAt:        n.UpdatedAt(),
	}

	// Map Trojan specific fields
	if n.TrojanConfig() != nil {
		dto.TransportProtocol = n.TrojanConfig().TransportProtocol()
		dto.Host = n.TrojanConfig().Host()
		dto.Path = n.TrojanConfig().Path()
		dto.SNI = n.TrojanConfig().SNI()
		dto.AllowInsecure = n.TrojanConfig().AllowInsecure()
	}

	// Map VLESS specific fields
	if n.VLESSConfig() != nil {
		dto.VLESSTransportType = n.VLESSConfig().TransportType()
		dto.VLESSFlow = n.VLESSConfig().Flow()
		dto.VLESSSecurity = n.VLESSConfig().Security()
		dto.VLESSSni = n.VLESSConfig().SNI()
		dto.VLESSFingerprint = n.VLESSConfig().Fingerprint()
		dto.VLESSAllowInsecure = n.VLESSConfig().AllowInsecure()
		dto.VLESSHost = n.VLESSConfig().Host()
		dto.VLESSPath = n.VLESSConfig().Path()
		dto.VLESSServiceName = n.VLESSConfig().ServiceName()
		dto.VLESSRealityPublicKey = n.VLESSConfig().PublicKey()
		dto.VLESSRealityShortID = n.VLESSConfig().ShortID()
		dto.VLESSRealitySpiderX = n.VLESSConfig().SpiderX()
	}

	// Map VMess specific fields
	if n.VMessConfig() != nil {
		dto.VMessAlterID = n.VMessConfig().AlterID()
		dto.VMessSecurity = n.VMessConfig().Security()
		dto.VMessTransportType = n.VMessConfig().TransportType()
		dto.VMessHost = n.VMessConfig().Host()
		dto.VMessPath = n.VMessConfig().Path()
		dto.VMessServiceName = n.VMessConfig().ServiceName()
		dto.VMessTLS = n.VMessConfig().TLS()
		dto.VMessSni = n.VMessConfig().SNI()
		dto.VMessAllowInsecure = n.VMessConfig().AllowInsecure()
	}

	// Map Hysteria2 specific fields
	if n.Hysteria2Config() != nil {
		dto.Hysteria2CongestionControl = n.Hysteria2Config().CongestionControl()
		dto.Hysteria2Obfs = n.Hysteria2Config().Obfs()
		dto.Hysteria2ObfsPassword = n.Hysteria2Config().ObfsPassword()
		dto.Hysteria2UpMbps = n.Hysteria2Config().UpMbps()
		dto.Hysteria2DownMbps = n.Hysteria2Config().DownMbps()
		dto.Hysteria2Sni = n.Hysteria2Config().SNI()
		dto.Hysteria2AllowInsecure = n.Hysteria2Config().AllowInsecure()
		dto.Hysteria2Fingerprint = n.Hysteria2Config().Fingerprint()
	}

	// Map TUIC specific fields
	if n.TUICConfig() != nil {
		dto.TUICCongestionControl = n.TUICConfig().CongestionControl()
		dto.TUICUDPRelayMode = n.TUICConfig().UDPRelayMode()
		dto.TUICAlpn = n.TUICConfig().ALPN()
		dto.TUICSni = n.TUICConfig().SNI()
		dto.TUICAllowInsecure = n.TUICConfig().AllowInsecure()
		dto.TUICDisableSNI = n.TUICConfig().DisableSNI()
	}

	// Map AnyTLS specific fields
	if n.AnyTLSConfig() != nil {
		dto.AnyTLSSni = n.AnyTLSConfig().SNI()
		dto.AnyTLSAllowInsecure = n.AnyTLSConfig().AllowInsecure()
		dto.AnyTLSFingerprint = n.AnyTLSConfig().Fingerprint()
		dto.AnyTLSIdleSessionCheckInterval = n.AnyTLSConfig().IdleSessionCheckInterval()
		dto.AnyTLSIdleSessionTimeout = n.AnyTLSConfig().IdleSessionTimeout()
		dto.AnyTLSMinIdleSession = n.AnyTLSConfig().MinIdleSession()
	}

	return dto
}

// ToUserNodeDTOList converts a list of node entities to user node DTOs
func ToUserNodeDTOList(nodes []*node.Node) []*UserNodeDTO {
	return mapper.MapSlicePtrSkipNil(nodes, ToUserNodeDTO)
}
