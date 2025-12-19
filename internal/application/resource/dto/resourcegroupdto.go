package dto

import "time"

// CreateResourceGroupRequest represents a request to create a new resource group
type CreateResourceGroupRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	PlanID      uint   `json:"plan_id" binding:"required"`
	Description string `json:"description,omitempty" binding:"max=500"`
}

// UpdateResourceGroupRequest represents a request to update a resource group
type UpdateResourceGroupRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=500"`
}

// ResourceGroupResponse represents a resource group in API responses
type ResourceGroupResponse struct {
	ID          uint      `json:"id"`
	SID         string    `json:"sid"`
	Name        string    `json:"name"`
	PlanID      uint      `json:"plan_id"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListResourceGroupsRequest represents a request to list resource groups
type ListResourceGroupsRequest struct {
	PlanID   *uint   `form:"plan_id,omitempty"`
	Status   *string `form:"status,omitempty"`
	Page     int     `form:"page,default=1" binding:"min=1"`
	PageSize int     `form:"page_size,default=20" binding:"min=1,max=100"`
}

// ListResourceGroupsResponse represents a paginated list of resource groups
type ListResourceGroupsResponse struct {
	Items      []ResourceGroupResponse `json:"items"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}
