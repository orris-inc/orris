package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/forward"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// DisableForwardChainUseCase handles disabling a forward chain and its rules.
type DisableForwardChainUseCase struct {
	chainRepo forward.ChainRepository
	ruleRepo  forward.Repository
	logger    logger.Interface
}

// NewDisableForwardChainUseCase creates a new DisableForwardChainUseCase.
func NewDisableForwardChainUseCase(
	chainRepo forward.ChainRepository,
	ruleRepo forward.Repository,
	logger logger.Interface,
) *DisableForwardChainUseCase {
	return &DisableForwardChainUseCase{
		chainRepo: chainRepo,
		ruleRepo:  ruleRepo,
		logger:    logger,
	}
}

// Execute disables a forward chain and all its associated rules.
func (uc *DisableForwardChainUseCase) Execute(ctx context.Context, id uint) error {
	uc.logger.Infow("executing disable forward chain use case", "id", id)

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

	// Disable chain
	chain.Disable()
	if err := uc.chainRepo.Update(ctx, chain); err != nil {
		uc.logger.Errorw("failed to disable forward chain", "id", id, "error", err)
		return err
	}

	// Disable all associated rules
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

		if err := rule.Disable(); err != nil {
			uc.logger.Warnw("failed to disable rule", "rule_id", ruleID, "error", err)
			continue
		}

		if err := uc.ruleRepo.Update(ctx, rule); err != nil {
			uc.logger.Warnw("failed to update rule", "rule_id", ruleID, "error", err)
		}
	}

	uc.logger.Infow("forward chain disabled successfully", "id", id, "rules_disabled", len(ruleIDs))
	return nil
}
