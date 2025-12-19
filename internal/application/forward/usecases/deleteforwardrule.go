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
	ShortID string // External API identifier
}

// DeleteForwardRuleUseCase handles forward rule deletion.
type DeleteForwardRuleUseCase struct {
	repo          forward.Repository
	configSyncSvc ConfigSyncNotifier
	logger        logger.Interface
}

// NewDeleteForwardRuleUseCase creates a new DeleteForwardRuleUseCase.
func NewDeleteForwardRuleUseCase(
	repo forward.Repository,
	configSyncSvc ConfigSyncNotifier,
	logger logger.Interface,
) *DeleteForwardRuleUseCase {
	return &DeleteForwardRuleUseCase{
		repo:          repo,
		configSyncSvc: configSyncSvc,
		logger:        logger,
	}
}

// Execute deletes a forward rule.
func (uc *DeleteForwardRuleUseCase) Execute(ctx context.Context, cmd DeleteForwardRuleCommand) error {
	if cmd.ShortID == "" {
		return errors.NewValidationError("short_id is required")
	}

	uc.logger.Infow("executing delete forward rule use case", "short_id", cmd.ShortID)
	rule, err := uc.repo.GetBySID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward rule", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to get forward rule: %w", err)
	}
	if rule == nil {
		return errors.NewNotFoundError("forward rule", cmd.ShortID)
	}

	// Store info for notification before deletion
	agentID := rule.AgentID()
	ruleShortID := rule.SID()
	wasEnabled := rule.IsEnabled()

	// Delete the rule using the internal ID
	if err := uc.repo.Delete(ctx, rule.ID()); err != nil {
		uc.logger.Errorw("failed to delete forward rule", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to delete forward rule: %w", err)
	}

	uc.logger.Infow("forward rule deleted successfully", "short_id", cmd.ShortID)

	// Notify config sync asynchronously if rule was enabled (failure only logs warning, doesn't block)
	if wasEnabled && uc.configSyncSvc != nil {
		go func() {
			if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), agentID, ruleShortID, "removed"); err != nil {
				uc.logger.Warnw("failed to notify config sync", "rule_id", ruleShortID, "error", err)
			}
		}()
	}

	return nil
}
