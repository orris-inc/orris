// Package dto provides data transfer objects for the node domain.
package dto

// NodeVersionInfo contains version information for a node agent.
type NodeVersionInfo struct {
	NodeID         string `json:"node_id"`                // Stripe-style node ID
	CurrentVersion string `json:"current_version"`        // Node agent's current version
	LatestVersion  string `json:"latest_version"`         // Latest available version
	HasUpdate      bool   `json:"has_update"`             // True if update is available
	Platform       string `json:"platform"`               // OS platform (linux, darwin, windows)
	Arch           string `json:"arch"`                   // CPU architecture (amd64, arm64)
	DownloadURL    string `json:"download_url,omitempty"` // Download URL for the update
	PublishedAt    string `json:"published_at,omitempty"` // Release publish time
}

// NodeUpdatePayload is the payload sent to node agent for update command.
type NodeUpdatePayload struct {
	Version     string `json:"version"`      // Target version (e.g., "v1.2.3")
	DownloadURL string `json:"download_url"` // Download URL for the binary
	Checksum    string `json:"checksum"`     // SHA256 checksum (if available)
}

// BatchUpdateRequest is the request body for batch update API.
type BatchUpdateRequest struct {
	NodeIDs   []string `json:"node_ids"`   // Optional: specific node IDs to update
	UpdateAll bool     `json:"update_all"` // Optional: update all nodes with available updates
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
	NodeID        string `json:"node_id"`
	CommandID     string `json:"command_id"`
	TargetVersion string `json:"target_version"`
}

// BatchUpdateFailed represents a failed update attempt.
type BatchUpdateFailed struct {
	NodeID string `json:"node_id"`
	Reason string `json:"reason"`
}

// BatchUpdateSkipped represents a skipped node.
type BatchUpdateSkipped struct {
	NodeID string `json:"node_id"`
	Reason string `json:"reason"`
}
