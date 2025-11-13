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
type NodeConfigResponse struct {
	NodeID            int    `json:"node_id" binding:"required"`                            // Node unique identifier
	NodeType          string `json:"node_type" binding:"required,oneof=shadowsocks trojan"` // Protocol type
	ServerHost        string `json:"server_host" binding:"required"`                        // Server hostname or IP address
	ServerPort        int    `json:"server_port" binding:"required,min=1,max=65535"`        // Server port number
	Method            string `json:"method,omitempty"`                                      // Encryption method for SS (e.g., "aes-256-gcm")
	ServerKey         string `json:"server_key,omitempty"`                                  // Server password for SS
	TransportProtocol string `json:"transport_protocol" binding:"required,oneof=tcp ws"`    // Transport protocol
	Host              string `json:"host,omitempty"`                                        // WebSocket host header
	Path              string `json:"path,omitempty"`                                        // WebSocket path
	EnableVless       bool   `json:"enable_vless"`                                          // Enable VLESS protocol
	EnableXTLS        bool   `json:"enable_xtls"`                                           // Enable XTLS
	SpeedLimit        uint64 `json:"speed_limit"`                                           // Speed limit in Mbps, 0 = unlimited
	DeviceLimit       int    `json:"device_limit"`                                          // Device connection limit, 0 = unlimited
	RuleListPath      string `json:"rule_list_path,omitempty"`                              // Path to routing rule list file
}

// NodeUserInfo represents individual user information for node access
// Endpoint: act=user (part of response)
type NodeUserInfo struct {
	ID          int    `json:"id" binding:"required"`          // User unique identifier
	UUID        string `json:"uuid" binding:"required"`        // For Trojan: password, For SS: could be email
	Email       string `json:"email" binding:"required,email"` // User email address
	SpeedLimit  uint64 `json:"st"`                             // Speed limit in bps (0 = unlimited)
	DeviceLimit int    `json:"dt"`                             // Device connection limit (0 = unlimited)
	ExpireTime  int64  `json:"expire_time"`                    // Unix timestamp of expiration date
}

// NodeUsersResponse represents the user list response for a node
// Endpoint: act=user
type NodeUsersResponse struct {
	Users []NodeUserInfo `json:"data" binding:"required"` // List of users authorized for this node
}

// UserTrafficItem represents traffic data for a single user
// Used in traffic reporting payload
type UserTrafficItem struct {
	UID      int   `json:"UID" binding:"required"`   // User unique identifier
	Upload   int64 `json:"Upload" binding:"min=0"`   // Upload traffic in bytes
	Download int64 `json:"Download" binding:"min=0"` // Download traffic in bytes
}

// ReportUserTrafficRequest represents user traffic report request
// Endpoint: act=submit
// Note: According to v2raysocks spec, this could be an array sent directly as body
type ReportUserTrafficRequest struct {
	Users []UserTrafficItem `json:"users" binding:"required,dive"` // Array of user traffic data
}

// ReportNodeStatusRequest represents node status report request
// Endpoint: act=nodestatus
type ReportNodeStatusRequest struct {
	CPU    string `json:"CPU" binding:"required"`  // CPU usage percentage (format: "XX%")
	Mem    string `json:"Mem" binding:"required"`  // Memory usage percentage (format: "XX%")
	Net    string `json:"Net" binding:"required"`  // Network usage (format: "XX MB")
	Disk   string `json:"Disk" binding:"required"` // Disk usage percentage (format: "XX%")
	Uptime int    `json:"Uptime" binding:"min=0"`  // System uptime in seconds
}

// OnlineUserItem represents a single online user connection
// Used in online users reporting payload
type OnlineUserItem struct {
	UID int    `json:"UID" binding:"required"` // User unique identifier
	IP  string `json:"IP" binding:"required"`  // User connection IP address
}

// ReportOnlineUsersRequest represents online users report request
// Endpoint: act=onlineusers
type ReportOnlineUsersRequest struct {
	Users []OnlineUserItem `json:"users" binding:"required,dive"` // Array of currently online users
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
		Method:            n.EncryptionConfig().Method(),
		ServerKey:         "", // Server key is not stored at node level; each user has their own subscription UUID
		TransportProtocol: "tcp", // Default to TCP, can be enhanced based on plugin config
		EnableVless:       false,
		EnableXTLS:        false,
		SpeedLimit:        0, // 0 = unlimited, can be set from node metadata
		DeviceLimit:       0, // 0 = unlimited, can be set from node metadata
		RuleListPath:      "",
	}

	// Determine node type based on encryption method or plugin
	// Shadowsocks methods: aes-128-gcm, aes-256-gcm, chacha20-ietf-poly1305, etc.
	if isSSMethod(config.Method) {
		config.NodeType = "shadowsocks"
	} else {
		config.NodeType = "trojan"
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

// ToNodeUsersResponse converts subscription list to XrayR users response
// Maps subscription entities to v2raysocks user list format
func ToNodeUsersResponse(subscriptions []*subscription.Subscription) *NodeUsersResponse {
	if subscriptions == nil {
		return &NodeUsersResponse{
			Users: []NodeUserInfo{},
		}
	}

	users := make([]NodeUserInfo, 0, len(subscriptions))
	for _, sub := range subscriptions {
		if sub == nil {
			continue
		}

		// Skip inactive subscriptions
		if !sub.IsActive() {
			continue
		}

		user := NodeUserInfo{
			ID:          int(sub.UserID()),
			UUID:        generateUserUUID(sub),
			Email:       generateUserEmail(sub),
			SpeedLimit:  0, // 0 = unlimited, can be set from subscription plan limits
			DeviceLimit: 0, // 0 = unlimited, can be set from subscription plan limits
			ExpireTime:  sub.EndDate().Unix(),
		}

		users = append(users, user)
	}

	return &NodeUsersResponse{
		Users: users,
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

// generateUserUUID generates UUID for user based on subscription
// For Trojan protocol: Uses UUID v5 (SHA-1 based) to generate deterministic UUIDs
// For Shadowsocks protocol: Could use email as identifier
//
// The UUID is generated using a custom namespace and subscription ID as the name,
// ensuring the same subscription always generates the same UUID (reproducible).
// This approach is stateless and doesn't require database storage.
func generateUserUUID(sub *subscription.Subscription) string {
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

// generateUserEmail generates email for user based on subscription
// Note: This is a placeholder implementation. In production, you should:
// 1. Inject UserRepository into the DTO conversion function
// 2. Fetch actual user email from User aggregate
// 3. Consider caching user data to avoid repeated queries
func generateUserEmail(sub *subscription.Subscription) string {
	// Temporary implementation: Generate a placeholder email
	// In production, fetch actual user email via UserRepository.GetByID()
	if sub == nil {
		return ""
	}
	// Format: user{userId}@node.local
	// This is a placeholder and should be replaced with real user email
	return fmt.Sprintf("user%d@node.local", sub.UserID())
}
