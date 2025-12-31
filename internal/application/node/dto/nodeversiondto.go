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
