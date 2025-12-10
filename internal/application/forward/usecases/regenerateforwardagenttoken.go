package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// RegenerateForwardAgentTokenCommand represents the input for regenerating an agent token.
// Use either ID (internal) or ShortID (external API identifier).
type RegenerateForwardAgentTokenCommand struct {
	ID      uint   // Internal database ID (deprecated, use ShortID for external API)
	ShortID string // External API identifier (without prefix)
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
	var agent *forward.ForwardAgent
	var err error

	// Prefer ShortID over internal ID for external API
	if cmd.ShortID != "" {
		uc.logger.Infow("executing regenerate forward agent token use case", "short_id", cmd.ShortID)
		agent, err = uc.repo.GetByShortID(ctx, cmd.ShortID)
		if err != nil {
			uc.logger.Errorw("failed to get forward agent", "short_id", cmd.ShortID, "error", err)
			return nil, fmt.Errorf("failed to get forward agent: %w", err)
		}
		if agent == nil {
			return nil, errors.NewNotFoundError("forward agent", cmd.ShortID)
		}
	} else if cmd.ID != 0 {
		uc.logger.Infow("executing regenerate forward agent token use case", "id", cmd.ID)
		agent, err = uc.repo.GetByID(ctx, cmd.ID)
		if err != nil {
			uc.logger.Errorw("failed to get forward agent", "id", cmd.ID, "error", err)
			return nil, fmt.Errorf("failed to get forward agent: %w", err)
		}
		if agent == nil {
			return nil, errors.NewNotFoundError("forward agent", fmt.Sprintf("%d", cmd.ID))
		}
	} else {
		return nil, errors.NewValidationError("agent ID or short_id is required")
	}

	// Generate new token
	plainToken, err := agent.GenerateAPIToken()
	if err != nil {
		uc.logger.Errorw("failed to generate API token", "id", agent.ID(), "short_id", agent.ShortID(), "error", err)
		return nil, fmt.Errorf("failed to generate API token: %w", err)
	}

	// Persist changes
	if err := uc.repo.Update(ctx, agent); err != nil {
		uc.logger.Errorw("failed to update forward agent token", "id", agent.ID(), "short_id", agent.ShortID(), "error", err)
		return nil, fmt.Errorf("failed to update forward agent: %w", err)
	}

	result := &RegenerateForwardAgentTokenResult{
		ID:    agent.ID(),
		Token: plainToken,
	}

	uc.logger.Infow("forward agent token regenerated successfully", "id", agent.ID(), "short_id", agent.ShortID())
	return result, nil
}
