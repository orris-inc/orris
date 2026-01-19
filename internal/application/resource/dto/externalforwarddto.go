package dto

import "time"

// ExternalForwardRuleSummaryResponse represents a summary of an external forward rule for resource group contexts.
type ExternalForwardRuleSummaryResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	ServerAddress  string    `json:"server_address"`
	ListenPort     uint16    `json:"listen_port"`
	ExternalSource string    `json:"external_source"`
	SortOrder      int       `json:"sort_order"`
	GroupSIDs      []string  `json:"group_ids,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// ListGroupExternalForwardRulesResponse represents a paginated list of external forward rules for a resource group.
type ListGroupExternalForwardRulesResponse struct {
	Items      []ExternalForwardRuleSummaryResponse `json:"items"`
	Total      int64                                `json:"total"`
	Page       int                                  `json:"page"`
	PageSize   int                                  `json:"page_size"`
	TotalPages int                                  `json:"total_pages"`
}
