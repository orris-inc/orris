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
	NodeSID           string `json:"node_id" binding:"required"`                              // Node SID (Stripe-style: node_xxx)
	Protocol          string `json:"protocol" binding:"required,oneof=shadowsocks trojan"`    // Protocol type
	ServerHost        string `json:"server_host" binding:"required"`                          // Server hostname or IP address
	ServerPort        int    `json:"server_port" binding:"required,min=1,max=65535"`          // Server port number
	EncryptionMethod  string `json:"encryption_method,omitempty"`                             // Encryption method for Shadowsocks
	ServerKey         string `json:"server_key,omitempty"`                                    // Server password for SS
	TransportProtocol string `json:"transport_protocol" binding:"required,oneof=tcp ws grpc"` // Transport protocol (tcp, ws, grpc)
	Host              string `json:"host,omitempty"`                                          // WebSocket host header
	Path              string `json:"path,omitempty"`                                          // WebSocket path
	ServiceName       string `json:"service_name,omitempty"`                                  // gRPC service name
	SNI               string `json:"sni,omitempty"`                                           // TLS Server Name Indication
	AllowInsecure     bool   `json:"allow_insecure"`                                          // Allow insecure TLS connection
	EnableVless       bool   `json:"enable_vless"`                                            // Enable VLESS protocol
	EnableXTLS        bool   `json:"enable_xtls"`                                             // Enable XTLS
	SpeedLimit        uint64 `json:"speed_limit"`                                             // Speed limit in Mbps, 0 = unlimited
	DeviceLimit       int    `json:"device_limit"`                                            // Device connection limit, 0 = unlimited
	RuleListPath      string `json:"rule_list_path,omitempty"`                                // Path to routing rule list file
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

// ReportNodeStatusRequest represents node status report request
type ReportNodeStatusRequest struct {
	CPU        string `json:"CPU" binding:"required"`  // CPU usage percentage (format: "XX%")
	Mem        string `json:"Mem" binding:"required"`  // Memory usage percentage (format: "XX%")
	Disk       string `json:"Disk" binding:"required"` // Disk usage percentage (format: "XX%")
	Uptime     int    `json:"Uptime" binding:"min=0"`  // System uptime in seconds
	PublicIPv4 string `json:"public_ipv4,omitempty"`   // Public IPv4 address reported by agent
	PublicIPv6 string `json:"public_ipv6,omitempty"`   // Public IPv6 address reported by agent
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

// ToNodeConfigResponse converts a domain node entity to agent node config response
// Supports both Shadowsocks and Trojan protocols with sing-box compatible configuration
func ToNodeConfigResponse(n *node.Node) *NodeConfigResponse {
	if n == nil {
		return nil
	}

	config := &NodeConfigResponse{
		NodeSID:           n.SID(),
		ServerHost:        n.ServerAddress().Value(),
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

	return config
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

	// Check if SS2022 method
	if vo.IsSS2022Method(method) {
		// SS2022 requires base64-encoded fixed-length key
		keySize := vo.GetSS2022KeySize(method)
		if keySize == 0 || keySize > len(keyMaterial) {
			// Invalid key size, fallback to hex encoding
			return hex.EncodeToString(keyMaterial)
		}

		// Use first N bytes of key material and encode with base64
		key := keyMaterial[:keySize]
		return base64.StdEncoding.EncodeToString(key)
	}

	// Traditional SS: hex-encoded (backward compatible)
	return hex.EncodeToString(keyMaterial)
}
