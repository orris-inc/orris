package dto

import (
	"time"

	"github.com/orris-inc/orris/internal/domain/node"
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
	MuteNotification bool                   `json:"mute_notification" example:"false" description:"Mute online/offline notifications for this node"`
	// Trojan specific fields
	TransportProtocol string     `json:"transport_protocol,omitempty" example:"tcp" enums:"tcp,ws,grpc" description:"Transport protocol for Trojan (tcp, ws, grpc)"`
	Host              string     `json:"host,omitempty" example:"cdn.example.com" description:"WebSocket host header or gRPC service name"`
	Path              string     `json:"path,omitempty" example:"/trojan" description:"WebSocket path"`
	SNI               string     `json:"sni,omitempty" example:"example.com" description:"TLS Server Name Indication"`
	AllowInsecure     bool       `json:"allow_insecure,omitempty" example:"true" description:"Allow insecure TLS connection"`
	MaintenanceReason *string    `json:"maintenance_reason,omitempty" example:"Scheduled maintenance" description:"Reason for maintenance status (only when status is maintenance)"`
	IsOnline          bool       `json:"is_online" example:"true" description:"Indicates if the node agent is online (reported within 5 minutes)"`
	LastSeenAt        *time.Time `json:"last_seen_at,omitempty" example:"2024-01-15T14:20:00Z" description:"Last time the node agent reported status"`
	AgentVersion      string     `json:"agent_version,omitempty" example:"1.2.0" description:"Agent software version, extracted from system_status for easy display"`
	Platform          string     `json:"platform,omitempty" example:"linux" description:"OS platform (linux, darwin, windows)"`
	Arch              string     `json:"arch,omitempty" example:"amd64" description:"CPU architecture (amd64, arm64, arm, 386)"`
	HasUpdate         bool       `json:"has_update" example:"true" description:"True if a newer agent version is available"`
	GroupSIDs         []string   `json:"group_ids,omitempty" example:"[\"rg_xK9mP2vL3nQ\"]" description:"Resource group SIDs this node belongs to"`
	Version           int        `json:"version" example:"1" description:"Version number for optimistic locking"`
	CreatedAt         time.Time  `json:"created_at" example:"2024-01-15T10:30:00Z" description:"Timestamp when the node was created"`
	UpdatedAt         time.Time  `json:"updated_at" example:"2024-01-15T14:20:00Z" description:"Timestamp when the node was last updated"`
	// System status fields (from Redis)
	SystemStatus *NodeSystemStatusDTO `json:"system_status,omitempty" description:"Real-time system metrics from monitoring"`
	// Owner information (for user-created nodes)
	Owner *NodeOwnerDTO `json:"owner,omitempty" description:"Owner information for user-created nodes"`
	// Route configuration for traffic splitting (sing-box compatible)
	Route *RouteConfigDTO `json:"route,omitempty" description:"Routing configuration for traffic splitting"`
}

// NodeOwnerDTO represents the owner information for a user-created node
type NodeOwnerDTO struct {
	ID    string `json:"id" example:"user_xK9mP2vL3nQ" description:"User's Stripe-style ID"`
	Email string `json:"email" example:"user@example.com" description:"User's email address"`
	Name  string `json:"name" example:"John Doe" description:"User's display name"`
}

