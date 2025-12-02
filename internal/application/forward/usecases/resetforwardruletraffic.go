package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/forward"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// ResetForwardRuleTrafficCommand represents the input for resetting traffic counters.
type ResetForwardRuleTrafficCommand struct {
	ID uint
}

// ResetForwardRuleTrafficUseCase handles resetting forward rule traffic counters.
type ResetForwardRuleTrafficUseCase struct {
	repo   forward.Repository
	logger logger.Interface
}

// NewResetForwardRuleTrafficUseCase creates a new ResetForwardRuleTrafficUseCase.
func NewResetForwardRuleTrafficUseCase(
	repo forward.Repository,
	logger logger.Interface,
) *ResetForwardRuleTrafficUseCase {
	return &ResetForwardRuleTrafficUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute resets the traffic counters for a forward rule.
func (uc *ResetForwardRuleTrafficUseCase) Execute(ctx context.Context, cmd ResetForwardRuleTrafficCommand) error {
	uc.logger.Infow("executing reset forward rule traffic use case", "id", cmd.ID)

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

	rule.ResetTraffic()

	if err := uc.repo.Update(ctx, rule); err != nil {
		uc.logger.Errorw("failed to reset forward rule traffic", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to reset forward rule traffic: %w", err)
	}

	uc.logger.Infow("forward rule traffic reset successfully", "id", cmd.ID)
	return nil
}
