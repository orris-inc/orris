package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ResetForwardRuleTrafficCommand represents the input for resetting traffic counters.
type ResetForwardRuleTrafficCommand struct {
	ID      uint   // Internal database ID (deprecated, use ShortID for external API)
	ShortID string // External API identifier (without prefix)
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
	var rule *forward.ForwardRule
	var err error

	// Prefer ShortID over internal ID for external API
	if cmd.ShortID != "" {
		uc.logger.Infow("executing reset forward rule traffic use case", "short_id", cmd.ShortID)
		rule, err = uc.repo.GetByShortID(ctx, cmd.ShortID)
		if err != nil {
			uc.logger.Errorw("failed to get forward rule", "short_id", cmd.ShortID, "error", err)
			return fmt.Errorf("failed to get forward rule: %w", err)
		}
		if rule == nil {
			return errors.NewNotFoundError("forward rule", cmd.ShortID)
		}
	} else if cmd.ID != 0 {
		uc.logger.Infow("executing reset forward rule traffic use case", "id", cmd.ID)
		rule, err = uc.repo.GetByID(ctx, cmd.ID)
		if err != nil {
			uc.logger.Errorw("failed to get forward rule", "id", cmd.ID, "error", err)
			return fmt.Errorf("failed to get forward rule: %w", err)
		}
		if rule == nil {
			return errors.NewNotFoundError("forward rule", fmt.Sprintf("%d", cmd.ID))
		}
	} else {
		return errors.NewValidationError("rule ID or short_id is required")
	}

	rule.ResetTraffic()

	if err := uc.repo.Update(ctx, rule); err != nil {
		if cmd.ShortID != "" {
			uc.logger.Errorw("failed to reset forward rule traffic", "short_id", cmd.ShortID, "error", err)
		} else {
			uc.logger.Errorw("failed to reset forward rule traffic", "id", cmd.ID, "error", err)
		}
		return fmt.Errorf("failed to reset forward rule traffic: %w", err)
	}

	if cmd.ShortID != "" {
		uc.logger.Infow("forward rule traffic reset successfully", "short_id", cmd.ShortID)
	} else {
		uc.logger.Infow("forward rule traffic reset successfully", "id", cmd.ID)
	}
	return nil
}
