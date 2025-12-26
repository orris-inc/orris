package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// RuleSyncStatusBatchQuerier defines the interface for batch querying rule sync status.
type RuleSyncStatusBatchQuerier interface {
	GetMultipleRuleStatus(ctx context.Context, agentIDs []uint) (map[uint]*dto.RuleSyncStatusQueryResult, error)
}

// GetRuleOverallStatusUseCase handles querying overall status for a forward rule.
type GetRuleOverallStatusUseCase struct {
	ruleRepo      forward.Repository
	agentRepo     forward.AgentRepository
	statusQuerier RuleSyncStatusBatchQuerier
	logger        logger.Interface
}

// NewGetRuleOverallStatusUseCase creates a new GetRuleOverallStatusUseCase.
func NewGetRuleOverallStatusUseCase(
	ruleRepo forward.Repository,
	agentRepo forward.AgentRepository,
	statusQuerier RuleSyncStatusBatchQuerier,
	logger logger.Interface,
) *GetRuleOverallStatusUseCase {
	return &GetRuleOverallStatusUseCase{
		ruleRepo:      ruleRepo,
		agentRepo:     agentRepo,
		statusQuerier: statusQuerier,
		logger:        logger,
	}
}

// Execute retrieves and aggregates the overall status for a forward rule.
func (uc *GetRuleOverallStatusUseCase) Execute(ctx context.Context, input *dto.GetRuleOverallStatusInput) (*dto.RuleOverallStatusResponse, error) {
	if input.RuleSID == "" {
		return nil, fmt.Errorf("rule_sid is required")
	}

	uc.logger.Infow("executing get rule overall status use case", "rule_sid", input.RuleSID)

	// 1. Get rule by SID
	rule, err := uc.ruleRepo.GetBySID(ctx, input.RuleSID)
	if err != nil {
		uc.logger.Errorw("failed to get rule", "rule_sid", input.RuleSID, "error", err)
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}
	if rule == nil {
		return nil, errors.NewNotFoundError("forward rule", input.RuleSID)
	}

	// 2. Collect all agent IDs involved in this rule
	agentIDs := uc.collectAgentIDs(rule)

	// 3. Batch query all agent statuses
	statusMap, err := uc.statusQuerier.GetMultipleRuleStatus(ctx, agentIDs)
	if err != nil {
		uc.logger.Errorw("failed to get multiple rule statuses",
			"rule_sid", input.RuleSID,
			"agent_count", len(agentIDs),
			"error", err,
		)
		return nil, fmt.Errorf("failed to get agent statuses: %w", err)
	}

	// 4. Batch query agent info (SIDs and names)
	agentMap, err := uc.agentRepo.GetByIDs(ctx, agentIDs)
	if err != nil {
		uc.logger.Errorw("failed to get agents",
			"rule_sid", input.RuleSID,
			"agent_count", len(agentIDs),
			"error", err,
		)
		return nil, fmt.Errorf("failed to get agents: %w", err)
	}

	// 5. Build SID and name maps from agent entities
	agentSIDMap := make(map[uint]string, len(agentMap))
	agentNameMap := make(map[uint]string, len(agentMap))
	for agentID, agent := range agentMap {
		agentSIDMap[agentID] = agent.SID()
		agentNameMap[agentID] = agent.Name()
	}

	// 6. Build agent status details
	agentStatuses := uc.buildAgentStatuses(rule, agentIDs, statusMap, agentSIDMap, agentNameMap)

	// 7. Aggregate overall status
	overallSyncStatus, overallRunStatus, healthyCount := uc.aggregateStatus(agentStatuses)

	// 8. Find the latest update timestamp
	latestUpdate := uc.findLatestUpdate(statusMap)

	response := &dto.RuleOverallStatusResponse{
		RuleID:            rule.SID(),
		OverallSyncStatus: overallSyncStatus,
		OverallRunStatus:  overallRunStatus,
		TotalAgents:       len(agentIDs),
		HealthyAgents:     healthyCount,
		AgentStatuses:     agentStatuses,
		UpdatedAt:         latestUpdate,
	}

	uc.logger.Infow("rule overall status retrieved successfully",
		"rule_sid", input.RuleSID,
		"total_agents", response.TotalAgents,
		"healthy_agents", response.HealthyAgents,
		"overall_sync_status", response.OverallSyncStatus,
		"overall_run_status", response.OverallRunStatus,
	)

	return response, nil
}

