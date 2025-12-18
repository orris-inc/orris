package node

import (
	"context"

	"github.com/orris-inc/orris/internal/shared/query"
)

type NodeGroupRepository interface {
	Create(ctx context.Context, group *NodeGroup) error
	GetByID(ctx context.Context, id uint) (*NodeGroup, error)
	Update(ctx context.Context, group *NodeGroup) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filter NodeGroupFilter) ([]*NodeGroup, int64, error)
	AddNode(ctx context.Context, groupID, nodeID uint) error
	RemoveNode(ctx context.Context, groupID, nodeID uint) error
	GetNodesByGroupID(ctx context.Context, groupID uint) ([]*Node, error)
	AssociateSubscriptionPlan(ctx context.Context, groupID, planID uint) error
	DisassociateSubscriptionPlan(ctx context.Context, groupID, planID uint) error
	GetSubscriptionPlansByGroupID(ctx context.Context, groupID uint) ([]uint, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
	// IsNodeInAnyGroup checks if a node is part of any node group
	IsNodeInAnyGroup(ctx context.Context, nodeID uint) (bool, error)
}

type NodeGroupFilter struct {
	query.BaseFilter
	Name     *string
	IsPublic *bool
}
