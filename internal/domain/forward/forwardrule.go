// Package forward provides domain models and business logic for TCP/UDP port forwarding.
package forward

import (
	"fmt"
	"net"
	"strconv"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
)

// ForwardRule represents the forward rule aggregate root.
type ForwardRule struct {
	id                uint
	sid               string // Stripe-style prefixed ID (fr_xxx)
	agentID           uint
	userID            *uint // user ID for user-owned rules (nil for admin-created rules)
	subscriptionID    *uint // subscription ID for subscription-bound rules (nil for admin-created rules)
	ruleType          vo.ForwardRuleType
	exitAgentID       uint             // exit agent ID (required for entry type, mutually exclusive with exitAgents)
	exitAgents        []vo.AgentWeight // multiple exit agents with weights for load balancing (mutually exclusive with exitAgentID)
	chainAgentIDs     []uint           // ordered array of intermediate agent IDs for chain forwarding
	chainPortConfig   map[uint]uint16  // map of agent_id -> listen_port for direct_chain type or hybrid chain direct hops
	tunnelHops        *int             // number of hops using tunnel (nil=full tunnel, N=first N hops use tunnel)
	tunnelType        vo.TunnelType    // tunnel type: ws or tls (default: ws)
	name              string
	listenPort        uint16
	targetAddress     string // final target address (required for direct and exit types if targetNodeID is not set)
	targetPort        uint16 // final target port (required for direct and exit types if targetNodeID is not set)
	targetNodeID      *uint  // target node ID for dynamic address resolution (mutually exclusive with targetAddress/targetPort)
	bindIP            string // bind IP address for outbound connections (optional)
	ipVersion         vo.IPVersion
	protocol          vo.ForwardProtocol
	status            vo.ForwardStatus
	remark            string
	uploadBytes       int64
	downloadBytes     int64
	trafficMultiplier *float64 // traffic multiplier for display. nil means auto-calculate based on node count
	sortOrder         int
	groupIDs          []uint // resource group IDs for access control
	// External rule fields (used when ruleType = external)
	serverAddress  string // server address for external rules (replaces agent's public address)
	externalSource string // external source identifier (required for external rules)
	externalRuleID string // external rule ID for reference (optional)
	createdAt      time.Time
	updatedAt      time.Time
}

