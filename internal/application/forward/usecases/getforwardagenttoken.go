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
	ID uint
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
	uc.logger.Infow("executing get forward agent token use case", "id", query.ID)

	if query.ID == 0 {
		return nil, errors.NewValidationError("agent ID is required")
	}

	// Get the agent
	agent, err := uc.repo.GetByID(ctx, query.ID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "id", query.ID, "error", err)
		return nil, fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return nil, errors.NewNotFoundError("forward agent", fmt.Sprintf("%d", query.ID))
	}

	result := &GetForwardAgentTokenResult{
		ID:       agent.ID(),
		Token:    agent.GetAPIToken(),
		HasToken: agent.HasToken(),
	}

	uc.logger.Infow("forward agent token retrieved successfully", "id", query.ID, "has_token", result.HasToken)
	return result, nil
}
