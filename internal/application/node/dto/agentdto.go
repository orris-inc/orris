package dto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/subscription"
)

// AgentResponse represents the standard response format for agent API
type AgentResponse struct {
	Data interface{} `json:"data,omitempty"` // Response payload, can be any type based on endpoint
	Ret  int         `json:"ret,omitempty"`  // Return code: 1 = success, 0 = error
	Msg  string      `json:"msg,omitempty"`  // Message describing the result or error
}

// NodeConfigResponse represents node configuration data for node agents
type NodeConfigResponse struct {
	NodeID            int    `json:"node_id" binding:"required"`                           // Node unique identifier
	Protocol          string `json:"protocol" binding:"required,oneof=shadowsocks trojan"` // Protocol type
	ServerHost        string `json:"server_host" binding:"required"`                       // Server hostname or IP address
	ServerPort        int    `json:"server_port" binding:"required,min=1,max=65535"`       // Server port number
	EncryptionMethod  string `json:"encryption_method,omitempty"`                          // Encryption method for Shadowsocks
	ServerKey         string `json:"server_key,omitempty"`                                 // Server password for SS
	TransportProtocol string `json:"transport_protocol" binding:"required,oneof=tcp ws"`   // Transport protocol
	Host              string `json:"host,omitempty"`                                       // WebSocket host header
	Path              string `json:"path,omitempty"`                                       // WebSocket path
	EnableVless       bool   `json:"enable_vless"`                                         // Enable VLESS protocol
	EnableXTLS        bool   `json:"enable_xtls"`                                          // Enable XTLS
	SpeedLimit        uint64 `json:"speed_limit"`                                          // Speed limit in Mbps, 0 = unlimited
	DeviceLimit       int    `json:"device_limit"`                                         // Device connection limit, 0 = unlimited
	RuleListPath      string `json:"rule_list_path,omitempty"`                             // Path to routing rule list file
}

// NodeSubscriptionInfo represents individual subscription information for node access
type NodeSubscriptionInfo struct {
	SubscriptionID int    `json:"subscription_id" binding:"required"` // Subscription ID (used for traffic reporting)
	Password       string `json:"password" binding:"required"`        // HMAC-SHA256 signed password derived from subscription UUID
	Name           string `json:"name" binding:"required"`            // User identifier for logging (sing-box compatible)
	SpeedLimit     uint64 `json:"speed_limit"`                        // Speed limit in bps (0 = unlimited)
	DeviceLimit    int    `json:"device_limit"`                       // Device connection limit (0 = unlimited)
	ExpireTime     int64  `json:"expire_time"`                        // Unix timestamp of expiration date
}

// NodeSubscriptionsResponse represents the subscription list response for a node
type NodeSubscriptionsResponse struct {
	Subscriptions []NodeSubscriptionInfo `json:"subscriptions" binding:"required"` // List of subscriptions authorized for this node
}

// SubscriptionTrafficItem represents traffic data for a single subscription
type SubscriptionTrafficItem struct {
	SubscriptionID int   `json:"subscription_id" binding:"required"` // Subscription ID for traffic tracking
	Upload         int64 `json:"upload" binding:"min=0"`             // Upload traffic in bytes
	Download       int64 `json:"download" binding:"min=0"`           // Download traffic in bytes
}

// ReportSubscriptionTrafficRequest represents subscription traffic report request
type ReportSubscriptionTrafficRequest struct {
	Subscriptions []SubscriptionTrafficItem `json:"subscriptions" binding:"required,dive"` // Array of subscription traffic data
}

// ReportNodeStatusRequest represents node status report request
type ReportNodeStatusRequest struct {
	CPU    string `json:"CPU" binding:"required"`  // CPU usage percentage (format: "XX%")
	Mem    string `json:"Mem" binding:"required"`  // Memory usage percentage (format: "XX%")
	Disk   string `json:"Disk" binding:"required"` // Disk usage percentage (format: "XX%")
	Uptime int    `json:"Uptime" binding:"min=0"`  // System uptime in seconds
}

