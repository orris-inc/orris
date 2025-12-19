package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DisableForwardAgentCommand represents the input for disabling a forward agent.
type DisableForwardAgentCommand struct {
	ShortID string // External API identifier
}

// DisableForwardAgentUseCase handles forward agent disabling.
type DisableForwardAgentUseCase struct {
	repo   forward.AgentRepository
	logger logger.Interface
}

// NewDisableForwardAgentUseCase creates a new DisableForwardAgentUseCase.
func NewDisableForwardAgentUseCase(
	repo forward.AgentRepository,
	logger logger.Interface,
) *DisableForwardAgentUseCase {
	return &DisableForwardAgentUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute disables a forward agent.
func (uc *DisableForwardAgentUseCase) Execute(ctx context.Context, cmd DisableForwardAgentCommand) error {
	if cmd.ShortID == "" {
		return errors.NewValidationError("short_id is required")
	}

	uc.logger.Infow("executing disable forward agent use case", "short_id", cmd.ShortID)

	agent, err := uc.repo.GetBySID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return errors.NewNotFoundError("forward agent", cmd.ShortID)
	}

	// Disable the agent
	if err := agent.Disable(); err != nil {
		uc.logger.Errorw("failed to disable forward agent", "id", agent.ID(), "short_id", agent.SID(), "error", err)
		return fmt.Errorf("failed to disable forward agent: %w", err)
	}

	// Persist changes
	if err := uc.repo.Update(ctx, agent); err != nil {
		uc.logger.Errorw("failed to update forward agent status", "id", agent.ID(), "short_id", agent.SID(), "error", err)
		return fmt.Errorf("failed to update forward agent: %w", err)
	}

	uc.logger.Infow("forward agent disabled successfully", "id", agent.ID(), "short_id", agent.SID())
	return nil
}