// NewForwardRule creates a new forward rule aggregate.
// Parameters depend on ruleType:
// - direct: requires agentID, listenPort, (targetAddress+targetPort OR targetNodeID)
// - entry: requires agentID, listenPort, (exitAgentID OR exitAgents), (targetAddress+targetPort OR targetNodeID)
// - chain: requires agentID, listenPort, chainAgentIDs (at least 1), (targetAddress+targetPort OR targetNodeID)
//   - optionally tunnelHops to create hybrid chain (first N hops tunnel, rest direct)
//   - if tunnelHops > 0, chainPortConfig required for direct hops
//   - optionally exitAgents for last hop load balancing
//
// - direct_chain: requires agentID, listenPort, chainAgentIDs (at least 1), chainPortConfig, (targetAddress+targetPort OR targetNodeID)
func NewForwardRule(
	agentID uint,
	userID *uint,
	subscriptionID *uint,
	ruleType vo.ForwardRuleType,
	exitAgentID uint,
	exitAgents []vo.AgentWeight,
	chainAgentIDs []uint,
	chainPortConfig map[uint]uint16,
	tunnelHops *int,
	tunnelType vo.TunnelType,
	name string,
	listenPort uint16,
	targetAddress string,
	targetPort uint16,
	targetNodeID *uint,
	bindIP string,
	ipVersion vo.IPVersion,
	protocol vo.ForwardProtocol,
	remark string,
	trafficMultiplier *float64,
	sortOrder int,
	shortIDGenerator func() (string, error),
) (*ForwardRule, error) {
	// Agent ID is required for non-external rules
	if ruleType.RequiresAgent() && agentID == 0 {
		return nil, fmt.Errorf("agent ID is required")
	}
	if !ruleType.IsValid() {
		return nil, fmt.Errorf("invalid rule type: %s", ruleType)
	}
	if name == "" {
		return nil, fmt.Errorf("forward rule name is required")
	}
	if !protocol.IsValid() {
		return nil, fmt.Errorf("invalid protocol: %s", protocol)
	}

	// Validate traffic multiplier
	if trafficMultiplier != nil {
		if *trafficMultiplier < 0 {
			return nil, fmt.Errorf("traffic multiplier cannot be negative: %f", *trafficMultiplier)
		}
		if *trafficMultiplier > 1000000 {
			return nil, fmt.Errorf("traffic multiplier exceeds maximum (1000000): %f", *trafficMultiplier)
		}
	}

	// Validate required fields based on rule type
	switch ruleType {
	case vo.ForwardRuleTypeDirect:
		if listenPort == 0 {
			return nil, fmt.Errorf("listen port is required for direct forward")
		}
		// Either targetAddress+targetPort OR targetNodeID must be set
		hasTarget := targetAddress != "" && targetPort != 0
		hasTargetNode := targetNodeID != nil && *targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return nil, fmt.Errorf("either target address+port or target node ID is required for direct forward")
		}
		if hasTarget && hasTargetNode {
			return nil, fmt.Errorf("target address+port and target node ID are mutually exclusive for direct forward")
		}
		if hasTarget {
			if err := validateAddress(targetAddress); err != nil {
				return nil, fmt.Errorf("invalid target address: %w", err)
			}
		}
	case vo.ForwardRuleTypeEntry:
		if listenPort == 0 {
			return nil, fmt.Errorf("listen port is required for entry forward")
		}
		// Validate exit agent configuration: either exitAgentID OR exitAgents, not both
		hasExitAgent := exitAgentID != 0
		hasExitAgents := len(exitAgents) > 0
		if !hasExitAgent && !hasExitAgents {
			return nil, fmt.Errorf("either exit agent ID or exit agents is required for entry forward")
		}
		if hasExitAgent && hasExitAgents {
			return nil, fmt.Errorf("exit agent ID and exit agents are mutually exclusive for entry forward")
		}
		// Validate single exit agent is not the same as entry agent
		if hasExitAgent && exitAgentID == agentID {
			return nil, fmt.Errorf("exit agent cannot be the same as entry agent")
		}
		// Validate exitAgents if provided
		if hasExitAgents {
			if err := vo.ValidateAgentWeights(exitAgents); err != nil {
				return nil, fmt.Errorf("invalid exit agents: %w", err)
			}
			// Validate no exit agent is the same as entry agent
			for _, aw := range exitAgents {
				if aw.AgentID() == agentID {
					return nil, fmt.Errorf("exit agent cannot be the same as entry agent")
				}
			}
		}
		// Entry rules now also require target information (to be passed to exit agent)
		hasTarget := targetAddress != "" && targetPort != 0
		hasTargetNode := targetNodeID != nil && *targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return nil, fmt.Errorf("either target address+port or target node ID is required for entry forward")
		}
		if hasTarget && hasTargetNode {
			return nil, fmt.Errorf("target address+port and target node ID are mutually exclusive for entry forward")
		}
		if hasTarget {
			if err := validateAddress(targetAddress); err != nil {
				return nil, fmt.Errorf("invalid target address: %w", err)
			}
		}
	case vo.ForwardRuleTypeChain:
		if listenPort == 0 {
			return nil, fmt.Errorf("listen port is required for chain forward")
		}
		if len(chainAgentIDs) == 0 {
			return nil, fmt.Errorf("chain agent IDs is required for chain forward (at least 1 intermediate agent)")
		}
		if len(chainAgentIDs) > 10 {
			return nil, fmt.Errorf("chain forward supports maximum 10 intermediate agents")
		}
		// Check for duplicates in chain (including entry agent)
		seen := make(map[uint]bool)
		seen[agentID] = true
		for _, id := range chainAgentIDs {
			if id == 0 {
				return nil, fmt.Errorf("chain agent ID cannot be zero")
			}
			if seen[id] {
				return nil, fmt.Errorf("chain contains duplicate agent ID: %d", id)
			}
			seen[id] = true
		}
		// Validate tunnelHops for hybrid chain
		if tunnelHops != nil {
			if *tunnelHops < 0 {
				return nil, fmt.Errorf("tunnel_hops cannot be negative")
			}
			// If tunnelHops > 0, validate chainPortConfig for direct hops
			// Total hops = len(chainAgentIDs) (entry -> chain[0] -> chain[1] -> ... -> chain[n-1] -> target)
			totalHops := len(chainAgentIDs)
			if *tunnelHops > 0 && *tunnelHops < totalHops {
				// This is a hybrid chain, need port config for direct hops
				// Direct hops start from position tunnelHops
				// Full chain: [agentID, chainAgentIDs[0], chainAgentIDs[1], ...]
				// Position tunnelHops is the boundary node (receives tunnel, sends direct)
				// Positions > tunnelHops are pure direct nodes
				for i := *tunnelHops; i < len(chainAgentIDs); i++ {
					chainAgentID := chainAgentIDs[i]
					port, exists := chainPortConfig[chainAgentID]
					if !exists {
						return nil, fmt.Errorf("chain_port_config missing port for agent ID %d (required for direct hop at position %d)", chainAgentID, i+1)
					}
					if port == 0 {
						return nil, fmt.Errorf("chain_port_config has invalid port for agent ID %d", chainAgentID)
					}
				}
			}
		}
		// Chain rules require target information (at the end of chain)
		hasTarget := targetAddress != "" && targetPort != 0
		hasTargetNode := targetNodeID != nil && *targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return nil, fmt.Errorf("either target address+port or target node ID is required for chain forward")
		}
		if hasTarget && hasTargetNode {
			return nil, fmt.Errorf("target address+port and target node ID are mutually exclusive for chain forward")
		}
		if hasTarget {
			if err := validateAddress(targetAddress); err != nil {
				return nil, fmt.Errorf("invalid target address: %w", err)
			}
		}
	case vo.ForwardRuleTypeDirectChain:
		if listenPort == 0 {
			return nil, fmt.Errorf("listen port is required for direct_chain forward")
		}
		if len(chainAgentIDs) == 0 {
			return nil, fmt.Errorf("chain agent IDs is required for direct_chain forward (at least 1 intermediate agent)")
		}
		if len(chainAgentIDs) > 10 {
			return nil, fmt.Errorf("direct_chain forward supports maximum 10 intermediate agents")
		}
		// Check for duplicates in chain (including entry agent)
		seen := make(map[uint]bool)
		seen[agentID] = true
		for _, id := range chainAgentIDs {
			if id == 0 {
				return nil, fmt.Errorf("chain agent ID cannot be zero")
			}
			if seen[id] {
				return nil, fmt.Errorf("chain contains duplicate agent ID: %d", id)
			}
			seen[id] = true
		}
		// Validate chain_port_config
		if len(chainPortConfig) == 0 {
			return nil, fmt.Errorf("chain_port_config is required for direct_chain forward")
		}
		// Verify all chain agents have port configuration
		for _, id := range chainAgentIDs {
			port, exists := chainPortConfig[id]
			if !exists {
				return nil, fmt.Errorf("chain_port_config missing port for agent ID %d", id)
			}
			if port == 0 {
				return nil, fmt.Errorf("chain_port_config has invalid port for agent ID %d", id)
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
				return nil, fmt.Errorf("chain_port_config contains agent ID %d not in chain_agent_ids", id)
			}
		}
		// Direct chain rules require target information (at the end of chain)
		hasTarget := targetAddress != "" && targetPort != 0
		hasTargetNode := targetNodeID != nil && *targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return nil, fmt.Errorf("either target address+port or target node ID is required for direct_chain forward")
		}
		if hasTarget && hasTargetNode {
			return nil, fmt.Errorf("target address+port and target node ID are mutually exclusive for direct_chain forward")
		}
		if hasTarget {
			if err := validateAddress(targetAddress); err != nil {
				return nil, fmt.Errorf("invalid target address: %w", err)
			}
		}
	case vo.ForwardRuleTypeExternal:
		// External rules don't support NewForwardRule, use NewExternalForwardRule instead
		return nil, fmt.Errorf("use NewExternalForwardRule to create external forward rules")
	}

	// Default ipVersion to auto if not set
	if ipVersion == "" {
		ipVersion = vo.IPVersionAuto
	}
	if !ipVersion.IsValid() {
		return nil, fmt.Errorf("invalid IP version: %s", ipVersion)
	}

	// Validate tunnel type (empty defaults to WS)
	if !tunnelType.IsValid() {
		return nil, fmt.Errorf("invalid tunnel type: %s", tunnelType)
	}

	// Generate SID for external API use
	sid, err := shortIDGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	now := biztime.NowUTC()
	return &ForwardRule{
		sid:               sid,
		agentID:           agentID,
		userID:            userID,
		subscriptionID:    subscriptionID,
		ruleType:          ruleType,
		exitAgentID:       exitAgentID,
		exitAgents:        exitAgents,
		chainAgentIDs:     chainAgentIDs,
		chainPortConfig:   chainPortConfig,
		tunnelHops:        tunnelHops,
		tunnelType:        tunnelType,
		name:              name,
		listenPort:        listenPort,
		targetAddress:     targetAddress,
		targetPort:        targetPort,
		targetNodeID:      targetNodeID,
		bindIP:            bindIP,
		ipVersion:         ipVersion,
		protocol:          protocol,
		status:            vo.ForwardStatusDisabled,
		remark:            remark,
		uploadBytes:       0,
		downloadBytes:     0,
		trafficMultiplier: trafficMultiplier,
		sortOrder:         sortOrder,
		createdAt:         now,
		updatedAt:         now,
	}, nil
}

