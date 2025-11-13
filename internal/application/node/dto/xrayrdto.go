package dto

import (
	"fmt"

	"github.com/google/uuid"

	"orris/internal/domain/node"
	"orris/internal/domain/subscription"
)

// V2RaySocksResponse represents the standard response format for v2raysocks API
// Follows the v2raysocks protocol specification
type V2RaySocksResponse struct {
	Data interface{} `json:"data,omitempty"` // Response payload, can be any type based on endpoint
	Ret  int         `json:"ret,omitempty"`  // Return code: 1 = success, 0 = error
	Msg  string      `json:"msg,omitempty"`  // Message describing the result or error
}

// NodeConfigResponse represents node configuration data for XrayR
// Endpoint: act=config
// Note: XrayR agent should map fields as needed (protocol -> node_type, encryption_method -> method)
type NodeConfigResponse struct {
	NodeID            int    `json:"node_id" binding:"required"`                               // Node unique identifier
	Protocol          string `json:"protocol" binding:"required,oneof=shadowsocks trojan"`     // Protocol type (XrayR should map to node_type)
	ServerHost        string `json:"server_host" binding:"required"`                           // Server hostname or IP address
	ServerPort        int    `json:"server_port" binding:"required,min=1,max=65535"`           // Server port number
	EncryptionMethod  string `json:"encryption_method,omitempty"`                              // Encryption method for SS (XrayR should map to method)
	ServerKey         string `json:"server_key,omitempty"`                                     // Server password for SS
	TransportProtocol string `json:"transport_protocol" binding:"required,oneof=tcp ws"`       // Transport protocol
	Host              string `json:"host,omitempty"`                                           // WebSocket host header
	Path              string `json:"path,omitempty"`                                           // WebSocket path
	EnableVless       bool   `json:"enable_vless"`                                             // Enable VLESS protocol
	EnableXTLS        bool   `json:"enable_xtls"`                                              // Enable XTLS
	SpeedLimit        uint64 `json:"speed_limit"`                                              // Speed limit in Mbps, 0 = unlimited
	DeviceLimit       int    `json:"device_limit"`                                             // Device connection limit, 0 = unlimited
	RuleListPath      string `json:"rule_list_path,omitempty"`                                 // Path to routing rule list file
}

// NodeSubscriptionInfo represents individual subscription information for node access
// Endpoint: act=user (part of response)
// Note: SubscriptionID field represents subscription_id for traffic tracking purposes
type NodeSubscriptionInfo struct {
	SubscriptionID int    `json:"subscription_id" binding:"required"` // Subscription ID (used for traffic reporting)
	UUID           string `json:"uuid" binding:"required"`            // For Trojan: password, For SS: could be email
	Email          string `json:"email" binding:"required,email"`     // User email address
	SpeedLimit     uint64 `json:"speed_limit"`                        // Speed limit in bps (0 = unlimited)
	DeviceLimit    int    `json:"device_limit"`                       // Device connection limit (0 = unlimited)
	ExpireTime     int64  `json:"expire_time"`                        // Unix timestamp of expiration date
}

// NodeSubscriptionsResponse represents the subscription list response for a node
// Endpoint: act=user
type NodeSubscriptionsResponse struct {
	Subscriptions []NodeSubscriptionInfo `json:"subscriptions" binding:"required"` // List of subscriptions authorized for this node
}

// SubscriptionTrafficItem represents traffic data for a single subscription
// Used in traffic reporting payload
type SubscriptionTrafficItem struct {
	SubscriptionID int   `json:"subscription_id" binding:"required"` // Subscription ID for traffic tracking
	Upload         int64 `json:"upload" binding:"min=0"`             // Upload traffic in bytes
	Download       int64 `json:"download" binding:"min=0"`           // Download traffic in bytes
}

// ReportSubscriptionTrafficRequest represents subscription traffic report request
// Endpoint: act=submit
// Note: According to v2raysocks spec, this could be an array sent directly as body
type ReportSubscriptionTrafficRequest struct {
	Subscriptions []SubscriptionTrafficItem `json:"subscriptions" binding:"required,dive"` // Array of subscription traffic data
}

