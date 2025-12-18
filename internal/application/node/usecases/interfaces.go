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
