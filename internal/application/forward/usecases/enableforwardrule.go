package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// EnableForwardRuleCommand represents the input for enabling a forward rule.
type EnableForwardRuleCommand struct {
	ID      uint   // Internal database ID (deprecated, use ShortID for external API)
	ShortID string // External API identifier (without prefix)
}

// EnableForwardRuleUseCase handles enabling a forward rule.
type EnableForwardRuleUseCase struct {
	repo   forward.Repository
	logger logger.Interface
}

// NewEnableForwardRuleUseCase creates a new EnableForwardRuleUseCase.
func NewEnableForwardRuleUseCase(
	repo forward.Repository,
	logger logger.Interface,
) *EnableForwardRuleUseCase {
	return &EnableForwardRuleUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute enables a forward rule.
func (uc *EnableForwardRuleUseCase) Execute(ctx context.Context, cmd EnableForwardRuleCommand) error {
	var rule *forward.ForwardRule
	var err error

	// Prefer ShortID over internal ID for external API
	if cmd.ShortID != "" {
		uc.logger.Infow("executing enable forward rule use case", "short_id", cmd.ShortID)
		rule, err = uc.repo.GetByShortID(ctx, cmd.ShortID)
		if err != nil {
			uc.logger.Errorw("failed to get forward rule", "short_id", cmd.ShortID, "error", err)
			return fmt.Errorf("failed to get forward rule: %w", err)
		}
		if rule == nil {
			return errors.NewNotFoundError("forward rule", cmd.ShortID)
		}
	} else if cmd.ID != 0 {
		uc.logger.Infow("executing enable forward rule use case", "id", cmd.ID)
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

	if err := rule.Enable(); err != nil {
		return errors.NewValidationError(err.Error())
	}

	if err := uc.repo.Update(ctx, rule); err != nil {
		if cmd.ShortID != "" {
			uc.logger.Errorw("failed to enable forward rule", "short_id", cmd.ShortID, "error", err)
		} else {
			uc.logger.Errorw("failed to enable forward rule", "id", cmd.ID, "error", err)
		}
		return fmt.Errorf("failed to enable forward rule: %w", err)
	}

	if cmd.ShortID != "" {
		uc.logger.Infow("forward rule enabled successfully", "short_id", cmd.ShortID)
	} else {
		uc.logger.Infow("forward rule enabled successfully", "id", cmd.ID)
	}
	return nil
}
