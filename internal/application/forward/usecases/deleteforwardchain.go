package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DeleteForwardChainUseCase handles forward chain deletion.
type DeleteForwardChainUseCase struct {
	chainRepo forward.ChainRepository
	ruleRepo  forward.Repository
	logger    logger.Interface
}

// NewDeleteForwardChainUseCase creates a new DeleteForwardChainUseCase.
func NewDeleteForwardChainUseCase(
	chainRepo forward.ChainRepository,
	ruleRepo forward.Repository,
	logger logger.Interface,
) *DeleteForwardChainUseCase {
	return &DeleteForwardChainUseCase{
		chainRepo: chainRepo,
		ruleRepo:  ruleRepo,
		logger:    logger,
	}
}

// Execute deletes a forward chain and its associated rules.
func (uc *DeleteForwardChainUseCase) Execute(ctx context.Context, id uint) error {
	uc.logger.Infow("executing delete forward chain use case", "id", id)

	if id == 0 {
		return errors.NewValidationError("chain ID is required")
	}

	// Get chain to verify it exists
	chain, err := uc.chainRepo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get forward chain", "id", id, "error", err)
		return err
	}
	if chain == nil {
		return errors.NewNotFoundError("forward chain", fmt.Sprintf("%d", id))
	}

	// Get associated rule IDs
	ruleIDs, err := uc.chainRepo.GetRuleIDsByChainID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get rule IDs for chain", "chain_id", id, "error", err)
		return fmt.Errorf("failed to get associated rules: %w", err)
	}

	// Delete associated rules
	for _, ruleID := range ruleIDs {
		if err := uc.ruleRepo.Delete(ctx, ruleID); err != nil {
			uc.logger.Warnw("failed to delete rule", "rule_id", ruleID, "error", err)
			// Continue deleting other rules
		}
	}

	// Delete chain
	if err := uc.chainRepo.Delete(ctx, id); err != nil {
		uc.logger.Errorw("failed to delete forward chain", "id", id, "error", err)
		return err
	}

	uc.logger.Infow("forward chain deleted successfully", "id", id, "rules_deleted", len(ruleIDs))
	return nil
}
