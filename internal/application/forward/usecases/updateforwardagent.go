package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/forward"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// UpdateForwardAgentCommand represents the input for updating a forward agent.
type UpdateForwardAgentCommand struct {
	ID     uint
	Name   *string
	Remark *string
}

// UpdateForwardAgentUseCase handles forward agent updates.
type UpdateForwardAgentUseCase struct {
	repo   forward.AgentRepository
	logger logger.Interface
}

// NewUpdateForwardAgentUseCase creates a new UpdateForwardAgentUseCase.
func NewUpdateForwardAgentUseCase(
	repo forward.AgentRepository,
	logger logger.Interface,
) *UpdateForwardAgentUseCase {
	return &UpdateForwardAgentUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute updates an existing forward agent.
func (uc *UpdateForwardAgentUseCase) Execute(ctx context.Context, cmd UpdateForwardAgentCommand) error {
	uc.logger.Infow("executing update forward agent use case", "id", cmd.ID)

	if cmd.ID == 0 {
		return errors.NewValidationError("agent ID is required")
	}

	// Get existing agent
	agent, err := uc.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return errors.NewNotFoundError("forward agent", fmt.Sprintf("%d", cmd.ID))
	}

	// Update fields
	if cmd.Name != nil {
		if err := agent.UpdateName(*cmd.Name); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.Remark != nil {
		if err := agent.UpdateRemark(*cmd.Remark); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Persist changes
	if err := uc.repo.Update(ctx, agent); err != nil {
		uc.logger.Errorw("failed to update forward agent", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to update forward agent: %w", err)
	}

	uc.logger.Infow("forward agent updated successfully", "id", cmd.ID)
	return nil
}