// ReportNodeStatusRequest represents node status report request
// Endpoint: act=nodestatus
type ReportNodeStatusRequest struct {
	CPU    string `json:"CPU" binding:"required"`  // CPU usage percentage (format: "XX%")
	Mem    string `json:"Mem" binding:"required"`  // Memory usage percentage (format: "XX%")
	Disk   string `json:"Disk" binding:"required"` // Disk usage percentage (format: "XX%")
	Uptime int    `json:"Uptime" binding:"min=0"`  // System uptime in seconds
}

// OnlineSubscriptionItem represents a single online subscription connection
// Used in online subscriptions reporting payload
type OnlineSubscriptionItem struct {
	SubscriptionID int    `json:"subscription_id" binding:"required"` // Subscription unique identifier
	IP             string `json:"ip" binding:"required"`              // Connection IP address
}

// ReportOnlineSubscriptionsRequest represents online subscriptions report request
// Endpoint: act=onlineusers
type ReportOnlineSubscriptionsRequest struct {
	Subscriptions []OnlineSubscriptionItem `json:"subscriptions" binding:"required,dive"` // Array of currently online subscriptions
}

// ToNodeConfigResponse converts a domain node entity to XrayR node config response
// Maps internal node model to v2raysocks protocol format
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

// ToNodeSubscriptionsResponse converts subscription list to XrayR subscriptions response
// Maps subscription entities to v2raysocks user list format
func ToNodeSubscriptionsResponse(subscriptions []*subscription.Subscription) *NodeSubscriptionsResponse {
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

		// IMPORTANT: The SubscriptionID field represents subscription_id for traffic tracking
		// XrayR uses this ID to report traffic, and traffic is tracked per subscription
		// This allows proper traffic management when a user has multiple subscriptions
		subscriptionInfo := NodeSubscriptionInfo{
			SubscriptionID: int(sub.ID()), // Using subscription ID for traffic tracking
			UUID:           generateSubscriptionUUID(sub),
			Email:          generateSubscriptionEmail(sub),
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

// NewSuccessResponse creates a success response for v2raysocks API
func NewSuccessResponse(data interface{}) *V2RaySocksResponse {
	return &V2RaySocksResponse{
		Data: data,
		Ret:  1,
		Msg:  "success",
	}
}

// NewErrorResponse creates an error response for v2raysocks API
func NewErrorResponse(msg string) *V2RaySocksResponse {
	return &V2RaySocksResponse{
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

// generateSubscriptionUUID generates UUID for subscription
// For Trojan protocol: Uses UUID v5 (SHA-1 based) to generate deterministic UUIDs
// For Shadowsocks protocol: Could use email as identifier
//
// The UUID is generated using a custom namespace and subscription ID as the name,
// ensuring the same subscription always generates the same UUID (reproducible).
// This approach is stateless and doesn't require database storage.
func generateSubscriptionUUID(sub *subscription.Subscription) string {
	if sub == nil {
		return ""
	}

	// Custom namespace UUID for Orris subscription system
	// This namespace is derived from DNS namespace with "orris.subscription" domain
	namespace := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("orris.subscription"))

	// Generate deterministic UUID v5 based on subscription ID
	// Same subscription ID will always produce the same UUID
	subUUID := uuid.NewSHA1(namespace, []byte(fmt.Sprintf("sub_%d", sub.ID())))

	return subUUID.String()
}

// generateSubscriptionEmail generates email identifier based on subscription
// Uses combination of user ID and subscription ID to ensure uniqueness
// Format: user{userId}-sub{subscriptionId}@node.local
//
// This ensures each subscription has a unique identifier even when
// a user has multiple subscriptions, maintaining subscription independence.
//
// Note: In production, consider:
// 1. Inject UserRepository to fetch actual user email
// 2. Append subscription ID to real email: "{realEmail}-sub{id}@node.local"
// 3. Cache user data to avoid repeated queries
func generateSubscriptionEmail(sub *subscription.Subscription) string {
	if sub == nil {
		return ""
	}
	// Format: user{userId}-sub{subscriptionId}@node.local
	// This maintains subscription independence while showing user relationship
	return fmt.Sprintf("user%d-sub%d@node.local", sub.UserID(), sub.ID())
}
