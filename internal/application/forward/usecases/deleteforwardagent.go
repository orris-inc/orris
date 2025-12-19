package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DeleteForwardAgentCommand represents the input for deleting a forward agent.
type DeleteForwardAgentCommand struct {
	ShortID string // External API identifier
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
	if cmd.ShortID == "" {
		return errors.NewValidationError("short_id is required")
	}

	uc.logger.Infow("executing delete forward agent use case", "short_id", cmd.ShortID)

	agent, err := uc.repo.GetBySID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return errors.NewNotFoundError("forward agent", cmd.ShortID)
	}

	// Delete the agent using internal ID
	if err := uc.repo.Delete(ctx, agent.ID()); err != nil {
		uc.logger.Errorw("failed to delete forward agent", "id", agent.ID(), "short_id", agent.SID(), "error", err)
		return fmt.Errorf("failed to delete forward agent: %w", err)
	}

	uc.logger.Infow("forward agent deleted successfully", "id", agent.ID(), "short_id", agent.SID())
	return nil
}
