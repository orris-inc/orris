package node

// CreateNodeGroupRequest represents the request body for creating a node group
type CreateNodeGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
	IsPublic    bool   `json:"is_public"`
	SortOrder   int    `json:"sort_order,omitempty"`
}

// UpdateNodeGroupRequest represents the request body for updating a node group
type UpdateNodeGroupRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsPublic    *bool   `json:"is_public,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

// AddNodeToGroupRequest represents the request body for adding a node to a group
type AddNodeToGroupRequest struct {
	NodeShortID string `json:"node_id" binding:"required"`
}

// ListNodeGroupsRequest represents the query parameters for listing node groups
type ListNodeGroupsRequest struct {
	Page     int
	PageSize int
	IsPublic *bool
}

// AssociatePlanRequest represents the request body for associating a plan with a group
type AssociatePlanRequest struct {
	PlanID uint `json:"plan_id" binding:"required"`
}

// BatchAddNodesToGroupRequest represents the request body for batch adding nodes to a group
type BatchAddNodesToGroupRequest struct {
	NodeIDs []uint `json:"node_ids" binding:"required,min=1,max=100" example:"1,2,3,4,5"`
}

// BatchRemoveNodesFromGroupRequest represents the request body for batch removing nodes from a group
type BatchRemoveNodesFromGroupRequest struct {
	NodeIDs []uint `json:"node_ids" binding:"required,min=1,max=100" example:"1,2,3,4,5"`
}
