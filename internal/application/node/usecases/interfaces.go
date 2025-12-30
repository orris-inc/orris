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

// NodeConfigChangeNotifier defines the interface for notifying node configuration changes.
// When a node's configuration changes (including route config), the node agent needs to be notified
// to reload its configuration.
type NodeConfigChangeNotifier interface {
	// NotifyConfigChange notifies the node agent about a configuration change.
	// This is called when a node's configuration (including route config) is updated.
	NotifyConfigChange(ctx context.Context, nodeID uint) error
}
