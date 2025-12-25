// Package usecases contains the application use cases for forward domain.
package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// RuleSyncStatusUpdater defines the interface for updating rule sync status in cache.
type RuleSyncStatusUpdater interface {
	UpdateRuleStatus(ctx context.Context, agentID uint, rules []dto.RuleSyncStatusItem) error
}

// ReportRuleSyncStatusUseCase handles rule sync status reporting from agents.
type ReportRuleSyncStatusUseCase struct {
	agentRepo     forward.AgentRepository
	statusUpdater RuleSyncStatusUpdater
	ruleRepo      forward.Repository
	logger        logger.Interface
}

// NewReportRuleSyncStatusUseCase creates a new ReportRuleSyncStatusUseCase.
func NewReportRuleSyncStatusUseCase(
	agentRepo forward.AgentRepository,
	statusUpdater RuleSyncStatusUpdater,
	ruleRepo forward.Repository,
	logger logger.Interface,
) *ReportRuleSyncStatusUseCase {
	return &ReportRuleSyncStatusUseCase{
		agentRepo:     agentRepo,
		statusUpdater: statusUpdater,
		ruleRepo:      ruleRepo,
		logger:        logger,
	}
}

// Execute validates agent and updates rule sync status.
func (uc *ReportRuleSyncStatusUseCase) Execute(ctx context.Context, input *dto.ReportRuleSyncStatusInput) error {
	// Verify agent exists
	agent, err := uc.agentRepo.GetByID(ctx, input.AgentID)
	if err != nil {
		uc.logger.Errorw("failed to get agent", "agent_id", input.AgentID, "error", err)
		return fmt.Errorf("get agent: %w", err)
	}
	if agent == nil {
		return fmt.Errorf("agent not found: %d", input.AgentID)
	}

	// Update rule sync status in cache
	if err := uc.statusUpdater.UpdateRuleStatus(ctx, input.AgentID, input.Rules); err != nil {
		uc.logger.Errorw("failed to update rule sync status", "agent_id", input.AgentID, "error", err)
		return fmt.Errorf("update rule status: %w", err)
	}

	uc.logger.Debugw("rule sync status reported",
		"agent_id", input.AgentID,
		"rules_count", len(input.Rules),
	)

	return nil
}
