package usecases

import "context"

// ConfigSyncNotifier defines the interface for notifying configuration changes to agents.
// This interface is implemented by ConfigSyncService and used by UseCases to avoid circular dependencies.
type ConfigSyncNotifier interface {
	// NotifyRuleChange notifies the agent about a rule change
	// changeType can be: "added", "removed", "updated"
	NotifyRuleChange(ctx context.Context, agentID uint, ruleShortID string, changeType string) error
}

// AgentAddressChangeNotifier defines the interface for notifying agent address changes.
// When an agent's public address or tunnel address changes, all rules using this agent
// need to be re-synced to the relevant entry agents.
type AgentAddressChangeNotifier interface {
	// NotifyAgentAddressChange notifies all entry agents that have rules using this agent.
	// This is called when an agent's public_address or tunnel_address changes.
	NotifyAgentAddressChange(ctx context.Context, agentID uint) error
}
