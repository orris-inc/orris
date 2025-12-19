package dto

import (
	"time"
)

// GrantEntitlementRequest represents the request to grant an entitlement
type GrantEntitlementRequest struct {
	SubjectType  string         `json:"subject_type"`             // "user"
	SubjectID    uint           `json:"subject_id"`               // ID of the subject
	ResourceType string         `json:"resource_type"`            // "node" | "forward_agent" | "feature"
	ResourceID   uint           `json:"resource_id"`              // ID of the resource
	SourceType   string         `json:"source_type"`              // "direct" | "promotion" | "trial"
	SourceID     uint           `json:"source_id"`                // ID of the source
	ExpiresAt    *string        `json:"expires_at,omitempty"`     // RFC3339 format
	Metadata     map[string]any `json:"metadata,omitempty"`       // Additional metadata
}

// RevokeEntitlementRequest represents the request to revoke an entitlement
type RevokeEntitlementRequest struct {
	EntitlementID uint `json:"entitlement_id"` // ID of the entitlement to revoke
}

// ListEntitlementsRequest represents the request to list entitlements
type ListEntitlementsRequest struct {
	SubjectType  *string `json:"subject_type,omitempty"`  // Filter by subject type
	SubjectID    *uint   `json:"subject_id,omitempty"`    // Filter by subject ID
	ResourceType *string `json:"resource_type,omitempty"` // Filter by resource type
	ResourceID   *uint   `json:"resource_id,omitempty"`   // Filter by resource ID
	Status       *string `json:"status,omitempty"`        // Filter by status ("active" | "expired" | "revoked")
	SourceType   *string `json:"source_type,omitempty"`   // Filter by source type ("subscription" | "direct" | "promotion" | "trial")
	Page         int     `json:"page"`                    // Page number (1-based)
	PageSize     int     `json:"page_size"`               // Number of items per page
}

// EntitlementResponse represents the response for a single entitlement
type EntitlementResponse struct {
	ID           uint           `json:"id"`            // Entitlement ID
	SubjectType  string         `json:"subject_type"`  // Subject type
	SubjectID    uint           `json:"subject_id"`    // Subject ID
	ResourceType string         `json:"resource_type"` // Resource type
	ResourceID   uint           `json:"resource_id"`   // Resource ID
	SourceType   string         `json:"source_type"`   // Source type
	SourceID     uint           `json:"source_id"`     // Source ID
	Status       string         `json:"status"`        // Entitlement status
	ExpiresAt    *time.Time     `json:"expires_at,omitempty"` // Expiration time
	Metadata     map[string]any `json:"metadata,omitempty"`   // Additional metadata
	CreatedAt    time.Time      `json:"created_at"`    // Creation timestamp
	UpdatedAt    time.Time      `json:"updated_at"`    // Last update timestamp
}

// ListEntitlementsResponse represents the response for listing entitlements
type ListEntitlementsResponse struct {
	Entitlements []*EntitlementResponse `json:"entitlements"` // List of entitlements
	Pagination   PaginationResponse     `json:"pagination"`   // Pagination metadata
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Page       int `json:"page"`        // Current page number
	PageSize   int `json:"page_size"`   // Items per page
	Total      int `json:"total"`       // Total number of items
	TotalPages int `json:"total_pages"` // Total number of pages
}

// AccessibleResourcesResponse represents accessible resources for a user
type AccessibleResourcesResponse struct {
	ResourceType string `json:"resource_type"` // Type of resources
	ResourceIDs  []uint `json:"resource_ids"`  // List of accessible resource IDs
}
