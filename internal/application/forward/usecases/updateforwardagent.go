package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdateForwardAgentCommand represents the input for updating a forward agent.
type UpdateForwardAgentCommand struct {
	ShortID       string // External API identifier
	Name          *string
	PublicAddress *string
	Remark        *string
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
	if cmd.ShortID == "" {
		return errors.NewValidationError("short_id is required")
	}

	uc.logger.Infow("executing update forward agent use case", "short_id", cmd.ShortID)

	agent, err := uc.repo.GetByShortID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return errors.NewNotFoundError("forward agent", cmd.ShortID)
	}

	// Update fields
	if cmd.Name != nil {
		if err := agent.UpdateName(*cmd.Name); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.PublicAddress != nil {
		if err := agent.UpdatePublicAddress(*cmd.PublicAddress); err != nil {
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
		uc.logger.Errorw("failed to update forward agent", "id", agent.ID(), "short_id", agent.ShortID(), "error", err)
		return fmt.Errorf("failed to update forward agent: %w", err)
	}

	uc.logger.Infow("forward agent updated successfully", "id", agent.ID(), "short_id", agent.ShortID())
	return nil
}
