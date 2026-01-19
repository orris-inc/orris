package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/externalforward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DeleteExternalForwardRuleCommand represents the input for deleting an external forward rule.
type DeleteExternalForwardRuleCommand struct {
	SID            string
	SubscriptionID uint
}

// DeleteExternalForwardRuleUseCase handles external forward rule deletion.
type DeleteExternalForwardRuleUseCase struct {
	repo   externalforward.Repository
	logger logger.Interface
}

// NewDeleteExternalForwardRuleUseCase creates a new use case.
func NewDeleteExternalForwardRuleUseCase(repo externalforward.Repository, logger logger.Interface) *DeleteExternalForwardRuleUseCase {
	return &DeleteExternalForwardRuleUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute deletes an external forward rule.
func (uc *DeleteExternalForwardRuleUseCase) Execute(ctx context.Context, cmd DeleteExternalForwardRuleCommand) error {
	uc.logger.Infow("executing delete external forward rule use case", "sid", cmd.SID)

	// Get existing rule to verify it exists
	rule, err := uc.repo.GetBySID(ctx, cmd.SID)
	if err != nil {
		return err
	}

	// Verify rule belongs to the specified subscription
	if rule.SubscriptionID() == nil || *rule.SubscriptionID() != cmd.SubscriptionID {
		uc.logger.Warnw("external forward rule does not belong to subscription",
			"rule_sid", cmd.SID,
			"rule_subscription_id", rule.SubscriptionID(),
			"requested_subscription_id", cmd.SubscriptionID,
		)
		return errors.NewNotFoundError("external forward rule", cmd.SID)
	}

	// Delete
	if err := uc.repo.Delete(ctx, rule.ID()); err != nil {
		uc.logger.Errorw("failed to delete external forward rule", "sid", cmd.SID, "error", err)
		return fmt.Errorf("failed to delete external forward rule: %w", err)
	}

	uc.logger.Infow("external forward rule deleted successfully", "sid", cmd.SID)
	return nil
}
