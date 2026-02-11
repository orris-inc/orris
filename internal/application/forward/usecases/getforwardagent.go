package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetForwardAgentQuery represents the input for getting a forward agent.
type GetForwardAgentQuery struct {
	ShortID string // External API identifier
}

// GetForwardAgentResult represents the output of getting a forward agent.
type GetForwardAgentResult struct {
	ID            string              `json:"id"` // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Name          string              `json:"name"`
	PublicAddress string              `json:"public_address"`
	Status        string              `json:"status"`
	Remark        string              `json:"remark"`
	CreatedAt     string              `json:"created_at"`
	UpdatedAt     string              `json:"updated_at"`
	SystemStatus  *dto.AgentStatusDTO `json:"system_status,omitempty"`
}

// GetForwardAgentUseCase handles retrieving a single forward agent.
type GetForwardAgentUseCase struct {
	repo          forward.AgentRepository
	statusQuerier AgentStatusQuerier
	logger        logger.Interface
}

// NewGetForwardAgentUseCase creates a new GetForwardAgentUseCase.
func NewGetForwardAgentUseCase(
	repo forward.AgentRepository,
	statusQuerier AgentStatusQuerier,
	logger logger.Interface,
) *GetForwardAgentUseCase {
	return &GetForwardAgentUseCase{
		repo:          repo,
		statusQuerier: statusQuerier,
		logger:        logger,
	}
}

// Execute retrieves a forward agent by ShortID.
func (uc *GetForwardAgentUseCase) Execute(ctx context.Context, query GetForwardAgentQuery) (*GetForwardAgentResult, error) {
	if query.ShortID == "" {
		return nil, errors.NewValidationError("short_id is required")
	}

	uc.logger.Debugw("executing get forward agent use case", "short_id", query.ShortID)

	agent, err := uc.repo.GetBySID(ctx, query.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "short_id", query.ShortID, "error", err)
		return nil, fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return nil, errors.NewNotFoundError("forward agent", query.ShortID)
	}

	result := &GetForwardAgentResult{
		ID:            agent.SID(),
		Name:          agent.Name(),
		PublicAddress: agent.PublicAddress(),
		Status:        string(agent.Status()),
		Remark:        agent.Remark(),
		CreatedAt:     agent.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     agent.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	// Query system status from cache using internal ID
	if uc.statusQuerier != nil {
		systemStatus, err := uc.statusQuerier.GetStatus(ctx, agent.ID())
		if err != nil {
			uc.logger.Warnw("failed to get agent system status, continuing without it",
				"agent_id", agent.ID(),
				"short_id", agent.SID(),
				"error", err,
			)
		} else if systemStatus != nil {
			result.SystemStatus = systemStatus
		}
	}

	uc.logger.Debugw("forward agent retrieved successfully", "short_id", agent.SID())
	return result, nil
}
