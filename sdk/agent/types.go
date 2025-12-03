// Package agent provides a Go SDK for interacting with the Orris Agent API.
package agent

// NodeConfig represents the node configuration returned by the API.
type NodeConfig struct {
	NodeID            int    `json:"node_id"`
	Protocol          string `json:"protocol"`
	ServerHost        string `json:"server_host"`
	ServerPort        int    `json:"server_port"`
	EncryptionMethod  string `json:"encryption_method,omitempty"`
	ServerKey         string `json:"server_key,omitempty"`
	TransportProtocol string `json:"transport_protocol"`
	Host              string `json:"host,omitempty"`
	Path              string `json:"path,omitempty"`
	EnableVless       bool   `json:"enable_vless"`
	EnableXTLS        bool   `json:"enable_xtls"`
	SpeedLimit        uint64 `json:"speed_limit"`
	DeviceLimit       int    `json:"device_limit"`
	RuleListPath      string `json:"rule_list_path,omitempty"`
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
