// Package usecases contains the application use cases for forward domain.
package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AgentStatusQuerier defines the interface for querying agent status from cache.
type AgentStatusQuerier interface {
	GetStatus(ctx context.Context, agentID uint) (*dto.AgentStatusDTO, error)
	GetMultipleStatus(ctx context.Context, agentIDs []uint) (map[uint]*dto.AgentStatusDTO, error)
}

// GetAgentStatusQuery represents the query for GetAgentStatus use case.
// Use either AgentID (internal) or ShortID (external API identifier).
type GetAgentStatusQuery struct {
	AgentID uint   // Internal database ID (deprecated, use ShortID for external API)
	ShortID string // External API identifier (without prefix)
}

// GetAgentStatusUseCase handles querying agent status.
type GetAgentStatusUseCase struct {
	agentRepo     forward.AgentRepository
	statusQuerier AgentStatusQuerier
	logger        logger.Interface
}

// NewGetAgentStatusUseCase creates a new GetAgentStatusUseCase.
func NewGetAgentStatusUseCase(
	agentRepo forward.AgentRepository,
	statusQuerier AgentStatusQuerier,
	logger logger.Interface,
) *GetAgentStatusUseCase {
	return &GetAgentStatusUseCase{
		agentRepo:     agentRepo,
		statusQuerier: statusQuerier,
		logger:        logger,
	}
}

// Execute queries agent status.
func (uc *GetAgentStatusUseCase) Execute(ctx context.Context, query GetAgentStatusQuery) (*dto.AgentStatusDTO, error) {
	var agent *forward.ForwardAgent
	var err error

	// Prefer ShortID over internal ID for external API
	if query.ShortID != "" {
		agent, err = uc.agentRepo.GetByShortID(ctx, query.ShortID)
		if err != nil {
			uc.logger.Errorw("failed to get agent", "short_id", query.ShortID, "error", err)
			return nil, fmt.Errorf("get agent: %w", err)
		}
		if agent == nil {
			return nil, fmt.Errorf("agent not found: %s", query.ShortID)
		}
	} else if query.AgentID != 0 {
		agent, err = uc.agentRepo.GetByID(ctx, query.AgentID)
		if err != nil {
			uc.logger.Errorw("failed to get agent", "agent_id", query.AgentID, "error", err)
			return nil, fmt.Errorf("get agent: %w", err)
		}
		if agent == nil {
			return nil, fmt.Errorf("agent not found: %d", query.AgentID)
		}
	} else {
		return nil, fmt.Errorf("agent ID or short_id is required")
	}

	// Get status from Redis using internal ID
	status, err := uc.statusQuerier.GetStatus(ctx, agent.ID())
	if err != nil {
		uc.logger.Errorw("failed to get agent status", "agent_id", agent.ID(), "short_id", agent.ShortID(), "error", err)
		return nil, fmt.Errorf("get status: %w", err)
	}

	// If no status found, return empty status (agent is offline)
	if status == nil {
		return &dto.AgentStatusDTO{}, nil
	}

	return status, nil
}