// OnlineSubscriptionItem represents a single online subscription connection
type OnlineSubscriptionItem struct {
	SubscriptionID int    `json:"subscription_id" binding:"required"` // Subscription unique identifier
	IP             string `json:"ip" binding:"required"`              // Connection IP address
}

// ReportOnlineSubscriptionsRequest represents online subscriptions report request
type ReportOnlineSubscriptionsRequest struct {
	Subscriptions []OnlineSubscriptionItem `json:"subscriptions" binding:"required,dive"` // Array of currently online subscriptions
}

// ToNodeConfigResponse converts a domain node entity to agent node config response
func ToNodeConfigResponse(n *node.Node) *NodeConfigResponse {
	if n == nil {
		return nil
	}

	config := &NodeConfigResponse{
		NodeID:            int(n.ID()),
		ServerHost:        n.ServerAddress().Value(),
		ServerPort:        int(n.ServerPort()),
		EncryptionMethod:  n.EncryptionConfig().Method(),
		ServerKey:         "",    // Server key is not stored at node level; each user has their own subscription UUID
		TransportProtocol: "tcp", // Default to TCP, can be enhanced based on plugin config
		EnableVless:       false,
		EnableXTLS:        false,
		SpeedLimit:        0, // 0 = unlimited, can be set from node metadata
		DeviceLimit:       0, // 0 = unlimited, can be set from node metadata
		RuleListPath:      "",
	}

	// Determine protocol type based on encryption method or plugin
	// Shadowsocks methods: aes-128-gcm, aes-256-gcm, chacha20-ietf-poly1305, etc.
	if isSSMethod(config.EncryptionMethod) {
		config.Protocol = "shadowsocks"
	} else {
		config.Protocol = "trojan"
	}

	// Handle plugin configuration for transport protocol
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

	return config
}

// ToNodeSubscriptionsResponse converts subscription list to agent subscriptions response
// The hmacSecret is used to generate HMAC-signed passwords from subscription UUIDs
func ToNodeSubscriptionsResponse(subscriptions []*subscription.Subscription, hmacSecret string) *NodeSubscriptionsResponse {
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
			SubscriptionID: int(sub.ID()), // Using subscription ID for traffic tracking
			Password:       generateSubscriptionPassword(sub, hmacSecret),
			Name:           generateSubscriptionName(sub),
			SpeedLimit:     0, // 0 = unlimited, can be set from subscription plan limits
			DeviceLimit:    0, // 0 = unlimited, can be set from subscription plan limits
			ExpireTime:     sub.EndDate().Unix(),
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

// isSSMethod checks if the encryption method is a Shadowsocks method
func isSSMethod(method string) bool {
	ssMethods := map[string]bool{
		"aes-128-gcm":             true,
		"aes-256-gcm":             true,
		"aes-128-cfb":             true,
		"aes-192-cfb":             true,
		"aes-256-cfb":             true,
		"aes-128-ctr":             true,
		"aes-192-ctr":             true,
		"aes-256-ctr":             true,
		"chacha20-ietf":           true,
		"chacha20-ietf-poly1305":  true,
		"xchacha20-ietf-poly1305": true,
		"rc4-md5":                 true,
	}
	return ssMethods[method]
}

// generateSubscriptionPassword generates HMAC-signed password for subscription
// Uses HMAC-SHA256 to sign the subscription UUID with a secret key
// This ensures the password is derived from the UUID but not directly exposed
//
// The password generation is deterministic: same UUID + secret always produces same password
// This allows agents to authenticate users without storing plain UUIDs
func generateSubscriptionPassword(sub *subscription.Subscription, secret string) string {
	if sub == nil || sub.UUID() == "" {
		return ""
	}

	// Use HMAC-SHA256 to generate password from subscription UUID
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sub.UUID()))

	return hex.EncodeToString(mac.Sum(nil))
}

// generateSubscriptionName generates name identifier for subscription (sing-box compatible)
// Format: user{userId}-sub{subscriptionId}
func generateSubscriptionName(sub *subscription.Subscription) string {
	if sub == nil {
		return ""
	}
	return fmt.Sprintf("user%d-sub%d", sub.UserID(), sub.ID())
}
