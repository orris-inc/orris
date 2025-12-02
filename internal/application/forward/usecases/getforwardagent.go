package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/forward"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// GetForwardAgentQuery represents the input for getting a forward agent.
type GetForwardAgentQuery struct {
	ID uint
}

// GetForwardAgentResult represents the output of getting a forward agent.
type GetForwardAgentResult struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Remark    string `json:"remark"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// GetForwardAgentUseCase handles retrieving a single forward agent.
type GetForwardAgentUseCase struct {
	repo   forward.AgentRepository
	logger logger.Interface
}

// NewGetForwardAgentUseCase creates a new GetForwardAgentUseCase.
func NewGetForwardAgentUseCase(
	repo forward.AgentRepository,
	logger logger.Interface,
) *GetForwardAgentUseCase {
	return &GetForwardAgentUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute retrieves a forward agent by ID.
func (uc *GetForwardAgentUseCase) Execute(ctx context.Context, query GetForwardAgentQuery) (*GetForwardAgentResult, error) {
	uc.logger.Infow("executing get forward agent use case", "id", query.ID)

	if query.ID == 0 {
		return nil, errors.NewValidationError("agent ID is required")
	}

	agent, err := uc.repo.GetByID(ctx, query.ID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "id", query.ID, "error", err)
		return nil, fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return nil, errors.NewNotFoundError("forward agent", fmt.Sprintf("%d", query.ID))
	}

	result := &GetForwardAgentResult{
		ID:        agent.ID(),
		Name:      agent.Name(),
		Status:    string(agent.Status()),
		Remark:    agent.Remark(),
		CreatedAt: agent.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: agent.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	uc.logger.Infow("forward agent retrieved successfully", "id", query.ID)
	return result, nil
}
