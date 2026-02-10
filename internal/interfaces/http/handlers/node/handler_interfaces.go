package node

import (
	"context"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/application/node/usecases"
)

// Use case interfaces for NodeHandler - enables unit testing with mocks.

type createNodeUseCase interface {
	Execute(ctx context.Context, cmd usecases.CreateNodeCommand) (*usecases.CreateNodeResult, error)
}

type getNodeUseCase interface {
	Execute(ctx context.Context, query usecases.GetNodeQuery) (*usecases.GetNodeResult, error)
}

type updateNodeUseCase interface {
	Execute(ctx context.Context, cmd usecases.UpdateNodeCommand) (*usecases.UpdateNodeResult, error)
}

type deleteNodeUseCase interface {
	Execute(ctx context.Context, cmd usecases.DeleteNodeCommand) (*usecases.DeleteNodeResult, error)
}

type listNodesUseCase interface {
	Execute(ctx context.Context, query usecases.ListNodesQuery) (*usecases.ListNodesResult, error)
}

type generateNodeTokenUseCase interface {
	Execute(ctx context.Context, cmd usecases.GenerateNodeTokenCommand) (*usecases.GenerateNodeTokenResult, error)
}

type generateNodeInstallScriptUseCase interface {
	Execute(ctx context.Context, query usecases.GenerateNodeInstallScriptQuery) (*usecases.GenerateNodeInstallScriptResult, error)
}

type generateBatchInstallScriptUseCase interface {
	Execute(ctx context.Context, query usecases.GenerateBatchInstallScriptQuery) (*usecases.GenerateBatchInstallScriptResult, error)
}

// Use case interfaces for UserNodeHandler - enables unit testing with mocks.

type createUserNodeUseCase interface {
	Execute(ctx context.Context, cmd usecases.CreateUserNodeCommand) (*usecases.CreateUserNodeResult, error)
}

type listUserNodesUseCase interface {
	Execute(ctx context.Context, q usecases.ListUserNodesQuery) (*usecases.ListUserNodesResult, error)
}

type getUserNodeUseCase interface {
	Execute(ctx context.Context, q usecases.GetUserNodeQuery) (*dto.UserNodeDTO, error)
}

type updateUserNodeUseCase interface {
	Execute(ctx context.Context, cmd usecases.UpdateUserNodeCommand) (*dto.UserNodeDTO, error)
}

type deleteUserNodeUseCase interface {
	Execute(ctx context.Context, cmd usecases.DeleteUserNodeCommand) error
}

type regenerateUserNodeTokenUseCase interface {
	Execute(ctx context.Context, cmd usecases.RegenerateUserNodeTokenCommand) (*usecases.RegenerateUserNodeTokenResult, error)
}

type getUserNodeUsageUseCase interface {
	Execute(ctx context.Context, query usecases.GetUserNodeUsageQuery) (*usecases.GetUserNodeUsageResult, error)
}

type getUserNodeInstallScriptUseCase interface {
	Execute(ctx context.Context, query usecases.GetUserNodeInstallScriptQuery) (*usecases.GetUserNodeInstallScriptResult, error)
}

type getUserBatchInstallScriptUseCase interface {
	Execute(ctx context.Context, query usecases.GetUserBatchInstallScriptQuery) (*usecases.GetUserBatchInstallScriptResult, error)
}
