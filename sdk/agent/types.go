// Package agent provides a Go SDK for interacting with the Orris Agent API.
package agent

// NodeConfig represents the node configuration returned by the API.
// Compatible with sing-box inbound configuration.
type NodeConfig struct {
	NodeID            int    `json:"node_id"`
	Protocol          string `json:"protocol"`                    // shadowsocks or trojan
	ServerHost        string `json:"server_host"`                 // Server hostname or IP address
	ServerPort        int    `json:"server_port"`                 // Server port number
	EncryptionMethod  string `json:"encryption_method,omitempty"` // Encryption method for Shadowsocks
	ServerKey         string `json:"server_key,omitempty"`        // Server password for SS
	TransportProtocol string `json:"transport_protocol"`          // Transport protocol (tcp, ws, grpc)
	Host              string `json:"host,omitempty"`              // WebSocket host header
	Path              string `json:"path,omitempty"`              // WebSocket path
	ServiceName       string `json:"service_name,omitempty"`      // gRPC service name
	SNI               string `json:"sni,omitempty"`               // TLS Server Name Indication
	AllowInsecure     bool   `json:"allow_insecure"`              // Allow insecure TLS connection
	EnableVless       bool   `json:"enable_vless"`
	EnableXTLS        bool   `json:"enable_xtls"`
	SpeedLimit        uint64 `json:"speed_limit"`
	DeviceLimit       int    `json:"device_limit"`
	RuleListPath      string `json:"rule_list_path,omitempty"`
}

// IsTrojan returns true if the node is configured for Trojan protocol.
func (c *NodeConfig) IsTrojan() bool {
	return c.Protocol == "trojan"
}

// IsShadowsocks returns true if the node is configured for Shadowsocks protocol.
func (c *NodeConfig) IsShadowsocks() bool {
	return c.Protocol == "shadowsocks"
}

// Subscription represents an individual subscription authorized for the node.
type Subscription struct {
	SubscriptionID int    `json:"subscription_id"`
	Password       string `json:"password"`
	Name           string `json:"name"`
	SpeedLimit     uint64 `json:"speed_limit"`
	DeviceLimit    int    `json:"device_limit"`
	ExpireTime     int64  `json:"expire_time"`
}

// TrafficReport represents traffic data for a single subscription.
type TrafficReport struct {
	SubscriptionID int   `json:"subscription_id"`
	Upload         int64 `json:"upload"`
	Download       int64 `json:"download"`
}

// NodeStatus represents the system status of a node.
type NodeStatus struct {
	CPU    string `json:"CPU"`
	Mem    string `json:"Mem"`
	Disk   string `json:"Disk"`
	Uptime int    `json:"Uptime"`
}

// OnlineSubscription represents an online subscription connection.
type OnlineSubscription struct {
	SubscriptionID int    `json:"subscription_id"`
	IP             string `json:"ip"`
}

// TrafficReportResult represents the result of a traffic report.
type TrafficReportResult struct {
	SubscriptionsUpdated int `json:"subscriptions_updated"`
}

// OnlineSubscriptionsResult represents the result of updating online subscriptions.
type OnlineSubscriptionsResult struct {
	OnlineCount int `json:"online_count"`
}

// apiResponse represents the standard API response structure.
type apiResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}