// collectAgentIDs collects all agent IDs involved in the rule based on rule type.
func (uc *GetRuleOverallStatusUseCase) collectAgentIDs(rule *forward.ForwardRule) []uint {
	agentIDs := []uint{rule.AgentID()} // Start with entry agent

	switch rule.RuleType().String() {
	case "entry":
		// Add exit agent
		if rule.ExitAgentID() != 0 {
			agentIDs = append(agentIDs, rule.ExitAgentID())
		}
	case "chain", "direct_chain":
		// Add all chain agents
		agentIDs = append(agentIDs, rule.ChainAgentIDs()...)
	}

	return agentIDs
}

// buildAgentStatuses builds detailed status for each agent.
func (uc *GetRuleOverallStatusUseCase) buildAgentStatuses(
	rule *forward.ForwardRule,
	agentIDs []uint,
	statusMap map[uint]*dto.RuleSyncStatusQueryResult,
	agentSIDMap map[uint]string,
	agentNameMap map[uint]string,
) []dto.AgentRuleSyncStatus {
	agentStatuses := make([]dto.AgentRuleSyncStatus, 0, len(agentIDs))

	for position, agentID := range agentIDs {
		agentSID := agentSIDMap[agentID]
		agentName := agentNameMap[agentID]

		// Default status if agent hasn't reported
		agentStatus := dto.AgentRuleSyncStatus{
			AgentID:      agentSID,
			AgentName:    agentName,
			Position:     position,
			SyncStatus:   "pending", // Default if not reported
			RunStatus:    "unknown", // Default if not reported
			ListenPort:   0,
			Connections:  0,
			ErrorMessage: "",
			SyncedAt:     0,
		}

		// Find this rule's status from agent's reported statuses
		if queryResult, ok := statusMap[agentID]; ok {
			for _, ruleStatus := range queryResult.Rules {
				if ruleStatus.RuleID == rule.SID() {
					// Found the status for this rule
					agentStatus.SyncStatus = ruleStatus.SyncStatus
					agentStatus.RunStatus = ruleStatus.RunStatus
					agentStatus.ListenPort = ruleStatus.ListenPort
					agentStatus.Connections = ruleStatus.Connections
					agentStatus.ErrorMessage = ruleStatus.ErrorMessage
					agentStatus.SyncedAt = ruleStatus.SyncedAt
					break
				}
			}
		}

		agentStatuses = append(agentStatuses, agentStatus)
	}

	return agentStatuses
}

// aggregateStatus aggregates overall sync and run status from all agents.
// Returns: overallSyncStatus, overallRunStatus, healthyAgentCount
func (uc *GetRuleOverallStatusUseCase) aggregateStatus(agentStatuses []dto.AgentRuleSyncStatus) (string, string, int) {
	if len(agentStatuses) == 0 {
		return "pending", "unknown", 0
	}

	// Collect all statuses
	syncStatuses := make([]string, 0, len(agentStatuses))
	runStatuses := make([]string, 0, len(agentStatuses))
	healthyCount := 0

	for _, agent := range agentStatuses {
		syncStatuses = append(syncStatuses, agent.SyncStatus)
		runStatuses = append(runStatuses, agent.RunStatus)

		// Count healthy agents
		if agent.SyncStatus == "synced" && agent.RunStatus == "running" && agent.ErrorMessage == "" {
			healthyCount++
		}
	}

	// Aggregate sync status: failed > pending > synced
	overallSyncStatus := dto.AggregateSyncStatus(syncStatuses)

	// Aggregate run status: error > stopped > starting > running
	overallRunStatus := dto.AggregateRunStatus(runStatuses)

	return overallSyncStatus, overallRunStatus, healthyCount
}

// findLatestUpdate finds the latest update timestamp from all agent statuses.
func (uc *GetRuleOverallStatusUseCase) findLatestUpdate(statusMap map[uint]*dto.RuleSyncStatusQueryResult) int64 {
	var latest int64 = 0

	for _, queryResult := range statusMap {
		if queryResult.UpdatedAt > latest {
			latest = queryResult.UpdatedAt
		}
	}

	// If no updates found, use current time
	if latest == 0 {
		latest = time.Now().Unix()
	}

	return latest
}
