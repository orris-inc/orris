package forward

import (
	"fmt"

	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
)

// ExitAgentID returns the exit agent ID (for entry type rules).
func (r *ForwardRule) ExitAgentID() uint {
	return r.exitAgentID
}

// ExitAgents returns the exit agents with weights for load balancing.
func (r *ForwardRule) ExitAgents() []vo.AgentWeight {
	return r.exitAgents
}

// HasMultipleExitAgents returns true if the rule has multiple exit agents configured.
func (r *ForwardRule) HasMultipleExitAgents() bool {
	return len(r.exitAgents) > 0
}

// LoadBalanceStrategy returns the load balance strategy for multi-exit rules.
func (r *ForwardRule) LoadBalanceStrategy() vo.LoadBalanceStrategy {
	return r.loadBalanceStrategy
}

// GetAllExitAgentIDs returns all exit agent IDs (single exitAgentID or all from exitAgents).
func (r *ForwardRule) GetAllExitAgentIDs() []uint {
	if len(r.exitAgents) > 0 {
		return vo.GetAgentIDs(r.exitAgents)
	}
	if r.exitAgentID != 0 {
		return []uint{r.exitAgentID}
	}
	return nil
}

// ChainAgentIDs returns the chain agent IDs (for chain type rules).
func (r *ForwardRule) ChainAgentIDs() []uint {
	return r.chainAgentIDs
}

// ChainPortConfig returns the chain port configuration (for direct_chain type rules).
func (r *ForwardRule) ChainPortConfig() map[uint]uint16 {
	return r.chainPortConfig
}

// TunnelType returns the tunnel type (ws or tls).
func (r *ForwardRule) TunnelType() vo.TunnelType {
	return r.tunnelType
}

// TunnelHops returns the number of hops using tunnel.
// Returns nil for full tunnel mode (all hops use tunnel).
// Returns N for hybrid mode (first N hops use tunnel, rest use direct).
func (r *ForwardRule) TunnelHops() *int {
	return r.tunnelHops
}

// IsHybridChain returns true if this is a hybrid chain rule
// (chain rule with tunnelHops > 0 and tunnelHops < total hops).
func (r *ForwardRule) IsHybridChain() bool {
	if !r.ruleType.IsChain() {
		return false
	}
	if r.tunnelHops == nil || *r.tunnelHops == 0 {
		return false
	}
	totalHops := len(r.chainAgentIDs)
	return *r.tunnelHops > 0 && *r.tunnelHops < totalHops
}

// NeedsTunnelAtPosition returns true if the hop at the given position needs tunnel.
// Position 0 is entry agent, position N is chainAgentIDs[N-1].
// For chain rules: all positions need tunnel unless tunnelHops is set.
// For hybrid chain: positions < tunnelHops need tunnel for outbound.
// For direct_chain: no positions need tunnel.
func (r *ForwardRule) NeedsTunnelAtPosition(position int) bool {
	if r.ruleType.IsDirectChain() {
		return false
	}
	if !r.ruleType.IsChain() {
		return false
	}
	// Full tunnel mode (nil, 0, or >= total hops means all hops use tunnel)
	if r.tunnelHops == nil || *r.tunnelHops == 0 || *r.tunnelHops >= len(r.chainAgentIDs) {
		return true
	}
	// Hybrid mode: positions < tunnelHops use tunnel for outbound connection
	return position < *r.tunnelHops
}

// GetHopMode returns the hop mode for an agent at the given position.
// Returns: "tunnel" (pure tunnel), "direct" (pure direct), or "boundary" (tunnel inbound, direct outbound).
func (r *ForwardRule) GetHopMode(position int) string {
	if r.ruleType.IsDirectChain() {
		return "direct"
	}
	if !r.ruleType.IsChain() {
		return "direct"
	}

	totalPositions := len(r.chainAgentIDs) + 1 // +1 for entry agent
	isLast := position >= totalPositions-1

	// Full tunnel mode (nil, 0, or >= total hops means all hops use tunnel)
	if r.tunnelHops == nil || *r.tunnelHops == 0 || *r.tunnelHops >= len(r.chainAgentIDs) {
		return "tunnel"
	}

	// Hybrid mode
	inboundNeedsTunnel := position > 0 && position <= *r.tunnelHops
	outboundNeedsTunnel := !isLast && position < *r.tunnelHops

	if inboundNeedsTunnel && !outboundNeedsTunnel {
		return "boundary"
	}
	if outboundNeedsTunnel {
		return "tunnel"
	}
	return "direct"
}

