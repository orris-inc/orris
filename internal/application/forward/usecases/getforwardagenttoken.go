package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetForwardAgentTokenQuery represents the input for getting an agent token.
type GetForwardAgentTokenQuery struct {
	ShortID string // External API identifier
}

// GetForwardAgentTokenResult represents the output of getting an agent token.
type GetForwardAgentTokenResult struct {
	ID       string `json:"id"` // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Token    string `json:"token"`
	HasToken bool   `json:"has_token"`
}

// GetForwardAgentTokenUseCase handles forward agent token retrieval.
type GetForwardAgentTokenUseCase struct {
	repo   forward.AgentRepository
	logger logger.Interface
}

// NewGetForwardAgentTokenUseCase creates a new GetForwardAgentTokenUseCase.
func NewGetForwardAgentTokenUseCase(
	repo forward.AgentRepository,
	logger logger.Interface,
) *GetForwardAgentTokenUseCase {
	return &GetForwardAgentTokenUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute retrieves the API token for a forward agent.
func (uc *GetForwardAgentTokenUseCase) Execute(ctx context.Context, query GetForwardAgentTokenQuery) (*GetForwardAgentTokenResult, error) {
	if query.ShortID == "" {
		return nil, errors.NewValidationError("short_id is required")
	}

	uc.logger.Infow("executing get forward agent token use case", "short_id", query.ShortID)

	agent, err := uc.repo.GetBySID(ctx, query.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "short_id", query.ShortID, "error", err)
		return nil, fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return nil, errors.NewNotFoundError("forward agent", query.ShortID)
	}

	result := &GetForwardAgentTokenResult{
		ID:       agent.SID(),
		Token:    agent.GetAPIToken(),
		HasToken: agent.HasToken(),
	}

	uc.logger.Infow("forward agent token retrieved successfully", "id", agent.ID(), "short_id", agent.SID(), "has_token", result.HasToken)
	return result, nil
}
