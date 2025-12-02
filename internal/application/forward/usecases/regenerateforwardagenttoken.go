package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/forward"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// RegenerateForwardAgentTokenCommand represents the input for regenerating an agent token.
type RegenerateForwardAgentTokenCommand struct {
	ID uint
}

// RegenerateForwardAgentTokenResult represents the output of regenerating an agent token.
type RegenerateForwardAgentTokenResult struct {
	ID    uint   `json:"id"`
	Token string `json:"token"`
}

// RegenerateForwardAgentTokenUseCase handles forward agent token regeneration.
type RegenerateForwardAgentTokenUseCase struct {
	repo   forward.AgentRepository
	logger logger.Interface
}

// NewRegenerateForwardAgentTokenUseCase creates a new RegenerateForwardAgentTokenUseCase.
func NewRegenerateForwardAgentTokenUseCase(
	repo forward.AgentRepository,
	logger logger.Interface,
) *RegenerateForwardAgentTokenUseCase {
	return &RegenerateForwardAgentTokenUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute regenerates the API token for a forward agent.
func (uc *RegenerateForwardAgentTokenUseCase) Execute(ctx context.Context, cmd RegenerateForwardAgentTokenCommand) (*RegenerateForwardAgentTokenResult, error) {
	uc.logger.Infow("executing regenerate forward agent token use case", "id", cmd.ID)

	if cmd.ID == 0 {
		return nil, errors.NewValidationError("agent ID is required")
	}

	// Get the agent
	agent, err := uc.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "id", cmd.ID, "error", err)
		return nil, fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return nil, errors.NewNotFoundError("forward agent", fmt.Sprintf("%d", cmd.ID))
	}

	// Generate new token
	plainToken, err := agent.GenerateAPIToken()
	if err != nil {
		uc.logger.Errorw("failed to generate API token", "id", cmd.ID, "error", err)
		return nil, fmt.Errorf("failed to generate API token: %w", err)
	}

	// Persist changes
	if err := uc.repo.Update(ctx, agent); err != nil {
		uc.logger.Errorw("failed to update forward agent token", "id", cmd.ID, "error", err)
		return nil, fmt.Errorf("failed to update forward agent: %w", err)
	}

	result := &RegenerateForwardAgentTokenResult{
		ID:    agent.ID(),
		Token: plainToken,
	}

	uc.logger.Infow("forward agent token regenerated successfully", "id", cmd.ID)
	return result, nil
}
