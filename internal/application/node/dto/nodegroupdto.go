package dto

import (
	"time"

	"github.com/orris-inc/orris/internal/domain/node"
)

type NodeGroupDTO struct {
	ID                  uint                   `json:"id"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	NodeIDs             []uint                 `json:"node_ids"`
	SubscriptionPlanIDs []uint                 `json:"subscription_plan_ids"`
	IsPublic            bool                   `json:"is_public"`
	SortOrder           int                    `json:"sort_order"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	NodeCount           int                    `json:"node_count"`
	Version             int                    `json:"version"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

type CreateNodeGroupDTO struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description,omitempty"`
	IsPublic    bool   `json:"is_public"`
	SortOrder   int    `json:"sort_order"`
}

type UpdateNodeGroupDTO struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=2,max=100"`
	Description *string `json:"description,omitempty"`
	IsPublic    *bool   `json:"is_public,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

type NodeGroupListDTO struct {
	NodeGroups []*NodeGroupDTO    `json:"node_groups"`
	Pagination PaginationResponse `json:"pagination"`
}

type ListNodeGroupsRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	IsPublic *bool  `json:"is_public,omitempty" form:"is_public"`
	OrderBy  string `json:"order_by,omitempty" form:"order_by"`
	Order    string `json:"order,omitempty" form:"order" binding:"omitempty,oneof=asc desc"`
}

type AddNodeToGroupRequest struct {
	GroupID uint `json:"group_id" binding:"required"`
	NodeID  uint `json:"node_id" binding:"required"`
}

type RemoveNodeFromGroupRequest struct {
	GroupID uint `json:"group_id" binding:"required"`
	NodeID  uint `json:"node_id" binding:"required"`
}

type AssociatePlanRequest struct {
	GroupID uint `json:"group_id" binding:"required"`
	PlanID  uint `json:"plan_id" binding:"required"`
}

func ToNodeGroupDTO(ng *node.NodeGroup) *NodeGroupDTO {
	if ng == nil {
		return nil
	}

	nodeIDs := ng.NodeIDs()
	if nodeIDs == nil {
		nodeIDs = make([]uint, 0)
	}

	planIDs := ng.SubscriptionPlanIDs()
	if planIDs == nil {
		planIDs = make([]uint, 0)
	}

	return &NodeGroupDTO{
		ID:                  ng.ID(),
		Name:                ng.Name(),
		Description:         ng.Description(),
		NodeIDs:             nodeIDs,
		SubscriptionPlanIDs: planIDs,
		IsPublic:            ng.IsPublic(),
		SortOrder:           ng.SortOrder(),
		Metadata:            ng.Metadata(),
		NodeCount:           ng.NodeCount(),
		Version:             ng.Version(),
		CreatedAt:           ng.CreatedAt(),
		UpdatedAt:           ng.UpdatedAt(),
	}
}

func ToNodeGroupDTOList(groups []*node.NodeGroup) []*NodeGroupDTO {
	if groups == nil {
		return nil
	}

	dtos := make([]*NodeGroupDTO, 0, len(groups))
	for _, ng := range groups {
		if dto := ToNodeGroupDTO(ng); dto != nil {
			dtos = append(dtos, dto)
		}
	}

	return dtos
}
