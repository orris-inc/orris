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

// RuleSyncStatusResponse represents the response for querying rule sync status from admin side.
type RuleSyncStatusResponse struct {
	AgentID   string               `json:"agent_id"`   // Stripe-style agent ID
	Rules     []RuleSyncStatusItem `json:"rules"`      // List of rule sync statuses
	UpdatedAt int64                `json:"updated_at"` // Last update timestamp (Unix seconds)
}

// RuleSyncStatusQueryResult represents the result of querying rule sync status from cache.
type RuleSyncStatusQueryResult struct {
	Rules     []RuleSyncStatusItem
	UpdatedAt int64 // Unix timestamp of last update
}
