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

// NodeAddressChangeNotifier defines the interface for notifying node address changes.
// When a node's public IP changes, all rules targeting this node need to be re-synced
// to the relevant forward agents.
type NodeAddressChangeNotifier interface {
	// NotifyNodeAddressChange notifies all forward agents that have rules targeting this node.
	// This is called when a node's public_ipv4 or public_ipv6 changes.
	NotifyNodeAddressChange(ctx context.Context, nodeID uint) error
}

// AgentConfigChangeNotifier defines the interface for notifying agent configuration changes.
// This is used when agent-level settings (like blocked_protocols) change and need to be
// synced to the agent.
type AgentConfigChangeNotifier interface {
	// NotifyAgentBlockedProtocolsChange notifies an agent that its blocked protocols changed.
	// This triggers a full sync to deliver the updated configuration.
	NotifyAgentBlockedProtocolsChange(ctx context.Context, agentID uint) error
}
