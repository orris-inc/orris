package dto

import (
	"time"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/id"
)

type NodeDTO struct {
	ID               string                 `json:"id" example:"node_xK9mP2vL3nQ" description:"Unique identifier for the node (Stripe-style prefixed ID)"`
	Name             string                 `json:"name" example:"US-Node-01" description:"Display name of the node"`
	ServerAddress    string                 `json:"server_address" example:"proxy.example.com" description:"Server hostname or IP address"`
	AgentPort        uint16                 `json:"agent_port" example:"8388" description:"Port for agent connections"`
	SubscriptionPort *uint16                `json:"subscription_port,omitempty" example:"8389" description:"Port for client subscriptions (if null, uses agent_port)"`
	Protocol         string                 `json:"protocol" example:"shadowsocks" enums:"shadowsocks,trojan" description:"Proxy protocol type"`
	EncryptionMethod string                 `json:"encryption_method" example:"aes-256-gcm" enums:"aes-256-gcm,aes-128-gcm,chacha20-ietf-poly1305" description:"Encryption method for the proxy connection"`
	Plugin           string                 `json:"plugin,omitempty" example:"obfs-local" description:"Optional plugin name"`
	PluginOpts       map[string]string      `json:"plugin_opts,omitempty" example:"obfs:http,obfs-host:example.com" description:"Plugin configuration options"`
	Status           string                 `json:"status" example:"active" enums:"active,inactive,maintenance" description:"Current operational status of the node"`
	Region           string                 `json:"region,omitempty" example:"us-west" description:"Geographic region or location identifier"`
	Tags             []string               `json:"tags,omitempty" example:"premium,fast" description:"Custom tags for categorization"`
	CustomFields     map[string]interface{} `json:"custom_fields,omitempty" description:"Additional custom metadata fields"`
	SortOrder        int                    `json:"sort_order" example:"100" description:"Display order for sorting nodes"`
	// Trojan specific fields
	TransportProtocol string     `json:"transport_protocol,omitempty" example:"tcp" enums:"tcp,ws,grpc" description:"Transport protocol for Trojan (tcp, ws, grpc)"`
	Host              string     `json:"host,omitempty" example:"cdn.example.com" description:"WebSocket host header or gRPC service name"`
	Path              string     `json:"path,omitempty" example:"/trojan" description:"WebSocket path"`
	SNI               string     `json:"sni,omitempty" example:"example.com" description:"TLS Server Name Indication"`
	AllowInsecure     bool       `json:"allow_insecure,omitempty" example:"true" description:"Allow insecure TLS connection"`
	MaintenanceReason *string    `json:"maintenance_reason,omitempty" example:"Scheduled maintenance" description:"Reason for maintenance status (only when status is maintenance)"`
	IsOnline          bool       `json:"is_online" example:"true" description:"Indicates if the node agent is online (reported within 5 minutes)"`
	LastSeenAt        *time.Time `json:"last_seen_at,omitempty" example:"2024-01-15T14:20:00Z" description:"Last time the node agent reported status"`
	Version           int        `json:"version" example:"1" description:"Version number for optimistic locking"`
	CreatedAt         time.Time  `json:"created_at" example:"2024-01-15T10:30:00Z" description:"Timestamp when the node was created"`
	UpdatedAt         time.Time  `json:"updated_at" example:"2024-01-15T14:20:00Z" description:"Timestamp when the node was last updated"`
	// System status fields (from Redis)
	SystemStatus *NodeSystemStatusDTO `json:"system_status,omitempty" description:"Real-time system metrics from monitoring"`
}

// NodeSystemStatusDTO represents real-time system status metrics
type NodeSystemStatusDTO struct {
	CPU        string `json:"cpu" example:"45.50" description:"CPU usage percentage"`
	Memory     string `json:"memory" example:"65.30" description:"Memory usage percentage"`
	Disk       string `json:"disk" example:"80.20" description:"Disk usage percentage"`
	Uptime     int    `json:"uptime" example:"86400" description:"Uptime in seconds"`
	UpdatedAt  int64  `json:"updated_at" example:"1705324800" description:"Last update timestamp (Unix)"`
	PublicIPv4 string `json:"public_ipv4,omitempty" example:"203.0.113.1" description:"Public IPv4 address reported by agent"`
	PublicIPv6 string `json:"public_ipv6,omitempty" example:"2001:db8::1" description:"Public IPv6 address reported by agent"`
}

