package admin

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/externalforward"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AdminEnableExternalForwardRuleCommand represents the input for enabling an external forward rule.
type AdminEnableExternalForwardRuleCommand struct {
	SID string
}

// AdminEnableExternalForwardRuleUseCase handles admin enabling external forward rules.
type AdminEnableExternalForwardRuleUseCase struct {
	repo   externalforward.Repository
	logger logger.Interface
}

// NewAdminEnableExternalForwardRuleUseCase creates a new admin enable use case.
func NewAdminEnableExternalForwardRuleUseCase(repo externalforward.Repository, logger logger.Interface) *AdminEnableExternalForwardRuleUseCase {
	return &AdminEnableExternalForwardRuleUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute enables an external forward rule.
func (uc *AdminEnableExternalForwardRuleUseCase) Execute(ctx context.Context, cmd AdminEnableExternalForwardRuleCommand) error {
	uc.logger.Infow("executing admin enable external forward rule use case", "sid", cmd.SID)

	rule, err := uc.repo.GetBySID(ctx, cmd.SID)
	if err != nil {
		return err
	}

	rule.Enable()

	if err := uc.repo.Update(ctx, rule); err != nil {
		uc.logger.Errorw("failed to enable external forward rule", "sid", cmd.SID, "error", err)
		return fmt.Errorf("failed to enable external forward rule: %w", err)
	}

	uc.logger.Infow("external forward rule enabled successfully by admin", "sid", cmd.SID)
	return nil
}

// AdminDisableExternalForwardRuleCommand represents the input for disabling an external forward rule.
type AdminDisableExternalForwardRuleCommand struct {
	SID string
}

// AdminDisableExternalForwardRuleUseCase handles admin disabling external forward rules.
type AdminDisableExternalForwardRuleUseCase struct {
	repo   externalforward.Repository
	logger logger.Interface
}

// NewAdminDisableExternalForwardRuleUseCase creates a new admin disable use case.
func NewAdminDisableExternalForwardRuleUseCase(repo externalforward.Repository, logger logger.Interface) *AdminDisableExternalForwardRuleUseCase {
	return &AdminDisableExternalForwardRuleUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute disables an external forward rule.
func (uc *AdminDisableExternalForwardRuleUseCase) Execute(ctx context.Context, cmd AdminDisableExternalForwardRuleCommand) error {
	uc.logger.Infow("executing admin disable external forward rule use case", "sid", cmd.SID)

	rule, err := uc.repo.GetBySID(ctx, cmd.SID)
	if err != nil {
		return err
	}

	rule.Disable()

	if err := uc.repo.Update(ctx, rule); err != nil {
		uc.logger.Errorw("failed to disable external forward rule", "sid", cmd.SID, "error", err)
		return fmt.Errorf("failed to disable external forward rule: %w", err)
	}

	uc.logger.Infow("external forward rule disabled successfully by admin", "sid", cmd.SID)
	return nil
}