// GetAgentListenPort returns the listen port for a specific agent in the chain.
// Returns 0 if the agent is not found in chain_port_config.
func (r *ForwardRule) GetAgentListenPort(agentID uint) uint16 {
	if r.chainPortConfig == nil {
		return 0
	}
	return r.chainPortConfig[agentID]
}

// GetNextHopForDirectChain returns the next hop agent ID and port for a given agent in the direct_chain.
// Returns (0, 0) if the agent is the last in chain or not part of the chain.
// WARNING: This method returns 0 for both "last in chain" and "missing port config" cases.
// DEPRECATED: Use GetNextHopForDirectChainSafe for better error handling.
func (r *ForwardRule) GetNextHopForDirectChain(currentAgentID uint) (nextAgentID uint, nextPort uint16) {
	nextID, port, _ := r.GetNextHopForDirectChainSafe(currentAgentID)
	return nextID, port
}

// GetNextHopForDirectChainSafe returns the next hop agent ID and port for a given agent in the direct_chain.
// Returns error if:
// - Not a direct_chain rule
// - Agent not found in chain
// - Next hop port configuration is missing or invalid
// Returns (0, 0, nil) if the agent is the last in the chain.
func (r *ForwardRule) GetNextHopForDirectChainSafe(currentAgentID uint) (nextAgentID uint, nextPort uint16, err error) {
	if !r.ruleType.IsDirectChain() {
		return 0, 0, fmt.Errorf("not a direct_chain rule")
	}

	// Build full chain: agentID -> chainAgentIDs[0] -> chainAgentIDs[1] -> ...
	fullChain := append([]uint{r.agentID}, r.chainAgentIDs...)

	for i, id := range fullChain {
		if id == currentAgentID {
			// Found the agent in chain
			if i >= len(fullChain)-1 {
				// This is the last agent in chain
				return 0, 0, nil
			}

			// Get next hop
			nextID := fullChain[i+1]
			nextPort := r.GetAgentListenPort(nextID)

			// Validate port configuration
			if nextPort == 0 {
				return 0, 0, fmt.Errorf("missing or invalid port configuration for next hop agent %d", nextID)
			}

			return nextID, nextPort, nil
		}
	}

	// Agent not found in chain
	return 0, 0, fmt.Errorf("agent %d not found in direct_chain", currentAgentID)
}

// GetNextHopAgentID returns the next hop agent ID for a given agent in the chain.
// Returns 0 if the agent is the last in chain or not part of the chain.
func (r *ForwardRule) GetNextHopAgentID(currentAgentID uint) uint {
	if !r.ruleType.IsChain() {
		return 0
	}

	// Build full chain: agentID -> chainAgentIDs[0] -> chainAgentIDs[1] -> ...
	fullChain := append([]uint{r.agentID}, r.chainAgentIDs...)

	for i, id := range fullChain {
		if id == currentAgentID && i < len(fullChain)-1 {
			return fullChain[i+1]
		}
	}
	return 0 // Last agent in chain or not found
}

// IsLastInChain checks if the given agent is the last in the forwarding chain.
// Works for both chain and direct_chain rule types.
func (r *ForwardRule) IsLastInChain(agentID uint) bool {
	if !r.ruleType.IsChain() && !r.ruleType.IsDirectChain() {
		return false
	}
	if len(r.chainAgentIDs) == 0 {
		return agentID == r.agentID
	}
	return agentID == r.chainAgentIDs[len(r.chainAgentIDs)-1]
}

// GetChainPosition returns the position (0-indexed) of the agent in the chain.
// Returns -1 if not in chain. Works for both chain and direct_chain rule types.
func (r *ForwardRule) GetChainPosition(agentID uint) int {
	if !r.ruleType.IsChain() && !r.ruleType.IsDirectChain() {
		return -1
	}

	fullChain := append([]uint{r.agentID}, r.chainAgentIDs...)
	for i, id := range fullChain {
		if id == agentID {
			return i
		}
	}
	return -1
}
