// Package usecases contains the application use cases for forward domain.
package usecases

import (
	"context"
	"encoding/json"
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

// String implements fmt.Stringer for logging purposes.
func (uc *ReportRuleSyncStatusUseCase) String() string {
	return "ReportRuleSyncStatusUseCase"
}

// HandleMessage processes rule sync status messages from agents via WebSocket.
// Implements agent.MessageHandler interface.
func (uc *ReportRuleSyncStatusUseCase) HandleMessage(agentID uint, msgType string, data any) bool {
	if msgType != dto.MsgTypeRuleSyncStatus {
		return false
	}

	// Parse the rule sync status data
	dataBytes, err := json.Marshal(data)
	if err != nil {
		uc.logger.Warnw("failed to marshal rule sync status data",
			"agent_id", agentID,
			"error", err,
		)
		return true
	}

	var statusData struct {
		Rules []dto.RuleSyncStatusItem `json:"rules"`
	}
	if err := json.Unmarshal(dataBytes, &statusData); err != nil {
		uc.logger.Warnw("failed to unmarshal rule sync status data",
			"agent_id", agentID,
			"error", err,
		)
		return true
	}

	// Execute the use case
	input := &dto.ReportRuleSyncStatusInput{
		AgentID: agentID,
		Rules:   statusData.Rules,
	}

	if err := uc.Execute(context.Background(), input); err != nil {
		uc.logger.Warnw("failed to process rule sync status via websocket",
			"agent_id", agentID,
			"error", err,
		)
	}

	return true
}
