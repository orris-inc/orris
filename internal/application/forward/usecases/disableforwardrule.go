package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DisableForwardRuleCommand represents the input for disabling a forward rule.
type DisableForwardRuleCommand struct {
	ID      uint   // Internal database ID (deprecated, use ShortID for external API)
	ShortID string // External API identifier (without prefix)
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
	var rule *forward.ForwardRule
	var err error

	// Prefer ShortID over internal ID for external API
	if cmd.ShortID != "" {
		uc.logger.Infow("executing disable forward rule use case", "short_id", cmd.ShortID)
		rule, err = uc.repo.GetByShortID(ctx, cmd.ShortID)
		if err != nil {
			uc.logger.Errorw("failed to get forward rule", "short_id", cmd.ShortID, "error", err)
			return fmt.Errorf("failed to get forward rule: %w", err)
		}
		if rule == nil {
			return errors.NewNotFoundError("forward rule", cmd.ShortID)
		}
	} else if cmd.ID != 0 {
		uc.logger.Infow("executing disable forward rule use case", "id", cmd.ID)
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

	if err := rule.Disable(); err != nil {
		return errors.NewValidationError(err.Error())
	}

	if err := uc.repo.Update(ctx, rule); err != nil {
		if cmd.ShortID != "" {
			uc.logger.Errorw("failed to disable forward rule", "short_id", cmd.ShortID, "error", err)
		} else {
			uc.logger.Errorw("failed to disable forward rule", "id", cmd.ID, "error", err)
		}
		return fmt.Errorf("failed to disable forward rule: %w", err)
	}

	if cmd.ShortID != "" {
		uc.logger.Infow("forward rule disabled successfully", "short_id", cmd.ShortID)
	} else {
		uc.logger.Infow("forward rule disabled successfully", "id", cmd.ID)
	}
	return nil
}
