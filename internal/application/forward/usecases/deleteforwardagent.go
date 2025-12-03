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
	ID uint
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
	uc.logger.Infow("executing delete forward agent use case", "id", cmd.ID)

	if cmd.ID == 0 {
		return errors.NewValidationError("agent ID is required")
	}

	// Check if agent exists
	agent, err := uc.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return errors.NewNotFoundError("forward agent", fmt.Sprintf("%d", cmd.ID))
	}

	// Delete the agent
	if err := uc.repo.Delete(ctx, cmd.ID); err != nil {
		uc.logger.Errorw("failed to delete forward agent", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to delete forward agent: %w", err)
	}

	uc.logger.Infow("forward agent deleted successfully", "id", cmd.ID)
	return nil
}
