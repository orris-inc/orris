package forward

import (
	"fmt"

	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
)

// Validate performs domain-level validation on a fully constructed ForwardRule.
// This is the single source of truth for all validation rules.
// Both NewForwardRule (after construction) and ReconstructForwardRule call this method.
func (r *ForwardRule) Validate() error {
	// Agent ID is required for non-external rules
	if r.ruleType.RequiresAgent() && r.agentID == 0 {
		return fmt.Errorf("agent ID is required")
	}
	if !r.ruleType.IsValid() {
		return fmt.Errorf("invalid rule type: %s", r.ruleType)
	}
	if r.name == "" {
		return fmt.Errorf("forward rule name is required")
	}
	if !r.protocol.IsValid() {
		return fmt.Errorf("invalid protocol: %s", r.protocol)
	}
	if !r.status.IsValid() {
		return fmt.Errorf("invalid status: %s", r.status)
	}

	// Validate traffic multiplier
	if r.trafficMultiplier != nil {
		if *r.trafficMultiplier < 0 {
			return fmt.Errorf("traffic multiplier cannot be negative: %f", *r.trafficMultiplier)
		}
		if *r.trafficMultiplier > 1000000 {
			return fmt.Errorf("traffic multiplier exceeds maximum (1000000): %f", *r.trafficMultiplier)
		}
	}

	// Validate IP version
	if !r.ipVersion.IsValid() {
		return fmt.Errorf("invalid IP version: %s", r.ipVersion)
	}

	// Validate tunnel type
	if !r.tunnelType.IsValid() {
		return fmt.Errorf("invalid tunnel type: %s", r.tunnelType)
	}

	// Validate load balance strategy
	if !r.loadBalanceStrategy.IsValid() {
		return fmt.Errorf("invalid load balance strategy: %s", r.loadBalanceStrategy)
	}

	// Validate required fields based on rule type
	switch r.ruleType {
	case vo.ForwardRuleTypeDirect:
		if r.listenPort == 0 {
			return fmt.Errorf("listen port is required for direct forward")
		}
		// Either targetAddress+targetPort OR targetNodeID must be set
		hasTarget := r.targetAddress != "" && r.targetPort != 0
		hasTargetNode := r.targetNodeID != nil && *r.targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return fmt.Errorf("either target address+port or target node ID is required for direct forward")
		}
		if hasTarget && hasTargetNode {
			return fmt.Errorf("target address+port and target node ID are mutually exclusive for direct forward")
		}
		if hasTarget {
			if err := validateAddress(r.targetAddress); err != nil {
				return fmt.Errorf("invalid target address: %w", err)
			}
		}
	case vo.ForwardRuleTypeEntry:
		if r.listenPort == 0 {
			return fmt.Errorf("listen port is required for entry forward")
		}
		// Validate exit agent configuration: either exitAgentID OR exitAgents, not both
		hasExitAgent := r.exitAgentID != 0
		hasExitAgents := len(r.exitAgents) > 0
		if !hasExitAgent && !hasExitAgents {
			return fmt.Errorf("either exit agent ID or exit agents is required for entry forward")
		}
		if hasExitAgent && hasExitAgents {
			return fmt.Errorf("exit agent ID and exit agents are mutually exclusive for entry forward")
		}
		// Validate single exit agent is not the same as entry agent
		if hasExitAgent && r.exitAgentID == r.agentID {
			return fmt.Errorf("exit agent cannot be the same as entry agent")
		}
		// Validate exitAgents if provided
		if hasExitAgents {
			if err := vo.ValidateAgentWeights(r.exitAgents); err != nil {
				return fmt.Errorf("invalid exit agents: %w", err)
			}
			// Validate no exit agent is the same as entry agent
			for _, aw := range r.exitAgents {
				if aw.AgentID() == r.agentID {
					return fmt.Errorf("exit agent cannot be the same as entry agent")
				}
			}
			// Validate weighted strategy requires at least one non-backup agent
			if r.loadBalanceStrategy.IsWeighted() {
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
		}
		// Entry rules require target information (to be passed to exit agent)
		hasTarget := r.targetAddress != "" && r.targetPort != 0
		hasTargetNode := r.targetNodeID != nil && *r.targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return fmt.Errorf("either target address+port or target node ID is required for entry forward")
		}
		if hasTarget && hasTargetNode {
			return fmt.Errorf("target address+port and target node ID are mutually exclusive for entry forward")
		}
		if hasTarget {
			if err := validateAddress(r.targetAddress); err != nil {
				return fmt.Errorf("invalid target address: %w", err)
			}
		}
	case vo.ForwardRuleTypeChain:
		if r.listenPort == 0 {
			return fmt.Errorf("listen port is required for chain forward")
		}
		if len(r.chainAgentIDs) == 0 {
			return fmt.Errorf("chain agent IDs is required for chain forward (at least 1 intermediate agent)")
		}
		if len(r.chainAgentIDs) > 10 {
			return fmt.Errorf("chain forward supports maximum 10 intermediate agents")
		}
		// Check for duplicates in chain (including entry agent)
		seen := make(map[uint]bool)
		seen[r.agentID] = true
		for _, id := range r.chainAgentIDs {
			if id == 0 {
				return fmt.Errorf("chain agent ID cannot be zero")
			}
			if seen[id] {
				return fmt.Errorf("chain contains duplicate agent ID: %d", id)
			}
			seen[id] = true
		}
		// Validate tunnelHops for hybrid chain
		if r.tunnelHops != nil {
			if *r.tunnelHops < 0 {
				return fmt.Errorf("tunnel_hops cannot be negative")
			}
			totalHops := len(r.chainAgentIDs)
			if *r.tunnelHops > 0 && *r.tunnelHops < totalHops {
				// This is a hybrid chain, verify port config for direct hops
				for i := *r.tunnelHops; i < len(r.chainAgentIDs); i++ {
					chainAgentID := r.chainAgentIDs[i]
					port, exists := r.chainPortConfig[chainAgentID]
					if !exists || port == 0 {
						return fmt.Errorf("chain_port_config missing valid port for agent ID %d (required for direct hop at position %d)", chainAgentID, i+1)
					}
				}
			}
		}
		// Chain rules require target information (at the end of chain)
		hasTarget := r.targetAddress != "" && r.targetPort != 0
		hasTargetNode := r.targetNodeID != nil && *r.targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return fmt.Errorf("either target address+port or target node ID is required for chain forward")
		}
		if hasTarget && hasTargetNode {
			return fmt.Errorf("target address+port and target node ID are mutually exclusive for chain forward")
		}
		if hasTarget {
			if err := validateAddress(r.targetAddress); err != nil {
				return fmt.Errorf("invalid target address: %w", err)
			}
		}
	case vo.ForwardRuleTypeDirectChain:
		if r.listenPort == 0 {
			return fmt.Errorf("listen port is required for direct_chain forward")
		}
		if len(r.chainAgentIDs) == 0 {
			return fmt.Errorf("chain agent IDs is required for direct_chain forward (at least 1 intermediate agent)")
		}
		if len(r.chainAgentIDs) > 10 {
			return fmt.Errorf("direct_chain forward supports maximum 10 intermediate agents")
		}
		// Check for duplicates in chain (including entry agent)
		seen := make(map[uint]bool)
		seen[r.agentID] = true
		for _, id := range r.chainAgentIDs {
			if id == 0 {
				return fmt.Errorf("chain agent ID cannot be zero")
			}
			if seen[id] {
				return fmt.Errorf("chain contains duplicate agent ID: %d", id)
			}
			seen[id] = true
		}
		// Validate chain_port_config
		if len(r.chainPortConfig) == 0 {
			return fmt.Errorf("chain_port_config is required for direct_chain forward")
		}
		// Verify all chain agents have port configuration
		for _, id := range r.chainAgentIDs {
			port, exists := r.chainPortConfig[id]
			if !exists {
				return fmt.Errorf("chain_port_config missing port for agent ID %d", id)
			}
			if port == 0 {
				return fmt.Errorf("chain_port_config has invalid port for agent ID %d", id)
			}
		}
		// Check for extra entries in chain_port_config
		for id := range r.chainPortConfig {
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
		// Direct chain rules require target information (at the end of chain)
		hasTarget := r.targetAddress != "" && r.targetPort != 0
		hasTargetNode := r.targetNodeID != nil && *r.targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return fmt.Errorf("either target address+port or target node ID is required for direct_chain forward")
		}
		if hasTarget && hasTargetNode {
			return fmt.Errorf("target address+port and target node ID are mutually exclusive for direct_chain forward")
		}
		if hasTarget {
			if err := validateAddress(r.targetAddress); err != nil {
				return fmt.Errorf("invalid target address: %w", err)
			}
		}
	case vo.ForwardRuleTypeExternal:
		if r.listenPort == 0 {
			return fmt.Errorf("listen port is required for external forward")
		}
		if r.serverAddress == "" {
			return fmt.Errorf("server address is required for external forward")
		}
		if r.targetNodeID == nil || *r.targetNodeID == 0 {
			return fmt.Errorf("target node ID is required for external forward (protocol is derived from target node)")
		}
		// externalSource is optional
	}

	return nil
}
