package node

import (
	"fmt"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/domain/shared"
)

// IsAvailable checks if node is available for use
func (n *Node) IsAvailable() bool {
	return n.status == vo.NodeStatusActive
}

// IsOnline checks if node agent is online (reported within 5 minutes)
func (n *Node) IsOnline() bool {
	return shared.IsOnline(n.lastSeenAt)
}

// IsExpired checks if the node has expired
func (n *Node) IsExpired() bool {
	return shared.IsExpired(n.expiresAt)
}

// IsExpiringSoon checks if the node will expire within the specified number of days
func (n *Node) IsExpiringSoon(days int) bool {
	return shared.IsExpiringSoon(n.expiresAt, days)
}

// HasRouteConfig checks if the node has a routing configuration
func (n *Node) HasRouteConfig() bool {
	return n.routeConfig != nil
}

// EffectiveServerAddress returns the server address to use for outbound connections.
// If serverAddress is configured, it returns that; otherwise, it falls back to publicIPv4.
// Returns empty string if neither is available.
func (n *Node) EffectiveServerAddress() string {
	if n.serverAddress.Value() != "" {
		return n.serverAddress.Value()
	}
	if n.publicIPv4 != nil && *n.publicIPv4 != "" {
		return *n.publicIPv4
	}
	return ""
}

// GenerateSubscriptionURI generates a subscription URI for this node
// The password parameter should be the subscription UUID
// Uses EffectiveSubscriptionPort() for the port (subscriptionPort if set, otherwise agentPort)
func (n *Node) GenerateSubscriptionURI(password string, remarks string) (string, error) {
	factory := vo.NewProtocolConfigFactory()
	serverAddr := n.serverAddress.Value()
	port := n.EffectiveSubscriptionPort()

	switch n.protocol {
	case vo.ProtocolShadowsocks:
		ssConfig := vo.NewShadowsocksProtocolConfig(n.encryptionConfig, n.pluginConfig)
		return factory.GenerateSubscriptionURI(n.protocol, ssConfig, serverAddr, port, password, remarks)

	case vo.ProtocolTrojan:
		if n.trojanConfig == nil {
			return "", fmt.Errorf("trojan config is required for Trojan protocol")
		}
		trojanConfig := vo.NewTrojanProtocolConfig(*n.trojanConfig)
		return factory.GenerateSubscriptionURI(n.protocol, trojanConfig, serverAddr, port, password, remarks)

	case vo.ProtocolAnyTLS:
		if n.anytlsConfig == nil {
			return "", fmt.Errorf("anytls config is required for AnyTLS protocol")
		}
		return n.anytlsConfig.ToURI(serverAddr, port, remarks, password), nil

	default:
		return "", fmt.Errorf("unsupported protocol: %s", n.protocol)
	}
}