// NewExternalForwardRule creates a new external forward rule aggregate.
// External rules are for third-party forward services that don't use agents.
// Parameters:
//   - userID: optional user ID (nil for admin-created rules distributed via resource groups)
//   - subscriptionID: optional subscription ID (nil for admin-created rules)
//   - targetNodeID: required for protocol information (protocol is derived from target node)
//   - name: rule name
//   - serverAddress: required server address for subscription delivery
//   - listenPort: listen port
//   - externalSource: optional source identifier
//   - externalRuleID: optional external reference ID
//   - remark: optional description
//   - sortOrder: display sort order
//   - groupIDs: optional resource group IDs for distribution
//   - shortIDGenerator: function to generate SID
func NewExternalForwardRule(
	userID *uint,
	subscriptionID *uint,
	targetNodeID *uint,
	name string,
	serverAddress string,
	listenPort uint16,
	externalSource string,
	externalRuleID string,
	remark string,
	sortOrder int,
	groupIDs []uint,
	shortIDGenerator func() (string, error),
) (*ForwardRule, error) {
	if name == "" {
		return nil, fmt.Errorf("external forward rule name is required")
	}
	if serverAddress == "" {
		return nil, fmt.Errorf("server address is required for external forward")
	}
	if listenPort == 0 {
		return nil, fmt.Errorf("listen port is required for external forward")
	}
	if targetNodeID == nil || *targetNodeID == 0 {
		return nil, fmt.Errorf("target node ID is required for external forward (protocol is derived from target node)")
	}
	// externalSource is optional

	// Generate SID for external API use
	sid, err := shortIDGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	now := biztime.NowUTC()
	return &ForwardRule{
		sid:            sid,
		agentID:        0, // External rules don't have agents
		userID:         userID,
		subscriptionID: subscriptionID,
		ruleType:       vo.ForwardRuleTypeExternal,
		name:           name,
		listenPort:     listenPort,
		targetNodeID:   targetNodeID,
		protocol:       vo.ForwardProtocolTCP, // Default, will be determined from targetNodeID
		status:         vo.ForwardStatusEnabled,
		remark:         remark,
		sortOrder:      sortOrder,
		groupIDs:       groupIDs,
		serverAddress:  serverAddress,
		externalSource: externalSource,
		externalRuleID: externalRuleID,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// ReconstructForwardRule reconstructs a forward rule from persistence.
// It performs full validation to ensure data integrity, even for persisted data.
func ReconstructForwardRule(
	id uint,
	sid string,
	agentID uint,
	userID *uint,
	subscriptionID *uint,
	ruleType vo.ForwardRuleType,
	exitAgentID uint,
	exitAgents []vo.AgentWeight,
	chainAgentIDs []uint,
	chainPortConfig map[uint]uint16,
	tunnelHops *int,
	tunnelType vo.TunnelType,
	name string,
	listenPort uint16,
	targetAddress string,
	targetPort uint16,
	targetNodeID *uint,
	bindIP string,
	ipVersion vo.IPVersion,
	protocol vo.ForwardProtocol,
	status vo.ForwardStatus,
	remark string,
	uploadBytes int64,
	downloadBytes int64,
	trafficMultiplier *float64,
	sortOrder int,
	groupIDs []uint,
	serverAddress string,
	externalSource string,
	externalRuleID string,
	createdAt, updatedAt time.Time,
) (*ForwardRule, error) {
	if id == 0 {
		return nil, fmt.Errorf("forward rule ID cannot be zero")
	}
	if sid == "" {
		return nil, fmt.Errorf("forward rule SID is required")
	}
	// Agent ID is required for non-external rules
	if ruleType.RequiresAgent() && agentID == 0 {
		return nil, fmt.Errorf("agent ID is required")
	}
	if !ruleType.IsValid() {
		return nil, fmt.Errorf("invalid rule type: %s", ruleType)
	}
	if name == "" {
		return nil, fmt.Errorf("forward rule name is required")
	}
	if !protocol.IsValid() {
		return nil, fmt.Errorf("invalid protocol: %s", protocol)
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid status: %s", status)
	}
	// Validate external rule specific fields
	if ruleType.IsExternal() {
		if serverAddress == "" {
			return nil, fmt.Errorf("server address is required for external forward")
		}
		if targetNodeID == nil || *targetNodeID == 0 {
			return nil, fmt.Errorf("target node ID is required for external forward (protocol is derived from target node)")
		}
		// externalSource is optional
	}

	// Validate traffic multiplier
	if trafficMultiplier != nil {
		if *trafficMultiplier < 0 {
			return nil, fmt.Errorf("traffic multiplier cannot be negative: %f", *trafficMultiplier)
		}
		if *trafficMultiplier > 1000000 {
			return nil, fmt.Errorf("traffic multiplier exceeds maximum (1000000): %f", *trafficMultiplier)
		}
	}

	// Default ipVersion to auto if not set
	if ipVersion == "" {
		ipVersion = vo.IPVersionAuto
	}

	rule := &ForwardRule{
		id:                id,
		sid:               sid,
		agentID:           agentID,
		userID:            userID,
		subscriptionID:    subscriptionID,
		ruleType:          ruleType,
		exitAgentID:       exitAgentID,
		exitAgents:        exitAgents,
		chainAgentIDs:     chainAgentIDs,
		chainPortConfig:   chainPortConfig,
		tunnelHops:        tunnelHops,
		tunnelType:        tunnelType,
		name:              name,
		listenPort:        listenPort,
		targetAddress:     targetAddress,
		targetPort:        targetPort,
		targetNodeID:      targetNodeID,
		bindIP:            bindIP,
		ipVersion:         ipVersion,
		protocol:          protocol,
		status:            status,
		remark:            remark,
		uploadBytes:       uploadBytes,
		downloadBytes:     downloadBytes,
		trafficMultiplier: trafficMultiplier,
		sortOrder:         sortOrder,
		groupIDs:          groupIDs,
		serverAddress:     serverAddress,
		externalSource:    externalSource,
		externalRuleID:    externalRuleID,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}

	// Perform full validation to catch data corruption or manual DB modifications
	if err := rule.Validate(); err != nil {
		return nil, fmt.Errorf("reconstructed rule failed validation: %w", err)
	}

	return rule, nil
}

// validateAddress validates the target address format.
// It accepts valid IP addresses or RFC 1123 compliant domain names.
func validateAddress(address string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	// Check if it's a valid IP
	if ip := net.ParseIP(address); ip != nil {
		return nil
	}

	// Validate as domain name using RFC 1123 hostname rules
	// Same regex as used in forwardagent.go for consistency
	if len(address) > 253 {
		return fmt.Errorf("domain name too long (max 253 characters)")
	}

	// Use the existing domainNameRegex pattern
	// Pattern: ^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$
	if !domainNameRegex.MatchString(address) {
		return fmt.Errorf("invalid domain name format (must comply with RFC 1123)")
	}

	return nil
}

// Getters

// ID returns the forward rule ID.
func (r *ForwardRule) ID() uint {
	return r.id
}

// SID returns the Stripe-style prefixed ID (fr_xxx).
func (r *ForwardRule) SID() string {
	return r.sid
}

// AgentID returns the forward agent ID.
func (r *ForwardRule) AgentID() uint {
	return r.agentID
}

// UserID returns the user ID.
func (r *ForwardRule) UserID() *uint {
	return r.userID
}

// SubscriptionID returns the subscription ID.
func (r *ForwardRule) SubscriptionID() *uint {
	return r.subscriptionID
}

// IsSubscriptionBound returns true if the rule is bound to a subscription.
func (r *ForwardRule) IsSubscriptionBound() bool {
	return r.subscriptionID != nil && *r.subscriptionID != 0
}

// IsUserOwned returns true if the rule is owned by a user.
func (r *ForwardRule) IsUserOwned() bool {
	return r.userID != nil && *r.userID != 0
}

// Scope returns the ownership scope of this rule.
// System scope for admin-created rules, User scope for user-owned rules.
func (r *ForwardRule) Scope() vo.RuleScope {
	if r.IsUserOwned() {
		return vo.UserScope(*r.userID)
	}
	return vo.SystemScope()
}

// RuleType returns the rule type.
func (r *ForwardRule) RuleType() vo.ForwardRuleType {
	return r.ruleType
}

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

// Name returns the forward rule name.
func (r *ForwardRule) Name() string {
	return r.name
}

// ListenPort returns the listen port.
func (r *ForwardRule) ListenPort() uint16 {
	return r.listenPort
}

// TargetAddress returns the target address.
func (r *ForwardRule) TargetAddress() string {
	return r.targetAddress
}

// TargetPort returns the target port.
func (r *ForwardRule) TargetPort() uint16 {
	return r.targetPort
}

// TargetNodeID returns the target node ID.
func (r *ForwardRule) TargetNodeID() *uint {
	return r.targetNodeID
}

// HasTargetNode returns true if targetNodeID is set.
func (r *ForwardRule) HasTargetNode() bool {
	return r.targetNodeID != nil && *r.targetNodeID != 0
}

// BindIP returns the bind IP address for outbound connections.
func (r *ForwardRule) BindIP() string {
	return r.bindIP
}

// IPVersion returns the IP version preference.
func (r *ForwardRule) IPVersion() vo.IPVersion {
	return r.ipVersion
}

// Target returns the full target address with port.
func (r *ForwardRule) Target() string {
	return net.JoinHostPort(r.targetAddress, strconv.Itoa(int(r.targetPort)))
}

// Protocol returns the protocol.
func (r *ForwardRule) Protocol() vo.ForwardProtocol {
	return r.protocol
}

// Status returns the status.
func (r *ForwardRule) Status() vo.ForwardStatus {
	return r.status
}

// Remark returns the remark.
func (r *ForwardRule) Remark() string {
	return r.remark
}

// UploadBytes returns the upload bytes count with traffic multiplier applied.
func (r *ForwardRule) UploadBytes() int64 {
	multiplier := r.GetEffectiveMultiplier()
	return int64(float64(r.uploadBytes) * multiplier)
}

// DownloadBytes returns the download bytes count with traffic multiplier applied.
func (r *ForwardRule) DownloadBytes() int64 {
	multiplier := r.GetEffectiveMultiplier()
	return int64(float64(r.downloadBytes) * multiplier)
}

// TotalBytes returns the total bytes count with traffic multiplier applied.
func (r *ForwardRule) TotalBytes() int64 {
	return r.UploadBytes() + r.DownloadBytes()
}

// GetRawUploadBytes returns the raw upload bytes without multiplier (for internal use).
func (r *ForwardRule) GetRawUploadBytes() int64 {
	return r.uploadBytes
}

// GetRawDownloadBytes returns the raw download bytes without multiplier (for internal use).
func (r *ForwardRule) GetRawDownloadBytes() int64 {
	return r.downloadBytes
}

// GetRawTotalBytes returns the raw total bytes without multiplier (for internal use).
func (r *ForwardRule) GetRawTotalBytes() int64 {
	return r.uploadBytes + r.downloadBytes
}

// GetTrafficMultiplier returns the configured traffic multiplier (may be nil).
func (r *ForwardRule) GetTrafficMultiplier() *float64 {
	return r.trafficMultiplier
}

// SortOrder returns the sort order.
func (r *ForwardRule) SortOrder() int {
	return r.sortOrder
}

// GroupIDs returns the resource group IDs.
func (r *ForwardRule) GroupIDs() []uint {
	return r.groupIDs
}

// SetGroupIDs sets the resource group IDs.
func (r *ForwardRule) SetGroupIDs(groupIDs []uint) {
	r.groupIDs = groupIDs
	r.updatedAt = biztime.NowUTC()
}

// AddGroupID adds a resource group ID if not already present.
// Returns true if the group ID was added, false if it already exists.
func (r *ForwardRule) AddGroupID(groupID uint) bool {
	for _, id := range r.groupIDs {
		if id == groupID {
			return false // already exists
		}
	}
	r.groupIDs = append(r.groupIDs, groupID)
	r.updatedAt = biztime.NowUTC()
	return true
}

// RemoveGroupID removes a resource group ID.
// Returns true if the group ID was removed, false if not found.
func (r *ForwardRule) RemoveGroupID(groupID uint) bool {
	for i, id := range r.groupIDs {
		if id == groupID {
			r.groupIDs = append(r.groupIDs[:i], r.groupIDs[i+1:]...)
			r.updatedAt = biztime.NowUTC()
			return true
		}
	}
	return false // not found
}

// HasGroupID checks if the rule belongs to a specific resource group.
func (r *ForwardRule) HasGroupID(groupID uint) bool {
	for _, id := range r.groupIDs {
		if id == groupID {
			return true
		}
	}
	return false
}

// ServerAddress returns the server address for external rules.
func (r *ForwardRule) ServerAddress() string {
	return r.serverAddress
}

// ExternalSource returns the external source identifier.
func (r *ForwardRule) ExternalSource() string {
	return r.externalSource
}

// ExternalRuleID returns the external rule ID reference.
func (r *ForwardRule) ExternalRuleID() string {
	return r.externalRuleID
}

// IsExternal returns true if this is an external forward rule.
func (r *ForwardRule) IsExternal() bool {
	return r.ruleType.IsExternal()
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

// CalculateNodeCount calculates the total number of nodes in the forward chain.
func (r *ForwardRule) CalculateNodeCount() int {
	switch r.ruleType {
	case vo.ForwardRuleTypeDirect:
		return 1 // Only entry agent
	case vo.ForwardRuleTypeEntry:
		// Entry + Exit: load balancing selects one exit agent per connection
		// So the actual node count in the forwarding path is always 2
		return 2 // Entry + one Exit (load balancing selects one at a time)
	case vo.ForwardRuleTypeChain:
		// Chain: Entry -> Chain[0] -> ... -> Chain[n-1] -> Target
		chainCount := 0
		if r.chainAgentIDs != nil {
			chainCount = len(r.chainAgentIDs)
		}
		return 1 + chainCount // Entry + Chain agents
	case vo.ForwardRuleTypeDirectChain:
		chainCount := 0
		if r.chainAgentIDs != nil {
			chainCount = len(r.chainAgentIDs)
		}
		return 2 + chainCount // Entry + Chain + Exit
	case vo.ForwardRuleTypeExternal:
		return 1 // External rules have no agents, traffic multiplier calculation returns 1.0
	default:
		return 1 // Safe fallback
	}
}

// GetEffectiveMultiplier returns the effective traffic multiplier to use.
// If a multiplier is configured, it uses that value.
// Otherwise, it auto-calculates based on node count (1 / nodeCount).
func (r *ForwardRule) GetEffectiveMultiplier() float64 {
	if r.trafficMultiplier != nil {
		return *r.trafficMultiplier
	}

	nodeCount := r.CalculateNodeCount()
	if nodeCount <= 0 {
		// Safety fallback, should not happen in practice
		return 1.0
	}

	return 1.0 / float64(nodeCount)
}

// CreatedAt returns when the rule was created.
func (r *ForwardRule) CreatedAt() time.Time {
	return r.createdAt
}

// UpdatedAt returns when the rule was last updated.
func (r *ForwardRule) UpdatedAt() time.Time {
	return r.updatedAt
}

// Setters and business operations

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
	// Clear single exitAgentID when switching to multiple exit agents
	r.exitAgentID = 0
	r.exitAgents = exitAgents
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

// RecordTraffic records traffic bytes.
func (r *ForwardRule) RecordTraffic(upload, download int64) {
	r.uploadBytes += upload
	r.downloadBytes += download
	r.updatedAt = biztime.NowUTC()
}

// ResetTraffic resets the traffic counters.
func (r *ForwardRule) ResetTraffic() {
	r.uploadBytes = 0
	r.downloadBytes = 0
	r.updatedAt = biztime.NowUTC()
}

// IsEnabled checks if the rule is enabled.
func (r *ForwardRule) IsEnabled() bool {
	return r.status.IsEnabled()
}

// Validate performs domain-level validation.
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
		}
		// Entry rules now also require target information (to be passed to exit agent)
		hasTarget := r.targetAddress != "" && r.targetPort != 0
		hasTargetNode := r.targetNodeID != nil && *r.targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return fmt.Errorf("either target address+port or target node ID is required for entry forward")
		}
		if hasTarget && hasTargetNode {
			return fmt.Errorf("target address+port and target node ID are mutually exclusive for entry forward")
		}
	case vo.ForwardRuleTypeChain:
		if r.listenPort == 0 {
			return fmt.Errorf("listen port is required for chain forward")
		}
		if len(r.chainAgentIDs) == 0 {
			return fmt.Errorf("chain agent IDs is required for chain forward")
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
						return fmt.Errorf("chain_port_config missing valid port for agent ID %d (required for direct hop)", chainAgentID)
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
	case vo.ForwardRuleTypeDirectChain:
		if r.listenPort == 0 {
			return fmt.Errorf("listen port is required for direct_chain forward")
		}
		if len(r.chainAgentIDs) == 0 {
			return fmt.Errorf("chain agent IDs is required for direct_chain forward")
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
		// Direct chain rules require target information (at the end of chain)
		hasTarget := r.targetAddress != "" && r.targetPort != 0
		hasTargetNode := r.targetNodeID != nil && *r.targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return fmt.Errorf("either target address+port or target node ID is required for direct_chain forward")
		}
		if hasTarget && hasTargetNode {
			return fmt.Errorf("target address+port and target node ID are mutually exclusive for direct_chain forward")
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
