// Package value_objects provides value objects for the forward domain.
package valueobjects

// ForwardRuleType represents the type of forward rule.
type ForwardRuleType string

const (
	// ForwardRuleTypeDirect forwards traffic directly to the target.
	ForwardRuleTypeDirect ForwardRuleType = "direct"
	// ForwardRuleTypeEntry is the entry point that forwards traffic to exit agent via WS tunnel.
	// The target information is configured on the entry rule, and the exit agent receives it from the entry rule.
	ForwardRuleTypeEntry ForwardRuleType = "entry"
	// ForwardRuleTypeChain is a multi-hop chain forward rule.
	// Traffic flows through multiple agents: entry -> relay1 -> relay2 -> ... -> exit -> target.
	// The chain_agent_ids field stores the ordered list of intermediate agent IDs.
	ForwardRuleTypeChain ForwardRuleType = "chain"
	// ForwardRuleTypeDirectChain is a multi-hop direct TCP/UDP forward rule.
	// Traffic is directly forwarded through TCP/UDP connections across multiple agents.
	// Each agent listens on a specific port defined in chain_port_config.
	ForwardRuleTypeDirectChain ForwardRuleType = "direct_chain"
	// ForwardRuleTypeExternal is for external/third-party forward rules.
	// External rules do not require an agent (agentID=0), instead they use serverAddress
	// directly for subscription delivery. Protocol info is derived from targetNodeID (required).
	ForwardRuleTypeExternal ForwardRuleType = "external"
)

var validForwardRuleTypes = map[ForwardRuleType]bool{
	ForwardRuleTypeDirect:      true,
	ForwardRuleTypeEntry:       true,
	ForwardRuleTypeChain:       true,
	ForwardRuleTypeDirectChain: true,
	ForwardRuleTypeExternal:    true,
}

// String returns the string representation.
func (t ForwardRuleType) String() string {
	return string(t)
}

// IsValid checks if the rule type is valid.
func (t ForwardRuleType) IsValid() bool {
	return validForwardRuleTypes[t]
}

// IsDirect checks if this is a direct forward rule.
func (t ForwardRuleType) IsDirect() bool {
	return t == ForwardRuleTypeDirect
}

// IsEntry checks if this is an entry rule.
func (t ForwardRuleType) IsEntry() bool {
	return t == ForwardRuleTypeEntry
}

// IsChain checks if this is a chain forward rule.
func (t ForwardRuleType) IsChain() bool {
	return t == ForwardRuleTypeChain
}

// IsDirectChain checks if this is a direct_chain forward rule.
func (t ForwardRuleType) IsDirectChain() bool {
	return t == ForwardRuleTypeDirectChain
}

// IsExternal checks if this is an external forward rule.
func (t ForwardRuleType) IsExternal() bool {
	return t == ForwardRuleTypeExternal
}

// RequiresTarget checks if this rule type requires target address/port.
func (t ForwardRuleType) RequiresTarget() bool {
	return t == ForwardRuleTypeDirect
}

// RequiresExitAgent checks if this rule type requires exit agent ID.
func (t ForwardRuleType) RequiresExitAgent() bool {
	return t == ForwardRuleTypeEntry
}

// RequiresListenPort checks if this rule type requires listen port.
func (t ForwardRuleType) RequiresListenPort() bool {
	return t == ForwardRuleTypeDirect || t == ForwardRuleTypeEntry || t == ForwardRuleTypeChain || t == ForwardRuleTypeDirectChain || t == ForwardRuleTypeExternal
}

// RequiresChainAgents checks if this rule type requires chain agent IDs.
func (t ForwardRuleType) RequiresChainAgents() bool {
	return t == ForwardRuleTypeChain || t == ForwardRuleTypeDirectChain
}

// RequiresChainPortConfig checks if this rule type requires chain port configuration.
func (t ForwardRuleType) RequiresChainPortConfig() bool {
	return t == ForwardRuleTypeDirectChain
}

// RequiresAgent checks if this rule type requires an agent (agentID > 0).
// External rules do not require an agent.
func (t ForwardRuleType) RequiresAgent() bool {
	return t != ForwardRuleTypeExternal
}

// RequiresServerAddress checks if this rule type requires a server address.
// Only external rules require a server address for subscription delivery.
func (t ForwardRuleType) RequiresServerAddress() bool {
	return t == ForwardRuleTypeExternal
}

// RequiresExternalSource checks if this rule type supports external source field.
// Note: externalSource is optional for external rules.
func (t ForwardRuleType) RequiresExternalSource() bool {
	return t == ForwardRuleTypeExternal
}