// NodeSystemStatusDTO represents real-time system status metrics retrieved from Redis cache
type NodeSystemStatusDTO struct {
	// System resources
	CPUPercent    float64 `json:"cpu_percent" example:"45.50"`       // CPU usage percentage (0-100)
	MemoryPercent float64 `json:"memory_percent" example:"65.30"`    // Memory usage percentage (0-100)
	MemoryUsed    uint64  `json:"memory_used" example:"4294967296"`  // Memory used in bytes
	MemoryTotal   uint64  `json:"memory_total" example:"8589934592"` // Total memory in bytes
	MemoryAvail   uint64  `json:"memory_avail" example:"2147483648"` // Available memory in bytes
	DiskPercent   float64 `json:"disk_percent" example:"80.20"`      // Disk usage percentage (0-100)
	DiskUsed      uint64  `json:"disk_used" example:"42949672960"`   // Disk used in bytes
	DiskTotal     uint64  `json:"disk_total" example:"107374182400"` // Total disk in bytes
	UptimeSeconds int64   `json:"uptime_seconds" example:"86400"`    // System uptime in seconds

	// System load
	LoadAvg1  float64 `json:"load_avg_1" example:"0.85"`  // 1-minute load average
	LoadAvg5  float64 `json:"load_avg_5" example:"0.72"`  // 5-minute load average
	LoadAvg15 float64 `json:"load_avg_15" example:"0.68"` // 15-minute load average

	// Network statistics
	NetworkRxBytes uint64 `json:"network_rx_bytes" example:"1073741824"` // Total received bytes
	NetworkTxBytes uint64 `json:"network_tx_bytes" example:"536870912"`  // Total transmitted bytes
	NetworkRxRate  uint64 `json:"network_rx_rate" example:"10485760"`    // Current receive rate in bytes per second
	NetworkTxRate  uint64 `json:"network_tx_rate" example:"5242880"`     // Current transmit rate in bytes per second

	// Connection statistics
	TCPConnections int `json:"tcp_connections" example:"150"` // Number of TCP connections
	UDPConnections int `json:"udp_connections" example:"20"`  // Number of UDP connections

	// Network info
	PublicIPv4 string `json:"public_ipv4,omitempty" example:"203.0.113.1"` // Public IPv4 address
	PublicIPv6 string `json:"public_ipv6,omitempty" example:"2001:db8::1"` // Public IPv6 address

	// Agent info
	AgentVersion string `json:"agent_version,omitempty" example:"1.2.0"` // Agent software version
	Platform     string `json:"platform,omitempty" example:"linux"`      // OS platform (linux, darwin, windows)
	Arch         string `json:"arch,omitempty" example:"amd64"`          // CPU architecture (amd64, arm64, arm, 386)

	// CPU details
	CPUCores     int     `json:"cpu_cores" example:"4"`                          // Number of CPU cores
	CPUModelName string  `json:"cpu_model_name" example:"AMD EPYC 9654 96-Core"` // CPU model name
	CPUMHz       float64 `json:"cpu_mhz" example:"2396.4"`                       // CPU frequency in MHz

	// Swap memory
	SwapTotal   uint64  `json:"swap_total" example:"8589934592"` // Total swap memory in bytes
	SwapUsed    uint64  `json:"swap_used" example:"1073741824"`  // Used swap memory in bytes
	SwapPercent float64 `json:"swap_percent" example:"12.50"`    // Swap usage percentage (0-100)

	// Disk I/O
	DiskReadBytes  uint64 `json:"disk_read_bytes" example:"1073741824"` // Total disk read bytes
	DiskWriteBytes uint64 `json:"disk_write_bytes" example:"536870912"` // Total disk write bytes
	DiskReadRate   uint64 `json:"disk_read_rate" example:"10485760"`    // Disk read rate in bytes per second
	DiskWriteRate  uint64 `json:"disk_write_rate" example:"5242880"`    // Disk write rate in bytes per second
	DiskIOPS       uint64 `json:"disk_iops" example:"1000"`             // Disk I/O operations per second

	// Pressure Stall Information (PSI)
	PSICPUSome    float64 `json:"psi_cpu_some" example:"0.50"`    // CPU pressure (some)
	PSICPUFull    float64 `json:"psi_cpu_full" example:"0.10"`    // CPU pressure (full)
	PSIMemorySome float64 `json:"psi_memory_some" example:"0.30"` // Memory pressure (some)
	PSIMemoryFull float64 `json:"psi_memory_full" example:"0.05"` // Memory pressure (full)
	PSIIOSome     float64 `json:"psi_io_some" example:"0.20"`     // I/O pressure (some)
	PSIIOFull     float64 `json:"psi_io_full" example:"0.02"`     // I/O pressure (full)

	// Network extended stats
	NetworkRxPackets uint64 `json:"network_rx_packets" example:"1000000"` // Total received packets
	NetworkTxPackets uint64 `json:"network_tx_packets" example:"500000"`  // Total transmitted packets
	NetworkRxErrors  uint64 `json:"network_rx_errors" example:"10"`       // Receive errors
	NetworkTxErrors  uint64 `json:"network_tx_errors" example:"5"`        // Transmit errors
	NetworkRxDropped uint64 `json:"network_rx_dropped" example:"2"`       // Receive dropped packets
	NetworkTxDropped uint64 `json:"network_tx_dropped" example:"1"`       // Transmit dropped packets

	// Socket statistics
	SocketsUsed      int `json:"sockets_used" example:"500"`       // Total sockets in use
	SocketsTCPInUse  int `json:"sockets_tcp_in_use" example:"300"` // TCP sockets in use
	SocketsUDPInUse  int `json:"sockets_udp_in_use" example:"50"`  // UDP sockets in use
	SocketsTCPOrphan int `json:"sockets_tcp_orphan" example:"10"`  // Orphaned TCP sockets
	SocketsTCPTW     int `json:"sockets_tcp_tw" example:"100"`     // TCP TIME_WAIT sockets

	// Process statistics
	ProcessesTotal   uint64 `json:"processes_total" example:"200"` // Total number of processes
	ProcessesRunning uint64 `json:"processes_running" example:"5"` // Running processes
	ProcessesBlocked uint64 `json:"processes_blocked" example:"0"` // Blocked processes

	// File descriptors
	FileNrAllocated uint64 `json:"file_nr_allocated" example:"10000"` // Allocated file descriptors
	FileNrMax       uint64 `json:"file_nr_max" example:"100000"`      // Maximum file descriptors

	// Context switches and interrupts
	ContextSwitches uint64 `json:"context_switches" example:"1000000"` // Context switches per second
	Interrupts      uint64 `json:"interrupts" example:"500000"`        // Interrupts per second

	// Kernel info
	KernelVersion string `json:"kernel_version,omitempty" example:"5.15.0-generic"` // Kernel version
	Hostname      string `json:"hostname,omitempty" example:"server-01"`            // Server hostname

	// Virtual memory statistics
	VMPageIn  uint64 `json:"vm_page_in" example:"1000000"` // Pages paged in
	VMPageOut uint64 `json:"vm_page_out" example:"500000"` // Pages paged out
	VMSwapIn  uint64 `json:"vm_swap_in" example:"1000"`    // Pages swapped in
	VMSwapOut uint64 `json:"vm_swap_out" example:"500"`    // Pages swapped out
	VMOOMKill uint64 `json:"vm_oom_kill" example:"0"`      // OOM killer invocations

	// Entropy pool
	EntropyAvailable uint64 `json:"entropy_available" example:"3500"` // Available entropy bits

	// Metadata
	UpdatedAt int64 `json:"updated_at" example:"1705324800"` // Last update timestamp (Unix seconds)
}

