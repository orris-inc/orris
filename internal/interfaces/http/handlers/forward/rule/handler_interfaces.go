package rule

import (
	"context"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
)

// Use case interfaces for Handler - enables unit testing with mocks.

type createRuleUseCase interface {
	Execute(ctx context.Context, cmd usecases.CreateForwardRuleCommand) (*usecases.CreateForwardRuleResult, error)
}

type getRuleUseCase interface {
	Execute(ctx context.Context, query usecases.GetForwardRuleQuery) (*dto.ForwardRuleDTO, error)
}

type updateRuleUseCase interface {
	Execute(ctx context.Context, cmd usecases.UpdateForwardRuleCommand) error
}

type deleteRuleUseCase interface {
	Execute(ctx context.Context, cmd usecases.DeleteForwardRuleCommand) error
}

type listRulesUseCase interface {
	Execute(ctx context.Context, query usecases.ListForwardRulesQuery) (*usecases.ListForwardRulesResult, error)
}

type enableRuleUseCase interface {
	Execute(ctx context.Context, cmd usecases.EnableForwardRuleCommand) error
}

type disableRuleUseCase interface {
	Execute(ctx context.Context, cmd usecases.DisableForwardRuleCommand) error
}

type resetTrafficUseCase interface {
	Execute(ctx context.Context, cmd usecases.ResetForwardRuleTrafficCommand) error
}

type reorderRulesUseCase interface {
	Execute(ctx context.Context, cmd usecases.ReorderForwardRulesCommand) error
}

type batchRuleUseCase interface {
	BatchCreate(ctx context.Context, cmd usecases.BatchCreateCommand) (*dto.BatchCreateResponse, error)
	BatchDelete(ctx context.Context, cmd usecases.BatchDeleteCommand) (*dto.BatchOperationResult, error)
	BatchToggleStatus(ctx context.Context, cmd usecases.BatchToggleStatusCommand) (*dto.BatchOperationResult, error)
	BatchUpdate(ctx context.Context, cmd usecases.BatchUpdateCommand) (*dto.BatchOperationResult, error)
}

type probeService interface {
	ProbeRuleByShortID(ctx context.Context, shortID string, ipVersionOverride string) (*dto.RuleProbeResponse, error)
}
