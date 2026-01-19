package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/externalforward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// EnableExternalForwardRuleCommand represents the input for enabling an external forward rule.
type EnableExternalForwardRuleCommand struct {
	SID            string
	SubscriptionID uint
}

// EnableExternalForwardRuleUseCase handles enabling external forward rules.
type EnableExternalForwardRuleUseCase struct {
	repo   externalforward.Repository
	logger logger.Interface
}

// NewEnableExternalForwardRuleUseCase creates a new use case.
func NewEnableExternalForwardRuleUseCase(repo externalforward.Repository, logger logger.Interface) *EnableExternalForwardRuleUseCase {
	return &EnableExternalForwardRuleUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute enables an external forward rule.
func (uc *EnableExternalForwardRuleUseCase) Execute(ctx context.Context, cmd EnableExternalForwardRuleCommand) error {
	uc.logger.Infow("executing enable external forward rule use case", "sid", cmd.SID)

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

	rule.Enable()

	if err := uc.repo.Update(ctx, rule); err != nil {
		uc.logger.Errorw("failed to enable external forward rule", "sid", cmd.SID, "error", err)
		return fmt.Errorf("failed to enable external forward rule: %w", err)
	}

	uc.logger.Infow("external forward rule enabled successfully", "sid", cmd.SID)
	return nil
}

// DisableExternalForwardRuleCommand represents the input for disabling an external forward rule.
type DisableExternalForwardRuleCommand struct {
	SID            string
	SubscriptionID uint
}

// DisableExternalForwardRuleUseCase handles disabling external forward rules.
type DisableExternalForwardRuleUseCase struct {
	repo   externalforward.Repository
	logger logger.Interface
}

// NewDisableExternalForwardRuleUseCase creates a new use case.
func NewDisableExternalForwardRuleUseCase(repo externalforward.Repository, logger logger.Interface) *DisableExternalForwardRuleUseCase {
	return &DisableExternalForwardRuleUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute disables an external forward rule.
func (uc *DisableExternalForwardRuleUseCase) Execute(ctx context.Context, cmd DisableExternalForwardRuleCommand) error {
	uc.logger.Infow("executing disable external forward rule use case", "sid", cmd.SID)

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

	rule.Disable()

	if err := uc.repo.Update(ctx, rule); err != nil {
		uc.logger.Errorw("failed to disable external forward rule", "sid", cmd.SID, "error", err)
		return fmt.Errorf("failed to disable external forward rule: %w", err)
	}

	uc.logger.Infow("external forward rule disabled successfully", "sid", cmd.SID)
	return nil
}
