// Package dto provides data transfer objects for the forward domain.
package dto

// AgentVersionInfo contains version information for a forward agent.
type AgentVersionInfo struct {
	AgentID        string `json:"agent_id"`               // Stripe-style agent ID
	CurrentVersion string `json:"current_version"`        // Agent's current version
	LatestVersion  string `json:"latest_version"`         // Latest available version
	HasUpdate      bool   `json:"has_update"`             // True if update is available
	Platform       string `json:"platform"`               // OS platform (linux, darwin, windows)
	Arch           string `json:"arch"`                   // CPU architecture (amd64, arm64)
	DownloadURL    string `json:"download_url,omitempty"` // Download URL for the update
	PublishedAt    string `json:"published_at,omitempty"` // Release publish time
}

// UpdatePayload is the payload sent to agent for update command.
type UpdatePayload struct {
	Version     string `json:"version"`      // Target version (e.g., "v1.2.3")
	DownloadURL string `json:"download_url"` // Download URL for the binary
	Checksum    string `json:"checksum"`     // SHA256 checksum (if available)
}

// BatchUpdateRequest is the request body for batch update API.
type BatchUpdateRequest struct {
	AgentIDs  []string `json:"agent_ids"`  // Optional: specific agent IDs to update
	UpdateAll bool     `json:"update_all"` // Optional: update all agents with available updates
}

// BatchUpdateResponse is the response for batch update API.
type BatchUpdateResponse struct {
	Total     int                  `json:"total"`
	Succeeded []BatchUpdateSuccess `json:"succeeded"`
	Failed    []BatchUpdateFailed  `json:"failed"`
	Skipped   []BatchUpdateSkipped `json:"skipped"`
	Truncated bool                 `json:"truncated,omitempty"` // True if results were truncated due to limit
}

// BatchUpdateSuccess represents a successfully triggered update.
type BatchUpdateSuccess struct {
	AgentID       string `json:"agent_id"`
	CommandID     string `json:"command_id"`
	TargetVersion string `json:"target_version"`
}

// BatchUpdateFailed represents a failed update attempt.
type BatchUpdateFailed struct {
	AgentID string `json:"agent_id"`
	Reason  string `json:"reason"`
}

// BatchUpdateSkipped represents a skipped agent.
type BatchUpdateSkipped struct {
	AgentID string `json:"agent_id"`
	Reason  string `json:"reason"`
}
