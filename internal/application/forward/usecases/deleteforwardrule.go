package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DeleteForwardRuleCommand represents the input for deleting a forward rule.
type DeleteForwardRuleCommand struct {
	ID uint
}

// DeleteForwardRuleUseCase handles forward rule deletion.
type DeleteForwardRuleUseCase struct {
	repo   forward.Repository
	logger logger.Interface
}

// NewDeleteForwardRuleUseCase creates a new DeleteForwardRuleUseCase.
func NewDeleteForwardRuleUseCase(
	repo forward.Repository,
	logger logger.Interface,
) *DeleteForwardRuleUseCase {
	return &DeleteForwardRuleUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute deletes a forward rule.
func (uc *DeleteForwardRuleUseCase) Execute(ctx context.Context, cmd DeleteForwardRuleCommand) error {
	uc.logger.Infow("executing delete forward rule use case", "id", cmd.ID)

	if cmd.ID == 0 {
		return errors.NewValidationError("rule ID is required")
	}

	// Check if rule exists
	rule, err := uc.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		uc.logger.Errorw("failed to get forward rule", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to get forward rule: %w", err)
	}
	if rule == nil {
		return errors.NewNotFoundError("forward rule", fmt.Sprintf("%d", cmd.ID))
	}

	// Delete the rule
	if err := uc.repo.Delete(ctx, cmd.ID); err != nil {
		uc.logger.Errorw("failed to delete forward rule", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to delete forward rule: %w", err)
	}

	uc.logger.Infow("forward rule deleted successfully", "id", cmd.ID)
	return nil
}
