// Package nodeutil provides utilities for building subscription nodes.
package nodeutil

import (
	"encoding/json"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// NodeSource contains the essential data needed to build a subscription node.
// This abstraction allows building nodes from both NodeModel and ForwardRule sources.
type NodeSource struct {
	ID        uint
	Name      string
	Address   string
	Port      uint16
	Protocol  string
	TokenHash string
	SortOrder int
}

// ProtocolConfigs holds loaded protocol configuration maps.
type ProtocolConfigs struct {
	Trojan      map[uint]*models.TrojanConfigModel
	Shadowsocks map[uint]*models.ShadowsocksConfigModel
}

// NewProtocolConfigs creates an empty ProtocolConfigs instance.
func NewProtocolConfigs() ProtocolConfigs {
	return ProtocolConfigs{
		Trojan:      make(map[uint]*models.TrojanConfigModel),
		Shadowsocks: make(map[uint]*models.ShadowsocksConfigModel),
	}
}

// BuildNode creates a usecases.Node from a NodeSource and applies protocol configuration.
func BuildNode(source NodeSource, configs ProtocolConfigs) *usecases.Node {
	protocol := normalizeProtocol(source.Protocol)

	node := &usecases.Node{
		ID:               source.ID,
		Name:             source.Name,
		ServerAddress:    source.Address,
		SubscriptionPort: source.Port,
		Protocol:         protocol,
		TokenHash:        source.TokenHash,
		Password:         "",
		SortOrder:        source.SortOrder,
	}

	ApplyProtocolConfig(node, protocol, source.ID, configs)
	return node
}

// normalizeProtocol returns the protocol name, defaulting to "shadowsocks" if empty.
func normalizeProtocol(protocol string) string {
	if protocol == "" {
		return "shadowsocks"
	}
	return protocol
}

// ApplyProtocolConfig applies protocol-specific configuration to a node.
func ApplyProtocolConfig(node *usecases.Node, protocol string, nodeID uint, configs ProtocolConfigs) {
	switch protocol {
	case "shadowsocks", "":
		applyShadowsocksConfig(node, nodeID, configs.Shadowsocks)
	case "trojan":
		applyTrojanConfig(node, nodeID, configs.Trojan)
	}
}

// applyShadowsocksConfig applies shadowsocks-specific configuration to a node.
func applyShadowsocksConfig(node *usecases.Node, nodeID uint, configs map[uint]*models.ShadowsocksConfigModel) {
	sc, ok := configs[nodeID]
	if !ok {
		return
	}

	node.EncryptionMethod = sc.EncryptionMethod
	if sc.Plugin != nil {
		node.Plugin = *sc.Plugin
	}
	if len(sc.PluginOpts) > 0 {
		node.PluginOpts = parsePluginOpts(sc.PluginOpts)
	}
}

// parsePluginOpts converts JSON plugin options to a string map.
func parsePluginOpts(optsJSON []byte) map[string]string {
	pluginOpts := make(map[string]string)
	var optsMap map[string]any
	if err := json.Unmarshal(optsJSON, &optsMap); err != nil {
		return pluginOpts
	}
	for key, val := range optsMap {
		if strVal, ok := val.(string); ok {
			pluginOpts[key] = strVal
		}
	}
	return pluginOpts
}

// applyTrojanConfig applies trojan-specific configuration to a node.
func applyTrojanConfig(node *usecases.Node, nodeID uint, configs map[uint]*models.TrojanConfigModel) {
	tc, ok := configs[nodeID]
	if !ok {
		// Default transport protocol for trojan
		node.TransportProtocol = "tcp"
		return
	}

	node.TransportProtocol = tc.TransportProtocol
	node.Host = tc.Host
	node.Path = tc.Path
	node.SNI = tc.SNI
	node.AllowInsecure = tc.AllowInsecure
}

// ResolveServerAddress returns the effective server address for subscription.
// If server address is configured, use it; otherwise fall back to agent's reported public IP.
func ResolveServerAddress(configuredAddr string, publicIPv4, publicIPv6 *string) string {
	if configuredAddr != "" {
		return configuredAddr
	}

	// Prefer IPv4 over IPv6 for better compatibility
	if publicIPv4 != nil && *publicIPv4 != "" {
		return *publicIPv4
	}

	if publicIPv6 != nil && *publicIPv6 != "" {
		return *publicIPv6
	}

	return ""
}

// NodeModelToSource converts a NodeModel to NodeSource for node building.
func NodeModelToSource(nm *models.NodeModel) NodeSource {
	port := nm.AgentPort
	if nm.SubscriptionPort != nil {
		port = *nm.SubscriptionPort
	}

	return NodeSource{
		ID:        nm.ID,
		Name:      nm.Name,
		Address:   ResolveServerAddress(nm.ServerAddress, nm.PublicIPv4, nm.PublicIPv6),
		Port:      port,
		Protocol:  nm.Protocol,
		TokenHash: nm.TokenHash,
		SortOrder: nm.SortOrder,
	}
}

// CopyProtocolFieldsFromNode copies protocol-related fields from one usecases.Node to another.
// This is used when building forwarded nodes that inherit protocol config from the original node.
func CopyProtocolFieldsFromNode(dst, src *usecases.Node) {
	dst.EncryptionMethod = src.EncryptionMethod
	dst.Plugin = src.Plugin
	dst.PluginOpts = src.PluginOpts
	dst.TransportProtocol = src.TransportProtocol
	dst.Host = src.Host
	dst.Path = src.Path
	dst.SNI = src.SNI
	dst.AllowInsecure = src.AllowInsecure
}
