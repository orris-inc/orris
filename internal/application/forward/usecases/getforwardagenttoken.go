package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetForwardAgentTokenQuery represents the input for getting an agent token.
// Use either ID (internal) or ShortID (external API identifier).
type GetForwardAgentTokenQuery struct {
	ID      uint   // Internal database ID (deprecated, use ShortID for external API)
	ShortID string // External API identifier (without prefix)
}

// GetForwardAgentTokenResult represents the output of getting an agent token.
type GetForwardAgentTokenResult struct {
	ID       uint   `json:"id"`
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
	var agent *forward.ForwardAgent
	var err error

	// Prefer ShortID over internal ID for external API
	if query.ShortID != "" {
		uc.logger.Infow("executing get forward agent token use case", "short_id", query.ShortID)
		agent, err = uc.repo.GetByShortID(ctx, query.ShortID)
		if err != nil {
			uc.logger.Errorw("failed to get forward agent", "short_id", query.ShortID, "error", err)
			return nil, fmt.Errorf("failed to get forward agent: %w", err)
		}
		if agent == nil {
			return nil, errors.NewNotFoundError("forward agent", query.ShortID)
		}
	} else if query.ID != 0 {
		uc.logger.Infow("executing get forward agent token use case", "id", query.ID)
		agent, err = uc.repo.GetByID(ctx, query.ID)
		if err != nil {
			uc.logger.Errorw("failed to get forward agent", "id", query.ID, "error", err)
			return nil, fmt.Errorf("failed to get forward agent: %w", err)
		}
		if agent == nil {
			return nil, errors.NewNotFoundError("forward agent", fmt.Sprintf("%d", query.ID))
		}
	} else {
		return nil, errors.NewValidationError("agent ID or short_id is required")
	}

	result := &GetForwardAgentTokenResult{
		ID:       agent.ID(),
		Token:    agent.GetAPIToken(),
		HasToken: agent.HasToken(),
	}

	uc.logger.Infow("forward agent token retrieved successfully", "id", agent.ID(), "short_id", agent.ShortID(), "has_token", result.HasToken)
	return result, nil
}
