package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DeleteForwardAgentCommand represents the input for deleting a forward agent.
// Use either ID (internal) or ShortID (external API identifier).
type DeleteForwardAgentCommand struct {
	ID      uint   // Internal database ID (deprecated, use ShortID for external API)
	ShortID string // External API identifier (without prefix)
}

// DeleteForwardAgentUseCase handles forward agent deletion.
type DeleteForwardAgentUseCase struct {
	repo   forward.AgentRepository
	logger logger.Interface
}

// NewDeleteForwardAgentUseCase creates a new DeleteForwardAgentUseCase.
func NewDeleteForwardAgentUseCase(
	repo forward.AgentRepository,
	logger logger.Interface,
) *DeleteForwardAgentUseCase {
	return &DeleteForwardAgentUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute deletes a forward agent.
func (uc *DeleteForwardAgentUseCase) Execute(ctx context.Context, cmd DeleteForwardAgentCommand) error {
	var agent *forward.ForwardAgent
	var err error

	// Prefer ShortID over internal ID for external API
	if cmd.ShortID != "" {
		uc.logger.Infow("executing delete forward agent use case", "short_id", cmd.ShortID)
		agent, err = uc.repo.GetByShortID(ctx, cmd.ShortID)
		if err != nil {
			uc.logger.Errorw("failed to get forward agent", "short_id", cmd.ShortID, "error", err)
			return fmt.Errorf("failed to get forward agent: %w", err)
		}
		if agent == nil {
			return errors.NewNotFoundError("forward agent", cmd.ShortID)
		}
	} else if cmd.ID != 0 {
		uc.logger.Infow("executing delete forward agent use case", "id", cmd.ID)
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

	// Delete the agent using internal ID
	if err := uc.repo.Delete(ctx, agent.ID()); err != nil {
		uc.logger.Errorw("failed to delete forward agent", "id", agent.ID(), "short_id", agent.ShortID(), "error", err)
		return fmt.Errorf("failed to delete forward agent: %w", err)
	}

	uc.logger.Infow("forward agent deleted successfully", "id", agent.ID(), "short_id", agent.ShortID())
	return nil
}
