package forward

import (
	"fmt"
	"net"

	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/shared"
	"github.com/orris-inc/orris/internal/shared/biztime"
)

// SetID sets the forward rule ID (only for persistence layer use).
func (r *ForwardRule) SetID(id uint) error {
	if r.id != 0 {
		return fmt.Errorf("forward rule ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("forward rule ID cannot be zero")
	}
	r.id = id
	return nil
}

// Enable enables the forward rule.
func (r *ForwardRule) Enable() error {
	if r.status.IsEnabled() {
		return nil
	}
	r.status = vo.ForwardStatusEnabled
	r.updatedAt = biztime.NowUTC()
	return nil
}

// Disable disables the forward rule.
func (r *ForwardRule) Disable() error {
	if r.status.IsDisabled() {
		return nil
	}
	r.status = vo.ForwardStatusDisabled
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateName updates the rule name.
func (r *ForwardRule) UpdateName(name string) error {
	if name == "" {
		return fmt.Errorf("forward rule name cannot be empty")
	}
	if r.name == name {
		return nil
	}
	r.name = name
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateListenPort updates the listen port.
func (r *ForwardRule) UpdateListenPort(port uint16) error {
	if port == 0 {
		return fmt.Errorf("listen port cannot be zero")
	}
	if r.listenPort == port {
		return nil
	}
	r.listenPort = port
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateTarget updates the target address and port.
// This will clear the targetNodeID when setting static address.
func (r *ForwardRule) UpdateTarget(address string, port uint16) error {
	if address == "" {
		return fmt.Errorf("target address cannot be empty")
	}
	if port == 0 {
		return fmt.Errorf("target port cannot be zero")
	}
	if err := validateAddress(address); err != nil {
		return fmt.Errorf("invalid target address: %w", err)
	}

	if r.targetAddress == address && r.targetPort == port && r.targetNodeID == nil {
		return nil
	}

	r.targetAddress = address
	r.targetPort = port
	r.targetNodeID = nil // clear targetNodeID when setting static address
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateTargetNodeID updates the target node ID for dynamic address resolution.
// This will clear the targetAddress and targetPort when setting node ID.
func (r *ForwardRule) UpdateTargetNodeID(nodeID *uint) error {
	// Only direct, entry, chain, and direct_chain types support targetNodeID
	if !r.ruleType.IsDirect() && !r.ruleType.IsEntry() && !r.ruleType.IsChain() && !r.ruleType.IsDirectChain() {
		return fmt.Errorf("target node ID can only be set for direct, entry, chain, or direct_chain type rules")
	}

	// If nodeID is nil or 0, clear the targetNodeID
	if nodeID == nil || *nodeID == 0 {
		r.targetNodeID = nil
		r.updatedAt = biztime.NowUTC()
		return nil
	}

	// Check if already set to the same value
	if r.targetNodeID != nil && *r.targetNodeID == *nodeID && r.targetAddress == "" && r.targetPort == 0 {
		return nil
	}

	r.targetNodeID = nodeID
	r.targetAddress = "" // clear static address when setting node ID
	r.targetPort = 0
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateIPVersion updates the IP version preference.
func (r *ForwardRule) UpdateIPVersion(ipVersion vo.IPVersion) error {
	if ipVersion == "" {
		ipVersion = vo.IPVersionAuto
	}
	if !ipVersion.IsValid() {
		return fmt.Errorf("invalid IP version: %s", ipVersion)
	}
	if r.ipVersion == ipVersion {
		return nil
	}
	r.ipVersion = ipVersion
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateProtocol updates the protocol.
func (r *ForwardRule) UpdateProtocol(protocol vo.ForwardProtocol) error {
	if !protocol.IsValid() {
		return fmt.Errorf("invalid protocol: %s", protocol)
	}
	if r.protocol == protocol {
		return nil
	}
	r.protocol = protocol
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateRemark updates the remark.
func (r *ForwardRule) UpdateRemark(remark string) error {
	if r.remark == remark {
		return nil
	}
	r.remark = remark
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateTrafficMultiplier updates the traffic multiplier.
// Set to nil to enable auto-calculation based on node count.
func (r *ForwardRule) UpdateTrafficMultiplier(multiplier *float64) error {
	// Validate if provided
	if multiplier != nil {
		if *multiplier < 0 {
			return fmt.Errorf("traffic multiplier cannot be negative: %f", *multiplier)
		}
		if *multiplier > 1000000 {
			return fmt.Errorf("traffic multiplier exceeds maximum (1000000): %f", *multiplier)
		}
	}

	// Check if already set to the same value
	if r.trafficMultiplier == nil && multiplier == nil {
		return nil
	}
	if r.trafficMultiplier != nil && multiplier != nil && *r.trafficMultiplier == *multiplier {
		return nil
	}

	r.trafficMultiplier = multiplier
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateSortOrder updates the sort order.
// Sort order must be non-negative.
func (r *ForwardRule) UpdateSortOrder(order int) error {
	if order < 0 {
		return fmt.Errorf("sort order must be non-negative, got %d", order)
	}
	r.sortOrder = order
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateBindIP updates the bind IP address for outbound connections.
func (r *ForwardRule) UpdateBindIP(bindIP string) error {
	if r.bindIP == bindIP {
		return nil
	}
	// Validate bindIP if not empty
	if bindIP != "" {
		if ip := net.ParseIP(bindIP); ip == nil {
			return fmt.Errorf("invalid bind IP address: %s", bindIP)
		}
	}
	r.bindIP = bindIP
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateExitAgentID updates the exit agent ID for entry type rules.
func (r *ForwardRule) UpdateExitAgentID(exitAgentID uint) error {
	if !r.ruleType.IsEntry() {
		return fmt.Errorf("exit agent ID can only be updated for entry type rules")
	}
	if exitAgentID == 0 {
		return fmt.Errorf("exit agent ID cannot be zero")
	}
	if exitAgentID == r.agentID {
		return fmt.Errorf("exit agent cannot be the same as entry agent")
	}
	// If exitAgents is set, clear it when switching to single exit agent
	if len(r.exitAgents) > 0 {
		r.exitAgents = nil
	}
	if r.exitAgentID == exitAgentID {
		return nil
	}
	r.exitAgentID = exitAgentID
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateExitAgents updates the exit agents for load balancing.
// This clears the single exitAgentID when setting multiple exit agents.
func (r *ForwardRule) UpdateExitAgents(exitAgents []vo.AgentWeight) error {
	if !r.ruleType.IsEntry() {
		return fmt.Errorf("exit agents can only be updated for entry type rules")
	}
	if len(exitAgents) == 0 {
		return fmt.Errorf("exit agents cannot be empty")
	}
	if err := vo.ValidateAgentWeights(exitAgents); err != nil {
		return fmt.Errorf("invalid exit agents: %w", err)
	}
	// Validate no exit agent is the same as entry agent
	for _, aw := range exitAgents {
		if aw.AgentID() == r.agentID {
			return fmt.Errorf("exit agent cannot be the same as entry agent")
		}
	}
	// Validate weighted strategy requires at least one non-backup agent
	if r.loadBalanceStrategy.IsWeighted() {
		hasNonBackup := false
		for _, aw := range exitAgents {
			if !aw.IsBackup() {
				hasNonBackup = true
				break
			}
		}
		if !hasNonBackup {
			return fmt.Errorf("weighted strategy requires at least one exit agent with non-zero weight")
		}
	}
	// Clear single exitAgentID when switching to multiple exit agents
	r.exitAgentID = 0
	r.exitAgents = exitAgents
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateLoadBalanceStrategy updates the load balance strategy for multi-exit rules.
func (r *ForwardRule) UpdateLoadBalanceStrategy(strategy vo.LoadBalanceStrategy) error {
	if !r.ruleType.IsEntry() {
		return fmt.Errorf("load balance strategy can only be updated for entry type rules")
	}
	if !strategy.IsValid() {
		return fmt.Errorf("invalid load balance strategy: %s", strategy)
	}
	if r.loadBalanceStrategy == strategy {
		return nil
	}
	// Validate weighted strategy requires at least one non-backup agent
	if strategy.IsWeighted() && len(r.exitAgents) > 0 {
		hasNonBackup := false
		for _, aw := range r.exitAgents {
			if !aw.IsBackup() {
				hasNonBackup = true
				break
			}
		}
		if !hasNonBackup {
			return fmt.Errorf("weighted strategy requires at least one exit agent with non-zero weight")
		}
	}
	r.loadBalanceStrategy = strategy
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateTunnelType updates the tunnel type.
func (r *ForwardRule) UpdateTunnelType(tunnelType vo.TunnelType) error {
	if !tunnelType.IsValid() {
		return fmt.Errorf("invalid tunnel type: %s", tunnelType)
	}
	if r.tunnelType == tunnelType {
		return nil
	}
	r.tunnelType = tunnelType
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateTunnelHops updates the number of hops using tunnel for hybrid chain.
// Set to nil or 0 for full tunnel mode (all hops use tunnel).
// Set to N for hybrid mode (first N hops use tunnel, rest use direct).
// Only valid for chain rules.
func (r *ForwardRule) UpdateTunnelHops(tunnelHops *int) error {
	if !r.ruleType.IsChain() {
		return fmt.Errorf("tunnel_hops can only be set for chain type rules")
	}

	// Validate tunnelHops
	if tunnelHops != nil {
		if *tunnelHops < 0 {
			return fmt.Errorf("tunnel_hops cannot be negative")
		}
		// Validate chainPortConfig for direct hops
		totalHops := len(r.chainAgentIDs)
		if *tunnelHops > 0 && *tunnelHops < totalHops {
			// This is a hybrid chain, verify port config for direct hops
			for i := *tunnelHops; i < len(r.chainAgentIDs); i++ {
				chainAgentID := r.chainAgentIDs[i]
				port, exists := r.chainPortConfig[chainAgentID]
				if !exists || port == 0 {
					return fmt.Errorf("chain_port_config missing valid port for agent ID %d (required for direct hop at position %d)", chainAgentID, i+1)
				}
			}
		}
	}

	// Check if already set to the same value
	if r.tunnelHops == nil && tunnelHops == nil {
		return nil
	}
	if r.tunnelHops != nil && tunnelHops != nil && *r.tunnelHops == *tunnelHops {
		return nil
	}

	r.tunnelHops = tunnelHops
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateAgentID updates the entry agent ID.
func (r *ForwardRule) UpdateAgentID(agentID uint) error {
	if agentID == 0 {
		return fmt.Errorf("agent ID cannot be zero")
	}
	if r.agentID == agentID {
		return nil
	}
	// For chain and direct_chain types, ensure the new agent is not in the chain
	if r.ruleType.IsChain() || r.ruleType.IsDirectChain() {
		for _, id := range r.chainAgentIDs {
			if id == agentID {
				return fmt.Errorf("agent ID cannot be the same as a chain agent ID")
			}
		}
	}
	r.agentID = agentID
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateChainAgentIDs updates the chain agent IDs for chain type rules.
func (r *ForwardRule) UpdateChainAgentIDs(chainAgentIDs []uint) error {
	if !r.ruleType.IsChain() && !r.ruleType.IsDirectChain() {
		return fmt.Errorf("chain agent IDs can only be updated for chain or direct_chain type rules")
	}
	if len(chainAgentIDs) == 0 {
		return fmt.Errorf("chain agent IDs cannot be empty for chain forward")
	}
	if len(chainAgentIDs) > 10 {
		return fmt.Errorf("chain forward supports maximum 10 intermediate agents")
	}
	// Check for duplicates (including entry agent)
	seen := make(map[uint]bool)
	seen[r.agentID] = true
	for _, id := range chainAgentIDs {
		if id == 0 {
			return fmt.Errorf("chain agent ID cannot be zero")
		}
		if seen[id] {
			return fmt.Errorf("chain contains duplicate agent ID: %d", id)
		}
		seen[id] = true
	}
	r.chainAgentIDs = chainAgentIDs
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateChainPortConfig updates the chain port configuration.
// Supported for:
// - direct_chain: all chain agents require port configuration
// - chain (hybrid mode): agents from tunnelHops position onward require port configuration
// DEPRECATED: Use UpdateDirectChainConfig instead for atomic updates of direct_chain rules.
func (r *ForwardRule) UpdateChainPortConfig(chainPortConfig map[uint]uint16) error {
	if !r.ruleType.IsDirectChain() && !r.ruleType.IsChain() {
		return fmt.Errorf("chain_port_config can only be updated for chain or direct_chain type rules")
	}

	// Handle different rule types
	if r.ruleType.IsDirectChain() {
		// For direct_chain: all chain agents need port configuration
		if len(chainPortConfig) == 0 {
			return fmt.Errorf("chain_port_config cannot be empty for direct_chain forward")
		}
		// Verify all chain agents have port configuration
		for _, id := range r.chainAgentIDs {
			port, exists := chainPortConfig[id]
			if !exists {
				return fmt.Errorf("chain_port_config missing port for agent ID %d", id)
			}
			if port == 0 {
				return fmt.Errorf("chain_port_config has invalid port for agent ID %d", id)
			}
		}
		// Check for extra entries in chain_port_config
		for id := range chainPortConfig {
			found := false
			for _, chainID := range r.chainAgentIDs {
				if id == chainID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("chain_port_config contains agent ID %d not in chain_agent_ids", id)
			}
		}
	} else if r.ruleType.IsChain() {
		// For chain (hybrid mode): only agents from tunnelHops position need port configuration
		// If tunnelHops is nil or 0, it's full tunnel mode and no port config is needed
		// If tunnelHops >= len(chainAgentIDs), it's also full tunnel mode
		if r.tunnelHops == nil || *r.tunnelHops == 0 || *r.tunnelHops >= len(r.chainAgentIDs) {
			// Full tunnel mode - chainPortConfig is optional but should be empty or not updated
			if len(chainPortConfig) > 0 {
				// Allow clearing or setting for future hybrid mode transition
				// Just validate the IDs are in chain
				for id := range chainPortConfig {
					found := false
					for _, chainID := range r.chainAgentIDs {
						if id == chainID {
							found = true
							break
						}
					}
					if !found {
						return fmt.Errorf("chain_port_config contains agent ID %d not in chain_agent_ids", id)
					}
				}
			}
		} else {
			// Hybrid chain mode - verify port config for direct hops
			for i := *r.tunnelHops; i < len(r.chainAgentIDs); i++ {
				chainAgentID := r.chainAgentIDs[i]
				port, exists := chainPortConfig[chainAgentID]
				if !exists {
					return fmt.Errorf("chain_port_config missing port for agent ID %d (required for direct hop at position %d)", chainAgentID, i+1)
				}
				if port == 0 {
					return fmt.Errorf("chain_port_config has invalid port for agent ID %d", chainAgentID)
				}
			}
			// Check for extra entries not in chain
			for id := range chainPortConfig {
				found := false
				for _, chainID := range r.chainAgentIDs {
					if id == chainID {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("chain_port_config contains agent ID %d not in chain_agent_ids", id)
				}
			}
		}
	}

	r.chainPortConfig = chainPortConfig
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateDirectChainConfig atomically updates both chainAgentIDs and chainPortConfig for direct_chain rules.
// This method ensures consistency between chainAgentIDs and chainPortConfig by validating them together.
func (r *ForwardRule) UpdateDirectChainConfig(chainAgentIDs []uint, chainPortConfig map[uint]uint16) error {
	if !r.ruleType.IsDirectChain() {
		return fmt.Errorf("direct chain config can only be updated for direct_chain type rules")
	}
	if len(chainAgentIDs) == 0 {
		return fmt.Errorf("chain agent IDs cannot be empty for direct_chain forward")
	}
	if len(chainAgentIDs) > 10 {
		return fmt.Errorf("direct_chain forward supports maximum 10 intermediate agents")
	}
	if len(chainPortConfig) == 0 {
		return fmt.Errorf("chain_port_config cannot be empty for direct_chain forward")
	}

	// Check for duplicates in chainAgentIDs (including entry agent)
	seen := make(map[uint]bool)
	seen[r.agentID] = true
	for _, id := range chainAgentIDs {
		if id == 0 {
			return fmt.Errorf("chain agent ID cannot be zero")
		}
		if seen[id] {
			return fmt.Errorf("chain contains duplicate agent ID: %d", id)
		}
		seen[id] = true
	}

	// Verify all chain agents have valid port configuration
	for _, id := range chainAgentIDs {
		port, exists := chainPortConfig[id]
		if !exists {
			return fmt.Errorf("chain_port_config missing port for agent ID %d", id)
		}
		if port == 0 {
			return fmt.Errorf("chain_port_config has invalid port for agent ID %d", id)
		}
	}

	// Check for extra entries in chain_port_config
	for id := range chainPortConfig {
		found := false
		for _, chainID := range chainAgentIDs {
			if id == chainID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("chain_port_config contains agent ID %d not in chain_agent_ids", id)
		}
	}

	// All validations passed, update both fields atomically
	r.chainAgentIDs = chainAgentIDs
	r.chainPortConfig = chainPortConfig
	r.updatedAt = biztime.NowUTC()
	return nil
}

// SetGroupIDs sets the resource group IDs.
func (r *ForwardRule) SetGroupIDs(groupIDs []uint) {
	r.groupIDs = groupIDs
	r.updatedAt = biztime.NowUTC()
}

// AddGroupID adds a resource group ID if not already present.
// Returns true if the group ID was added, false if it already exists.
func (r *ForwardRule) AddGroupID(groupID uint) bool {
	newIDs, added := shared.AddToGroupIDs(r.groupIDs, groupID)
	if added {
		r.groupIDs = newIDs
		r.updatedAt = biztime.NowUTC()
	}
	return added
}

// RemoveGroupID removes a resource group ID.
// Returns true if the group ID was removed, false if not found.
func (r *ForwardRule) RemoveGroupID(groupID uint) bool {
	newIDs, removed := shared.RemoveFromGroupIDs(r.groupIDs, groupID)
	if removed {
		r.groupIDs = newIDs
		r.updatedAt = biztime.NowUTC()
	}
	return removed
}

// HasGroupID checks if the rule belongs to a specific resource group.
func (r *ForwardRule) HasGroupID(groupID uint) bool {
	return shared.HasGroupID(r.groupIDs, groupID)
}

// UpdateServerAddress updates the server address for external rules.
func (r *ForwardRule) UpdateServerAddress(serverAddress string) error {
	if !r.ruleType.IsExternal() {
		return fmt.Errorf("server address can only be updated for external type rules")
	}
	if serverAddress == "" {
		return fmt.Errorf("server address cannot be empty for external rules")
	}
	if r.serverAddress == serverAddress {
		return nil
	}
	r.serverAddress = serverAddress
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateExternalSource updates the external source identifier.
func (r *ForwardRule) UpdateExternalSource(externalSource string) error {
	if !r.ruleType.IsExternal() {
		return fmt.Errorf("external source can only be updated for external type rules")
	}
	if externalSource == "" {
		return fmt.Errorf("external source cannot be empty for external rules")
	}
	if r.externalSource == externalSource {
		return nil
	}
	r.externalSource = externalSource
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateExternalRuleID updates the external rule ID reference.
func (r *ForwardRule) UpdateExternalRuleID(externalRuleID string) error {
	if !r.ruleType.IsExternal() {
		return fmt.Errorf("external rule ID can only be updated for external type rules")
	}
	if r.externalRuleID == externalRuleID {
		return nil
	}
	r.externalRuleID = externalRuleID
	r.updatedAt = biztime.NowUTC()
	return nil
}
