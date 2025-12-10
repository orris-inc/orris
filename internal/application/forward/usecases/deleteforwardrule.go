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
	ID      uint   // Internal database ID (deprecated, use ShortID for external API)
	ShortID string // External API identifier (without prefix)
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
	var rule *forward.ForwardRule
	var err error

	// Prefer ShortID over internal ID for external API
	if cmd.ShortID != "" {
		uc.logger.Infow("executing delete forward rule use case", "short_id", cmd.ShortID)
		rule, err = uc.repo.GetByShortID(ctx, cmd.ShortID)
		if err != nil {
			uc.logger.Errorw("failed to get forward rule", "short_id", cmd.ShortID, "error", err)
			return fmt.Errorf("failed to get forward rule: %w", err)
		}
		if rule == nil {
			return errors.NewNotFoundError("forward rule", cmd.ShortID)
		}
	} else if cmd.ID != 0 {
		uc.logger.Infow("executing delete forward rule use case", "id", cmd.ID)
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

	// Delete the rule using the internal ID
	if err := uc.repo.Delete(ctx, rule.ID()); err != nil {
		if cmd.ShortID != "" {
			uc.logger.Errorw("failed to delete forward rule", "short_id", cmd.ShortID, "error", err)
		} else {
			uc.logger.Errorw("failed to delete forward rule", "id", cmd.ID, "error", err)
		}
		return fmt.Errorf("failed to delete forward rule: %w", err)
	}

	if cmd.ShortID != "" {
		uc.logger.Infow("forward rule deleted successfully", "short_id", cmd.ShortID)
	} else {
		uc.logger.Infow("forward rule deleted successfully", "id", cmd.ID)
	}
	return nil
}
