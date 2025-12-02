package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/forward"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// EnableForwardChainUseCase handles enabling a forward chain and its rules.
type EnableForwardChainUseCase struct {
	chainRepo forward.ChainRepository
	ruleRepo  forward.Repository
	logger    logger.Interface
}

// NewEnableForwardChainUseCase creates a new EnableForwardChainUseCase.
func NewEnableForwardChainUseCase(
	chainRepo forward.ChainRepository,
	ruleRepo forward.Repository,
	logger logger.Interface,
) *EnableForwardChainUseCase {
	return &EnableForwardChainUseCase{
		chainRepo: chainRepo,
		ruleRepo:  ruleRepo,
		logger:    logger,
	}
}

// Execute enables a forward chain and all its associated rules.
func (uc *EnableForwardChainUseCase) Execute(ctx context.Context, id uint) error {
	uc.logger.Infow("executing enable forward chain use case", "id", id)

	if id == 0 {
		return errors.NewValidationError("chain ID is required")
	}

	chain, err := uc.chainRepo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get forward chain", "id", id, "error", err)
		return err
	}
	if chain == nil {
		return errors.NewNotFoundError("forward chain", fmt.Sprintf("%d", id))
	}

	// Enable chain
	chain.Enable()
	if err := uc.chainRepo.Update(ctx, chain); err != nil {
		uc.logger.Errorw("failed to enable forward chain", "id", id, "error", err)
		return err
	}

	// Enable all associated rules
	ruleIDs, err := uc.chainRepo.GetRuleIDsByChainID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get rule IDs for chain", "chain_id", id, "error", err)
		return fmt.Errorf("failed to get associated rules: %w", err)
	}

	for _, ruleID := range ruleIDs {
		rule, err := uc.ruleRepo.GetByID(ctx, ruleID)
		if err != nil {
			uc.logger.Warnw("failed to get rule", "rule_id", ruleID, "error", err)
			continue
		}
		if rule == nil {
			continue
		}

		if err := rule.Enable(); err != nil {
			uc.logger.Warnw("failed to enable rule", "rule_id", ruleID, "error", err)
			continue
		}

		if err := uc.ruleRepo.Update(ctx, rule); err != nil {
			uc.logger.Warnw("failed to update rule", "rule_id", ruleID, "error", err)
		}
	}

	uc.logger.Infow("forward chain enabled successfully", "id", id, "rules_enabled", len(ruleIDs))
	return nil
}
