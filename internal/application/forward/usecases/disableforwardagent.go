package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DisableForwardAgentCommand represents the input for disabling a forward agent.
// Use either ID (internal) or ShortID (external API identifier).
type DisableForwardAgentCommand struct {
	ID      uint   // Internal database ID (deprecated, use ShortID for external API)
	ShortID string // External API identifier (without prefix)
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
	var agent *forward.ForwardAgent
	var err error

	// Prefer ShortID over internal ID for external API
	if cmd.ShortID != "" {
		uc.logger.Infow("executing disable forward agent use case", "short_id", cmd.ShortID)
		agent, err = uc.repo.GetByShortID(ctx, cmd.ShortID)
		if err != nil {
			uc.logger.Errorw("failed to get forward agent", "short_id", cmd.ShortID, "error", err)
			return fmt.Errorf("failed to get forward agent: %w", err)
		}
		if agent == nil {
			return errors.NewNotFoundError("forward agent", cmd.ShortID)
		}
	} else if cmd.ID != 0 {
		uc.logger.Infow("executing disable forward agent use case", "id", cmd.ID)
		agent, err = uc.repo.GetByID(ctx, cmd.ID)
		if err != nil {
			uc.logger.Errorw("failed to get forward agent", "id", cmd.ID, "error", err)
			return fmt.Errorf("failed to get forward agent: %w", err)
		}
		if agent == nil {
			return errors.NewNotFoundError("forward agent", fmt.Sprintf("%d", cmd.ID))
		}
	} else {
		return errors.NewValidationError("agent ID or short_id is required")
	}

	// Disable the agent
	if err := agent.Disable(); err != nil {
		uc.logger.Errorw("failed to disable forward agent", "id", agent.ID(), "short_id", agent.ShortID(), "error", err)
		return fmt.Errorf("failed to disable forward agent: %w", err)
	}

	// Persist changes
	if err := uc.repo.Update(ctx, agent); err != nil {
		uc.logger.Errorw("failed to update forward agent status", "id", agent.ID(), "short_id", agent.ShortID(), "error", err)
		return fmt.Errorf("failed to update forward agent: %w", err)
	}

	uc.logger.Infow("forward agent disabled successfully", "id", agent.ID(), "short_id", agent.ShortID())
	return nil
}
