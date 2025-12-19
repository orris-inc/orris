package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// RegenerateForwardAgentTokenCommand represents the input for regenerating an agent token.
type RegenerateForwardAgentTokenCommand struct {
	ShortID string // External API identifier
}

// RegenerateForwardAgentTokenResult represents the output of regenerating an agent token.
type RegenerateForwardAgentTokenResult struct {
	ID    string `json:"id"` // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Token string `json:"token"`
}

// RegenerateForwardAgentTokenUseCase handles forward agent token regeneration.
type RegenerateForwardAgentTokenUseCase struct {
	repo     forward.AgentRepository
	tokenGen AgentTokenGenerator
	logger   logger.Interface
}

// NewRegenerateForwardAgentTokenUseCase creates a new RegenerateForwardAgentTokenUseCase.
func NewRegenerateForwardAgentTokenUseCase(
	repo forward.AgentRepository,
	tokenGen AgentTokenGenerator,
	logger logger.Interface,
) *RegenerateForwardAgentTokenUseCase {
	return &RegenerateForwardAgentTokenUseCase{
		repo:     repo,
		tokenGen: tokenGen,
		logger:   logger,
	}
}

// Execute regenerates the API token for a forward agent.
func (uc *RegenerateForwardAgentTokenUseCase) Execute(ctx context.Context, cmd RegenerateForwardAgentTokenCommand) (*RegenerateForwardAgentTokenResult, error) {
	if cmd.ShortID == "" {
		return nil, errors.NewValidationError("short_id is required")
	}

	uc.logger.Infow("executing regenerate forward agent token use case", "short_id", cmd.ShortID)

	agent, err := uc.repo.GetBySID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "short_id", cmd.ShortID, "error", err)
		return nil, fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return nil, errors.NewNotFoundError("forward agent", cmd.ShortID)
	}

	// Generate new token using HMAC-based token generator
	plainToken, tokenHash := uc.tokenGen.Generate(agent.SID())
	agent.SetAPIToken(plainToken, tokenHash)

	// Persist changes
	if err := uc.repo.Update(ctx, agent); err != nil {
		uc.logger.Errorw("failed to update forward agent token", "id", agent.ID(), "short_id", agent.SID(), "error", err)
		return nil, fmt.Errorf("failed to update forward agent: %w", err)
	}

	result := &RegenerateForwardAgentTokenResult{
		ID:    agent.SID(),
		Token: plainToken,
	}

	uc.logger.Infow("forward agent token regenerated successfully", "id", agent.ID(), "short_id", agent.SID())
	return result, nil
}