type CreateNodeDTO struct {
	Name             string                 `json:"name" binding:"required,min=2,max=100" example:"US-Node-01" description:"Display name of the node (2-100 characters)"`
	ServerAddress    string                 `json:"server_address,omitempty" example:"proxy.example.com" description:"Server hostname or IP address (optional, can be auto-detected from agent)"`
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
	Route            *RouteConfigDTO        `json:"route,omitempty" description:"Routing configuration for traffic splitting (sing-box compatible)"`
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
	MuteNotification *bool                  `json:"mute_notification,omitempty" example:"false" description:"Mute online/offline notifications for this node"`
	Route            *RouteConfigDTO        `json:"route,omitempty" description:"Routing configuration for traffic splitting (sing-box compatible, null to clear)"`
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
		Version:           n.Version(),
		CreatedAt:         n.CreatedAt(),
		UpdatedAt:         n.UpdatedAt(),
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

// UserNodeDTO represents a user-owned node for API responses
// It contains fewer fields than NodeDTO, hiding admin-specific information
type UserNodeDTO struct {
	ID                string     `json:"id" example:"node_xK9mP2vL3nQ" description:"Unique identifier for the node"`
	Name              string     `json:"name" example:"My-Node-01" description:"Display name of the node"`
	ServerAddress     string     `json:"server_address" example:"proxy.example.com" description:"Server hostname or IP address"`
	AgentPort         uint16     `json:"agent_port" example:"8388" description:"Port for agent connections"`
	SubscriptionPort  *uint16    `json:"subscription_port,omitempty" example:"8389" description:"Port for client subscriptions"`
	Protocol          string     `json:"protocol" example:"shadowsocks" enums:"shadowsocks,trojan" description:"Proxy protocol type"`
	EncryptionMethod  string     `json:"encryption_method,omitempty" example:"aes-256-gcm" description:"Encryption method (Shadowsocks only)"`
	Status            string     `json:"status" example:"active" enums:"active,inactive,maintenance" description:"Current operational status"`
	IsOnline          bool       `json:"is_online" example:"true" description:"Indicates if the node agent is online"`
	LastSeenAt        *time.Time `json:"last_seen_at,omitempty" example:"2024-01-15T14:20:00Z" description:"Last time the node agent reported status"`
	TransportProtocol string     `json:"transport_protocol,omitempty" example:"tcp" description:"Transport protocol for Trojan"`
	Host              string     `json:"host,omitempty" example:"cdn.example.com" description:"WebSocket host or gRPC service name"`
	Path              string     `json:"path,omitempty" example:"/trojan" description:"WebSocket path"`
	SNI               string     `json:"sni,omitempty" example:"example.com" description:"TLS SNI"`
	AllowInsecure     bool       `json:"allow_insecure,omitempty" example:"false" description:"Allow insecure TLS"`
	CreatedAt         time.Time  `json:"created_at" example:"2024-01-15T10:30:00Z" description:"Timestamp when the node was created"`
	UpdatedAt         time.Time  `json:"updated_at" example:"2024-01-15T14:20:00Z" description:"Timestamp when the node was last updated"`
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

	return dto
}

// ToUserNodeDTOList converts a list of node entities to user node DTOs
func ToUserNodeDTOList(nodes []*node.Node) []*UserNodeDTO {
	if nodes == nil {
		return nil
	}

	dtos := make([]*UserNodeDTO, 0, len(nodes))
	for _, n := range nodes {
		if dto := ToUserNodeDTO(n); dto != nil {
			dtos = append(dtos, dto)
		}
	}

	return dtos
}
