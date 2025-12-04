package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// EnableForwardChainUseCase handles enabling a forward chain and its rules.
type EnableForwardChainUseCase struct {
	chainRepo forward.ChainRepository
	ruleRepo  forward.Repository
	txManager *db.TransactionManager
	logger    logger.Interface
}

// NewEnableForwardChainUseCase creates a new EnableForwardChainUseCase.
func NewEnableForwardChainUseCase(
	chainRepo forward.ChainRepository,
	ruleRepo forward.Repository,
	txManager *db.TransactionManager,
	logger logger.Interface,
) *EnableForwardChainUseCase {
	return &EnableForwardChainUseCase{
		chainRepo: chainRepo,
		ruleRepo:  ruleRepo,
		txManager: txManager,
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

	// Execute all operations within a transaction
	err = uc.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Enable chain
		chain.Enable()
		if err := uc.chainRepo.Update(txCtx, chain); err != nil {
			uc.logger.Errorw("failed to enable forward chain", "id", id, "error", err)
			return err
		}

		// Enable all associated rules
		ruleIDs, err := uc.chainRepo.GetRuleIDsByChainID(txCtx, id)
		if err != nil {
			uc.logger.Errorw("failed to get rule IDs for chain", "chain_id", id, "error", err)
			return fmt.Errorf("failed to get associated rules: %w", err)
		}

		for _, ruleID := range ruleIDs {
			rule, err := uc.ruleRepo.GetByID(txCtx, ruleID)
			if err != nil {
				uc.logger.Errorw("failed to get rule", "rule_id", ruleID, "error", err)
				return fmt.Errorf("failed to get rule %d: %w", ruleID, err)
			}
			if rule == nil {
				uc.logger.Errorw("rule not found", "rule_id", ruleID)
				return fmt.Errorf("rule %d not found", ruleID)
			}

			if err := rule.Enable(); err != nil {
				uc.logger.Errorw("failed to enable rule", "rule_id", ruleID, "error", err)
				return fmt.Errorf("failed to enable rule %d: %w", ruleID, err)
			}

			if err := uc.ruleRepo.Update(txCtx, rule); err != nil {
				uc.logger.Errorw("failed to update rule", "rule_id", ruleID, "error", err)
				return fmt.Errorf("failed to update rule %d: %w", ruleID, err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	uc.logger.Infow("forward chain enabled successfully", "id", id)
	return nil
}
