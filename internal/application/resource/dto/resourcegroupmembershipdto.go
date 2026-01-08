package dto

import "time"

// AddNodesToGroupRequest represents a request to add nodes to a resource group
type AddNodesToGroupRequest struct {
	NodeSIDs []string `json:"node_ids" binding:"required,min=1,dive,required"`
}

// RemoveNodesFromGroupRequest represents a request to remove nodes from a resource group
type RemoveNodesFromGroupRequest struct {
	NodeSIDs []string `json:"node_ids" binding:"required,min=1,dive,required"`
}

// AddForwardAgentsToGroupRequest represents a request to add forward agents to a resource group
type AddForwardAgentsToGroupRequest struct {
	AgentSIDs []string `json:"agent_ids" binding:"required,min=1,dive,required"`
}

// RemoveForwardAgentsFromGroupRequest represents a request to remove forward agents from a resource group
type RemoveForwardAgentsFromGroupRequest struct {
	AgentSIDs []string `json:"agent_ids" binding:"required,min=1,dive,required"`
}

// NodeSummaryResponse represents a node in group member list (simplified)
type NodeSummaryResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	GroupSIDs []string  `json:"group_ids,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ForwardAgentSummaryResponse represents a forward agent in group member list (simplified)
type ForwardAgentSummaryResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	GroupSID  *string   `json:"group_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ListGroupNodesResponse represents the response for listing nodes in a group
type ListGroupNodesResponse struct {
	Items      []NodeSummaryResponse `json:"items"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
}

// ListGroupForwardAgentsResponse represents the response for listing forward agents in a group
type ListGroupForwardAgentsResponse struct {
	Items      []ForwardAgentSummaryResponse `json:"items"`
	Total      int64                         `json:"total"`
	Page       int                           `json:"page"`
	PageSize   int                           `json:"page_size"`
	TotalPages int                           `json:"total_pages"`
}

// BatchOperationResult represents the result of a batch operation
type BatchOperationResult struct {
	Succeeded []string            `json:"succeeded"`
	Failed    []BatchOperationErr `json:"failed,omitempty"`
}

// BatchOperationErr represents an error for a single item in batch operation
type BatchOperationErr struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// ListGroupMembersRequest represents a request to list members in a group
type ListGroupMembersRequest struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	PageSize int    `form:"page_size,default=20" binding:"min=1,max=100"`
	OrderBy  string `form:"order_by"`
	Order    string `form:"order"`
}

// AddForwardRulesToGroupRequest represents a request to add forward rules to a resource group
type AddForwardRulesToGroupRequest struct {
	RuleSIDs []string `json:"rule_ids" binding:"required,min=1,dive,required"`
}

// RemoveForwardRulesFromGroupRequest represents a request to remove forward rules from a resource group
type RemoveForwardRulesFromGroupRequest struct {
	RuleSIDs []string `json:"rule_ids" binding:"required,min=1,dive,required"`
}

// ForwardRuleSummaryResponse represents a forward rule in group member list (simplified)
type ForwardRuleSummaryResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Protocol   string    `json:"protocol"`
	ListenPort uint16    `json:"listen_port"`
	SortOrder  int       `json:"sort_order"`
	GroupSIDs  []string  `json:"group_ids,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ListGroupForwardRulesResponse represents the response for listing forward rules in a group
type ListGroupForwardRulesResponse struct {
	Items      []ForwardRuleSummaryResponse `json:"items"`
	Total      int64                        `json:"total"`
	Page       int                          `json:"page"`
	PageSize   int                          `json:"page_size"`
	TotalPages int                          `json:"total_pages"`
}
