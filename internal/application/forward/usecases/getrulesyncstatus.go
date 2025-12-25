package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// RuleSyncStatusQuerier defines the interface for querying rule sync status from cache.
type RuleSyncStatusQuerier interface {
	GetRuleStatus(ctx context.Context, agentID uint) (*dto.RuleSyncStatusQueryResult, error)
}

// GetRuleSyncStatusQuery represents the query for GetRuleSyncStatus use case.
type GetRuleSyncStatusQuery struct {
	ShortID string // Stripe-style agent ID (external API identifier)
}

// GetRuleSyncStatusUseCase handles rule sync status querying from admin side.
type GetRuleSyncStatusUseCase struct {
	agentRepo     forward.AgentRepository
	statusQuerier RuleSyncStatusQuerier
	logger        logger.Interface
}

// NewGetRuleSyncStatusUseCase creates a new GetRuleSyncStatusUseCase.
func NewGetRuleSyncStatusUseCase(
	agentRepo forward.AgentRepository,
	statusQuerier RuleSyncStatusQuerier,
	logger logger.Interface,
) *GetRuleSyncStatusUseCase {
	return &GetRuleSyncStatusUseCase{
		agentRepo:     agentRepo,
		statusQuerier: statusQuerier,
		logger:        logger,
	}
}

// Execute retrieves rule sync status for an agent.
func (uc *GetRuleSyncStatusUseCase) Execute(ctx context.Context, query GetRuleSyncStatusQuery) (*dto.RuleSyncStatusResponse, error) {
	if query.ShortID == "" {
		return nil, fmt.Errorf("short_id is required")
	}

	uc.logger.Infow("executing get rule sync status use case", "short_id", query.ShortID)

	// Verify agent exists
	agent, err := uc.agentRepo.GetBySID(ctx, query.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get agent", "short_id", query.ShortID, "error", err)
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	if agent == nil {
		return nil, fmt.Errorf("agent not found: %s", query.ShortID)
	}

	// Query rule sync status from cache using internal ID
	result, err := uc.statusQuerier.GetRuleStatus(ctx, agent.ID())
	if err != nil {
		uc.logger.Errorw("failed to get rule sync status", "agent_id", agent.ID(), "short_id", agent.SID(), "error", err)
		return nil, fmt.Errorf("failed to get rule sync status: %w", err)
	}

	// Build response from query result
	response := &dto.RuleSyncStatusResponse{
		AgentID:   agent.SID(),
		Rules:     result.Rules,
		UpdatedAt: result.UpdatedAt,
	}

	uc.logger.Infow("rule sync status retrieved successfully",
		"short_id", agent.SID(),
		"rules_count", len(result.Rules),
	)

	return response, nil
}
