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
	ID uint
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
	uc.logger.Infow("executing disable forward agent use case", "id", cmd.ID)

	if cmd.ID == 0 {
		return errors.NewValidationError("agent ID is required")
	}

	// Get the agent
	agent, err := uc.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return errors.NewNotFoundError("forward agent", fmt.Sprintf("%d", cmd.ID))
	}

	// Disable the agent
	if err := agent.Disable(); err != nil {
		uc.logger.Errorw("failed to disable forward agent", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to disable forward agent: %w", err)
	}

	// Persist changes
	if err := uc.repo.Update(ctx, agent); err != nil {
		uc.logger.Errorw("failed to update forward agent status", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to update forward agent: %w", err)
	}

	uc.logger.Infow("forward agent disabled successfully", "id", cmd.ID)
	return nil
}
