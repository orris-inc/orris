package node

import "context"

type NodeGroupRepository interface {
	Create(ctx context.Context, group *NodeGroup) error
	GetByID(ctx context.Context, id uint) (*NodeGroup, error)
	GetByName(ctx context.Context, name string) (*NodeGroup, error)
	Update(ctx context.Context, group *NodeGroup) error
	Delete(ctx context.Context, id uint) error

	List(ctx context.Context, filter NodeGroupFilter) ([]*NodeGroup, int64, error)
	GetPublicGroups(ctx context.Context) ([]*NodeGroup, error)
	GetBySubscriptionPlanID(ctx context.Context, planID uint) ([]*NodeGroup, error)

	AddNode(ctx context.Context, groupID, nodeID uint) error
	RemoveNode(ctx context.Context, groupID, nodeID uint) error
	GetNodesByGroupID(ctx context.Context, groupID uint) ([]*Node, error)

	AssociateSubscriptionPlan(ctx context.Context, groupID, planID uint) error
	DisassociateSubscriptionPlan(ctx context.Context, groupID, planID uint) error
	GetSubscriptionPlansByGroupID(ctx context.Context, groupID uint) ([]uint, error)

	ExistsByName(ctx context.Context, name string) (bool, error)
}

type NodeGroupFilter struct {
	Name     *string
	IsPublic *bool
	Page     int
	PageSize int
	SortBy   string
	SortDesc bool
}
