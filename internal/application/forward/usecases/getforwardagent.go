package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetForwardAgentQuery represents the input for getting a forward agent.
// Use either ID (internal) or ShortID (external API identifier).
type GetForwardAgentQuery struct {
	ID      uint   // Internal database ID (deprecated, use ShortID for external API)
	ShortID string // External API identifier (without prefix)
}

// GetForwardAgentResult represents the output of getting a forward agent.
type GetForwardAgentResult struct {
	ID            string `json:"id"` // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Name          string `json:"name"`
	PublicAddress string `json:"public_address"`
	Status        string `json:"status"`
	Remark        string `json:"remark"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
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

// Execute retrieves a forward agent by ID or ShortID.
func (uc *GetForwardAgentUseCase) Execute(ctx context.Context, query GetForwardAgentQuery) (*GetForwardAgentResult, error) {
	var agent *forward.ForwardAgent
	var err error

	// Prefer ShortID over internal ID for external API
	if query.ShortID != "" {
		uc.logger.Infow("executing get forward agent use case", "short_id", query.ShortID)
		agent, err = uc.repo.GetByShortID(ctx, query.ShortID)
		if err != nil {
			uc.logger.Errorw("failed to get forward agent", "short_id", query.ShortID, "error", err)
			return nil, fmt.Errorf("failed to get forward agent: %w", err)
		}
		if agent == nil {
			return nil, errors.NewNotFoundError("forward agent", query.ShortID)
		}
	} else if query.ID != 0 {
		uc.logger.Infow("executing get forward agent use case", "id", query.ID)
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

	result := &GetForwardAgentResult{
		ID:            id.FormatForwardAgentID(agent.ShortID()),
		Name:          agent.Name(),
		PublicAddress: agent.PublicAddress(),
		Status:        string(agent.Status()),
		Remark:        agent.Remark(),
		CreatedAt:     agent.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     agent.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	uc.logger.Infow("forward agent retrieved successfully", "short_id", agent.ShortID())
	return result, nil
}