type CreateNodeDTO struct {
	Name             string                 `json:"name" binding:"required,min=2,max=100" example:"US-Node-01" description:"Display name of the node (2-100 characters)"`
	ServerAddress    string                 `json:"server_address" binding:"required" example:"proxy.example.com" description:"Server hostname or IP address"`
	AgentPort        uint16                 `json:"agent_port" binding:"required,min=1,max=65535" example:"8388" description:"Port for agent connections (1-65535)"`
	SubscriptionPort *uint16                `json:"subscription_port,omitempty" binding:"omitempty,min=1,max=65535" example:"8389" description:"Port for client subscriptions (if null, uses agent_port)"`
	EncryptionMethod string                 `json:"encryption_method" binding:"required" example:"aes-256-gcm" enums:"aes-256-gcm,aes-128-gcm,chacha20-ietf-poly1305" description:"Encryption method for the proxy connection"`
	Password         string                 `json:"password" binding:"required" example:"mySecurePassword123" description:"Authentication password"`
	Plugin           string                 `json:"plugin,omitempty" example:"obfs-local" description:"Optional plugin name"`
	PluginOpts       map[string]string      `json:"plugin_opts,omitempty" example:"obfs:http,obfs-host:example.com" description:"Plugin configuration options"`
	Region           string                 `json:"region,omitempty" example:"us-west" description:"Geographic region or location identifier"`
	Tags             []string               `json:"tags,omitempty" example:"premium,fast" description:"Custom tags for categorization"`
	CustomFields     map[string]interface{} `json:"custom_fields,omitempty" description:"Additional custom metadata fields"`
	SortOrder        int                    `json:"sort_order" example:"100" description:"Display order for sorting nodes"`
}

type UpdateNodeDTO struct {
	Name             *string                `json:"name,omitempty" binding:"omitempty,min=2,max=100" example:"US-Node-01" description:"Display name of the node (2-100 characters)"`
	ServerAddress    *string                `json:"server_address,omitempty" example:"proxy.example.com" description:"Server hostname or IP address"`
	AgentPort        *uint16                `json:"agent_port,omitempty" binding:"omitempty,min=1,max=65535" example:"8388" description:"Port for agent connections (1-65535)"`
	SubscriptionPort *uint16                `json:"subscription_port,omitempty" binding:"omitempty,min=1,max=65535" example:"8389" description:"Port for client subscriptions"`
	EncryptionMethod *string                `json:"encryption_method,omitempty" example:"aes-256-gcm" enums:"aes-256-gcm,aes-128-gcm,chacha20-ietf-poly1305" description:"Encryption method for the proxy connection"`
	Password         *string                `json:"password,omitempty" example:"mySecurePassword123" description:"Authentication password"`
	Plugin           *string                `json:"plugin,omitempty" example:"obfs-local" description:"Optional plugin name"`
	PluginOpts       map[string]string      `json:"plugin_opts,omitempty" example:"obfs:http,obfs-host:example.com" description:"Plugin configuration options"`
	Region           *string                `json:"region,omitempty" example:"us-west" description:"Geographic region or location identifier"`
	Tags             []string               `json:"tags,omitempty" example:"premium,fast" description:"Custom tags for categorization"`
	CustomFields     map[string]interface{} `json:"custom_fields,omitempty" description:"Additional custom metadata fields"`
	SortOrder        *int                   `json:"sort_order,omitempty" example:"100" description:"Display order for sorting nodes"`
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

	dto := &NodeDTO{
		ID:                id.FormatNodeID(n.ShortID()),
		Name:              n.Name(),
		ServerAddress:     n.ServerAddress().Value(),
		AgentPort:         n.AgentPort(),
		SubscriptionPort:  n.SubscriptionPort(),
		Protocol:          n.Protocol().String(),
		EncryptionMethod:  n.EncryptionConfig().Method(),
		Status:            n.Status().String(),
		SortOrder:         n.SortOrder(),
		MaintenanceReason: n.MaintenanceReason(),
		IsOnline:          n.IsOnline(),
		LastSeenAt:        n.LastSeenAt(),
		Version:           n.Version(),
		CreatedAt:         n.CreatedAt(),
		UpdatedAt:         n.UpdatedAt(),
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

	metadata := n.Metadata()
	if metadata.Region() != "" {
		dto.Region = metadata.Region()
	}
	if len(metadata.Tags()) > 0 {
		dto.Tags = metadata.Tags()
	}

	return dto
}

func ToNodeDTOList(nodes []*node.Node) []*NodeDTO {
	if nodes == nil {
		return nil
	}

	dtos := make([]*NodeDTO, 0, len(nodes))
	for _, n := range nodes {
		if dto := ToNodeDTO(n); dto != nil {
			dtos = append(dtos, dto)
		}
	}

	return dtos
}
