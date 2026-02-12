package node

import (
	"fmt"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/domain/shared"
	"github.com/orris-inc/orris/internal/shared/biztime"
)

// --- Configuration mutations ---

// UpdateServerAddress updates the server address
func (n *Node) UpdateServerAddress(address vo.ServerAddress) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.serverAddress.Value() == address.Value() {
		return nil
	}

	n.serverAddress = address
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateAgentPort updates the agent connection port
func (n *Node) UpdateAgentPort(port uint16) error {
	if port == 0 {
		return fmt.Errorf("agent port cannot be zero")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.agentPort == port {
		return nil
	}

	n.agentPort = port
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateSubscriptionPort updates the subscription port
func (n *Node) UpdateSubscriptionPort(port *uint16) error {
	if port != nil && *port == 0 {
		return fmt.Errorf("subscription port cannot be zero")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// Check if values are equal
	if n.subscriptionPort == nil && port == nil {
		return nil
	}
	if n.subscriptionPort != nil && port != nil && *n.subscriptionPort == *port {
		return nil
	}

	n.subscriptionPort = port
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateEncryption updates the encryption configuration
func (n *Node) UpdateEncryption(config vo.EncryptionConfig) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.encryptionConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdatePlugin updates the plugin configuration
func (n *Node) UpdatePlugin(config *vo.PluginConfig) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.pluginConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateTrojanConfig updates the trojan configuration
func (n *Node) UpdateTrojanConfig(config *vo.TrojanConfig) error {
	if !n.protocol.IsTrojan() {
		return fmt.Errorf("cannot update trojan config for non-trojan protocol")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.trojanConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateVLESSConfig updates the VLESS configuration
func (n *Node) UpdateVLESSConfig(config *vo.VLESSConfig) error {
	if !n.protocol.IsVLESS() {
		return fmt.Errorf("cannot update vless config for non-vless protocol")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.vlessConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateVMessConfig updates the VMess configuration
func (n *Node) UpdateVMessConfig(config *vo.VMessConfig) error {
	if !n.protocol.IsVMess() {
		return fmt.Errorf("cannot update vmess config for non-vmess protocol")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.vmessConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateHysteria2Config updates the Hysteria2 configuration
func (n *Node) UpdateHysteria2Config(config *vo.Hysteria2Config) error {
	if !n.protocol.IsHysteria2() {
		return fmt.Errorf("cannot update hysteria2 config for non-hysteria2 protocol")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.hysteria2Config = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateTUICConfig updates the TUIC configuration
func (n *Node) UpdateTUICConfig(config *vo.TUICConfig) error {
	if !n.protocol.IsTUIC() {
		return fmt.Errorf("cannot update tuic config for non-tuic protocol")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.tuicConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateAnyTLSConfig updates the AnyTLS configuration
func (n *Node) UpdateAnyTLSConfig(config *vo.AnyTLSConfig) error {
	if !n.protocol.IsAnyTLS() {
		return fmt.Errorf("cannot update anytls config for non-anytls protocol")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.anytlsConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateMetadata updates the node metadata
func (n *Node) UpdateMetadata(metadata vo.NodeMetadata) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.metadata = metadata
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateName updates the node name
func (n *Node) UpdateName(name string) error {
	if name == "" {
		return fmt.Errorf("node name cannot be empty")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.name == name {
		return nil
	}

	n.name = name
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateSortOrder updates the sort order
func (n *Node) UpdateSortOrder(order int) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.sortOrder = order
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// SetMuteNotification sets the mute notification flag
func (n *Node) SetMuteNotification(mute bool) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.muteNotification = mute
	n.updatedAt = biztime.NowUTC()
	n.version++
}

// UpdateRouteConfig updates the routing configuration
func (n *Node) UpdateRouteConfig(config *vo.RouteConfig) error {
	if config != nil {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid route config: %w", err)
		}
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.routeConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// ClearRouteConfig removes the routing configuration
func (n *Node) ClearRouteConfig() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.routeConfig = nil
	n.updatedAt = biztime.NowUTC()
	n.version++
}

// SetExpiresAt sets the expiration time (nil to clear)
func (n *Node) SetExpiresAt(t *time.Time) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.expiresAt = t
	n.updatedAt = biztime.NowUTC()
	n.version++
}

// SetCostLabel sets the cost label (nil to clear)
func (n *Node) SetCostLabel(label *string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.costLabel = label
	n.updatedAt = biztime.NowUTC()
	n.version++
}

// --- Group and ownership mutations ---

// SetUserID sets the owner user ID
func (n *Node) SetUserID(userID *uint) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.userID = userID
	n.updatedAt = biztime.NowUTC()
	n.version++
}

// SetGroupIDs sets the resource group IDs
func (n *Node) SetGroupIDs(groupIDs []uint) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.groupIDs = groupIDs
	n.updatedAt = biztime.NowUTC()
	n.version++
}

// AddGroupID adds a resource group ID if not already present
func (n *Node) AddGroupID(groupID uint) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	newIDs, added := shared.AddToGroupIDs(n.groupIDs, groupID)
	if added {
		n.groupIDs = newIDs
		n.updatedAt = biztime.NowUTC()
		n.version++
	}
	return added
}

// RemoveGroupID removes a resource group ID
func (n *Node) RemoveGroupID(groupID uint) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	newIDs, removed := shared.RemoveFromGroupIDs(n.groupIDs, groupID)
	if removed {
		n.groupIDs = newIDs
		n.updatedAt = biztime.NowUTC()
		n.version++
	}
	return removed
}

// HasGroupID checks if the node belongs to a specific resource group
func (n *Node) HasGroupID(groupID uint) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return shared.HasGroupID(n.groupIDs, groupID)
}

// IsUserOwned returns true if this node is owned by a user (not admin-created)
func (n *Node) IsUserOwned() bool {
	return n.userID != nil
}

// IsOwnedBy checks if the node is owned by the specified user
func (n *Node) IsOwnedBy(userID uint) bool {
	return n.userID != nil && *n.userID == userID
}
