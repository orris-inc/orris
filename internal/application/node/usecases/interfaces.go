package usecases

import "context"

type CreateNodeExecutor interface {
	Execute(ctx context.Context, cmd CreateNodeCommand) (*CreateNodeResult, error)
}

type GetNodeExecutor interface {
	Execute(ctx context.Context, query GetNodeQuery) (*GetNodeResult, error)
}

type UpdateNodeExecutor interface {
	Execute(ctx context.Context, cmd UpdateNodeCommand) (*UpdateNodeResult, error)
}

type DeleteNodeExecutor interface {
	Execute(ctx context.Context, cmd DeleteNodeCommand) (*DeleteNodeResult, error)
}

type ListNodesExecutor interface {
	Execute(ctx context.Context, query ListNodesQuery) (*ListNodesResult, error)
}

type GenerateNodeTokenExecutor interface {
	Execute(ctx context.Context, cmd GenerateNodeTokenCommand) (*GenerateNodeTokenResult, error)
}

type GetNodeTrafficStatsExecutor interface {
	Execute(ctx context.Context, query GetNodeTrafficStatsQuery) ([]*NodeTrafficStatsResult, error)
}

// NodeGroup use case executors
type CreateNodeGroupExecutor interface {
	Execute(ctx context.Context, cmd CreateNodeGroupCommand) (*CreateNodeGroupResult, error)
}

type GetNodeGroupExecutor interface {
	Execute(ctx context.Context, query GetNodeGroupQuery) (*GetNodeGroupResult, error)
}

type UpdateNodeGroupExecutor interface {
	Execute(ctx context.Context, cmd UpdateNodeGroupCommand) (*UpdateNodeGroupResult, error)
}

type DeleteNodeGroupExecutor interface {
	Execute(ctx context.Context, cmd DeleteNodeGroupCommand) (*DeleteNodeGroupResult, error)
}

type ListNodeGroupsExecutor interface {
	Execute(ctx context.Context, query ListNodeGroupsQuery) (*ListNodeGroupsResult, error)
}

type AddNodeToGroupExecutor interface {
	Execute(ctx context.Context, cmd AddNodeToGroupCommand) (*AddNodeToGroupResult, error)
}

type RemoveNodeFromGroupExecutor interface {
	Execute(ctx context.Context, cmd RemoveNodeFromGroupCommand) (*RemoveNodeFromGroupResult, error)
}

type ListGroupNodesExecutor interface {
	Execute(ctx context.Context, query ListGroupNodesQuery) (*ListGroupNodesResult, error)
}

type AssociateGroupWithPlanExecutor interface {
	Execute(ctx context.Context, cmd AssociateGroupWithPlanCommand) (*AssociateGroupWithPlanResult, error)
}

type DisassociateGroupFromPlanExecutor interface {
	Execute(ctx context.Context, cmd DisassociateGroupFromPlanCommand) (*DisassociateGroupFromPlanResult, error)
}
