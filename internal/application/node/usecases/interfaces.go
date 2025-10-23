package usecases

import "context"

type CreateNodeExecutor interface {
	Execute(ctx context.Context, cmd CreateNodeCommand) (*CreateNodeResult, error)
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
