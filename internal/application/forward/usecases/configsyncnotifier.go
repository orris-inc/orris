package usecases

import "context"

// ConfigSyncNotifier defines the interface for notifying configuration changes to agents.
// This interface is implemented by ConfigSyncService and used by UseCases to avoid circular dependencies.
type ConfigSyncNotifier interface {
	// NotifyRuleChange notifies the agent about a rule change
	// changeType can be: "added", "removed", "updated"
	NotifyRuleChange(ctx context.Context, agentID uint, ruleShortID string, changeType string) error
}
