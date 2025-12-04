// Package value_objects provides value objects for the forward domain.
package value_objects

// ForwardRuleType represents the type of forward rule.
type ForwardRuleType string

const (
	// ForwardRuleTypeDirect forwards traffic directly to the target.
	ForwardRuleTypeDirect ForwardRuleType = "direct"
	// ForwardRuleTypeEntry is the entry point that forwards traffic to exit agent via WS tunnel.
	ForwardRuleTypeEntry ForwardRuleType = "entry"
	// ForwardRuleTypeExit receives traffic from entry agent and forwards to the target.
	ForwardRuleTypeExit ForwardRuleType = "exit"
)

var validForwardRuleTypes = map[ForwardRuleType]bool{
	ForwardRuleTypeDirect: true,
	ForwardRuleTypeEntry:  true,
	ForwardRuleTypeExit:   true,
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

// IsExit checks if this is an exit rule.
func (t ForwardRuleType) IsExit() bool {
	return t == ForwardRuleTypeExit
}

// RequiresTarget checks if this rule type requires target address/port.
func (t ForwardRuleType) RequiresTarget() bool {
	return t == ForwardRuleTypeDirect || t == ForwardRuleTypeExit
}

// RequiresExitAgent checks if this rule type requires exit agent ID.
func (t ForwardRuleType) RequiresExitAgent() bool {
	return t == ForwardRuleTypeEntry
}

// RequiresListenPort checks if this rule type requires listen port.
func (t ForwardRuleType) RequiresListenPort() bool {
	return t == ForwardRuleTypeDirect || t == ForwardRuleTypeEntry || t == ForwardRuleTypeExit
}
