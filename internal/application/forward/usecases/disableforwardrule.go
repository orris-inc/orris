package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/forward"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// DisableForwardRuleCommand represents the input for disabling a forward rule.
type DisableForwardRuleCommand struct {
	ID uint
}

// DisableForwardRuleUseCase handles disabling a forward rule.
type DisableForwardRuleUseCase struct {
	repo   forward.Repository
	logger logger.Interface
}

// NewDisableForwardRuleUseCase creates a new DisableForwardRuleUseCase.
func NewDisableForwardRuleUseCase(
	repo forward.Repository,
	logger logger.Interface,
) *DisableForwardRuleUseCase {
	return &DisableForwardRuleUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute disables a forward rule.
func (uc *DisableForwardRuleUseCase) Execute(ctx context.Context, cmd DisableForwardRuleCommand) error {
	uc.logger.Infow("executing disable forward rule use case", "id", cmd.ID)

	if cmd.ID == 0 {
		return errors.NewValidationError("rule ID is required")
	}

	rule, err := uc.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		uc.logger.Errorw("failed to get forward rule", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to get forward rule: %w", err)
	}
	if rule == nil {
		return errors.NewNotFoundError("forward rule", fmt.Sprintf("%d", cmd.ID))
	}

	if err := rule.Disable(); err != nil {
		return errors.NewValidationError(err.Error())
	}

	if err := uc.repo.Update(ctx, rule); err != nil {
		uc.logger.Errorw("failed to disable forward rule", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to disable forward rule: %w", err)
	}

	uc.logger.Infow("forward rule disabled successfully", "id", cmd.ID)
	return nil
}
