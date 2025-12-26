// Package dto provides data transfer objects for the forward domain.
package dto

// RuleSyncStatusItem represents the sync and runtime status of a single forward rule.
type RuleSyncStatusItem struct {
	RuleID       string `json:"rule_id"`       // Stripe-style rule ID (e.g., "fr_xK9mP2vL3nQ")
	SyncStatus   string `json:"sync_status"`   // Sync status: synced, pending, failed
	RunStatus    string `json:"run_status"`    // Runtime status: running, stopped, error, starting
	ListenPort   uint16 `json:"listen_port"`   // Actual listening port
	Connections  int    `json:"connections"`   // Current number of connections
	ErrorMessage string `json:"error_message"` // Error message if any
	SyncedAt     int64  `json:"synced_at"`     // Last sync timestamp (Unix seconds)
}

// ReportRuleSyncStatusInput represents the input for ReportRuleSyncStatus use case.
type ReportRuleSyncStatusInput struct {
	AgentID uint
	Rules   []RuleSyncStatusItem
}

// RuleSyncStatusQueryResult represents the result of querying rule sync status from cache.
type RuleSyncStatusQueryResult struct {
	Rules     []RuleSyncStatusItem
	UpdatedAt int64 // Unix timestamp of last update
}

// AgentRuleSyncStatus represents the sync status of a single agent for a specific rule.
type AgentRuleSyncStatus struct {
	AgentID      string `json:"agent_id"`      // Stripe-style agent ID (e.g., "fa_xK9mP2vL3nQ")
	AgentName    string `json:"agent_name"`    // Agent name
	Position     int    `json:"position"`      // Position in forwarding chain (0=entry)
	SyncStatus   string `json:"sync_status"`   // Sync status: synced, pending, failed
	RunStatus    string `json:"run_status"`    // Runtime status: running, stopped, error, starting
	ListenPort   uint16 `json:"listen_port"`   // Actual listening port
	Connections  int    `json:"connections"`   // Current number of connections
	ErrorMessage string `json:"error_message"` // Error message if any
	SyncedAt     int64  `json:"synced_at"`     // Last sync timestamp (Unix seconds)
}

// RuleOverallStatusResponse represents the aggregated status response for a rule.
type RuleOverallStatusResponse struct {
	RuleID            string                `json:"rule_id"`             // Stripe-style rule ID (e.g., "fr_xK9mP2vL3nQ")
	OverallSyncStatus string                `json:"overall_sync_status"` // Aggregated sync status: synced, pending, failed
	OverallRunStatus  string                `json:"overall_run_status"`  // Aggregated run status: running, stopped, error, starting
	TotalAgents       int                   `json:"total_agents"`        // Total number of agents in chain
	HealthyAgents     int                   `json:"healthy_agents"`      // Number of healthy agents
	AgentStatuses     []AgentRuleSyncStatus `json:"agent_statuses"`      // Detailed status for each agent
	UpdatedAt         int64                 `json:"updated_at"`          // Last update timestamp (Unix seconds)
}

// GetRuleOverallStatusInput represents the input for querying rule overall status.
type GetRuleOverallStatusInput struct {
	RuleSID string // Stripe-style rule ID (e.g., "fr_xK9mP2vL3nQ")
}

// AggregateSyncStatus aggregates sync statuses with priority: failed > pending > synced.
func AggregateSyncStatus(statuses []string) string {
	hasFailed := false
	hasPending := false

	for _, status := range statuses {
		switch status {
		case "failed":
			hasFailed = true
		case "pending":
			hasPending = true
		}
	}

	if hasFailed {
		return "failed"
	}
	if hasPending {
		return "pending"
	}
	return "synced"
}

// AggregateRunStatus aggregates run statuses with priority: error > stopped > starting > unknown > running.
func AggregateRunStatus(statuses []string) string {
	hasError := false
	hasStopped := false
	hasStarting := false
	hasUnknown := false

	for _, status := range statuses {
		switch status {
		case "error":
			hasError = true
		case "stopped":
			hasStopped = true
		case "starting":
			hasStarting = true
		case "unknown":
			hasUnknown = true
		}
	}

	if hasError {
		return "error"
	}
	if hasStopped {
		return "stopped"
	}
	if hasStarting {
		return "starting"
	}
	if hasUnknown {
		return "unknown"
	}
	return "running"
}
