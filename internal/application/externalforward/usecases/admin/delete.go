package admin

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/externalforward"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AdminDeleteExternalForwardRuleCommand represents the input for deleting an external forward rule.
type AdminDeleteExternalForwardRuleCommand struct {
	SID string
}

// AdminDeleteExternalForwardRuleUseCase handles admin external forward rule deletion.
type AdminDeleteExternalForwardRuleUseCase struct {
	repo   externalforward.Repository
	logger logger.Interface
}

// NewAdminDeleteExternalForwardRuleUseCase creates a new admin delete use case.
func NewAdminDeleteExternalForwardRuleUseCase(repo externalforward.Repository, logger logger.Interface) *AdminDeleteExternalForwardRuleUseCase {
	return &AdminDeleteExternalForwardRuleUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute deletes an external forward rule.
func (uc *AdminDeleteExternalForwardRuleUseCase) Execute(ctx context.Context, cmd AdminDeleteExternalForwardRuleCommand) error {
	uc.logger.Infow("executing admin delete external forward rule use case", "sid", cmd.SID)

	// Get existing rule to verify it exists
	rule, err := uc.repo.GetBySID(ctx, cmd.SID)
	if err != nil {
		return err
	}

	// Delete
	if err := uc.repo.Delete(ctx, rule.ID()); err != nil {
		uc.logger.Errorw("failed to delete external forward rule", "sid", cmd.SID, "error", err)
		return fmt.Errorf("failed to delete external forward rule: %w", err)
	}

	uc.logger.Infow("external forward rule deleted successfully by admin", "sid", cmd.SID)
	return nil
}
