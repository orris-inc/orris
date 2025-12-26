package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ListForwardRulesQuery represents the input for listing forward rules.
type ListForwardRulesQuery struct {
	Page     int
	PageSize int
	Name     string
	Protocol string
	Status   string
	OrderBy  string
	Order    string
}

// ListForwardRulesResult represents the output of listing forward rules.
type ListForwardRulesResult struct {
	Rules []*dto.ForwardRuleDTO `json:"rules"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Pages int                   `json:"pages"`
}

// ListForwardRulesUseCase handles listing forward rules.
type ListForwardRulesUseCase struct {
	repo          forward.Repository
	agentRepo     forward.AgentRepository
	nodeRepo      node.NodeRepository
	statusQuerier RuleSyncStatusBatchQuerier
	logger        logger.Interface
}

// NewListForwardRulesUseCase creates a new ListForwardRulesUseCase.
func NewListForwardRulesUseCase(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	statusQuerier RuleSyncStatusBatchQuerier,
	logger logger.Interface,
) *ListForwardRulesUseCase {
	return &ListForwardRulesUseCase{
		repo:          repo,
		agentRepo:     agentRepo,
		nodeRepo:      nodeRepo,
		statusQuerier: statusQuerier,
		logger:        logger,
	}
}

// Execute retrieves a list of forward rules.
func (uc *ListForwardRulesUseCase) Execute(ctx context.Context, query ListForwardRulesQuery) (*ListForwardRulesResult, error) {
	uc.logger.Infow("executing list forward rules use case", "page", query.Page, "page_size", query.PageSize)

	// Set defaults
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	filter := forward.ListFilter{
		Page:     query.Page,
		PageSize: query.PageSize,
		Name:     query.Name,
		Protocol: query.Protocol,
		Status:   query.Status,
		OrderBy:  query.OrderBy,
		Order:    query.Order,
	}

	rules, total, err := uc.repo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list forward rules", "error", err)
		return nil, fmt.Errorf("failed to list forward rules: %w", err)
	}

	// Calculate total pages
	pages := int(total) / query.PageSize
	if int(total)%query.PageSize > 0 {
		pages++
	}

	// Convert to DTOs
	dtos := dto.ToForwardRuleDTOs(rules)

	// Populate agent info (AgentID and ExitAgentID)
	agentIDs := dto.CollectAgentIDs(dtos)
	if len(agentIDs) > 0 && uc.agentRepo != nil {
		agentShortIDs, err := uc.agentRepo.GetSIDsByIDs(ctx, agentIDs)
		if err != nil {
			uc.logger.Warnw("failed to fetch agent short IDs", "error", err)
			// Continue without agent info
		} else {
			for _, ruleDTO := range dtos {
				ruleDTO.PopulateAgentInfo(agentShortIDs)
			}
		}
	}

	// Collect target node IDs from DTOs
	nodeIDs := dto.CollectTargetNodeIDs(dtos)

	// Fetch target nodes and populate info
	if len(nodeIDs) > 0 && uc.nodeRepo != nil {
		nodes, err := uc.nodeRepo.GetByIDs(ctx, nodeIDs)
		if err != nil {
			uc.logger.Warnw("failed to fetch target nodes", "error", err)
			// Continue without node info
		} else {
			// Build node map for info and SID map
			nodeMap := make(map[uint]*node.Node)
			nodeSIDMap := make(dto.NodeSIDMap)
			for _, n := range nodes {
				nodeMap[n.ID()] = n
				nodeSIDMap[n.ID()] = n.SID()
			}
			// Populate target node SID and info
			for _, ruleDTO := range dtos {
				ruleDTO.PopulateTargetNodeSID(nodeSIDMap)
				if targetNodeID := ruleDTO.InternalTargetNodeID(); targetNodeID != nil {
					if n, ok := nodeMap[*targetNodeID]; ok {
						ruleDTO.PopulateTargetNodeInfo(&dto.TargetNodeInfo{
							ServerAddress: n.ServerAddress().Value(),
							PublicIPv4:    n.PublicIPv4(),
							PublicIPv6:    n.PublicIPv6(),
						})
					}
				}
			}
		}
	}

	// Populate runtime sync status for each rule
	if uc.statusQuerier != nil && len(dtos) > 0 {
		uc.populateSyncStatus(ctx, dtos, agentIDs)
	}

	return &ListForwardRulesResult{
		Rules: dtos,
		Total: total,
		Page:  query.Page,
		Pages: pages,
	}, nil
}

// populateSyncStatus populates runtime sync status for all rules.
func (uc *ListForwardRulesUseCase) populateSyncStatus(ctx context.Context, dtos []*dto.ForwardRuleDTO, agentIDs []uint) {
	// Batch query status for all agents involved in the rules
	if len(agentIDs) == 0 {
		return
	}

	statusMap, err := uc.statusQuerier.GetMultipleRuleStatus(ctx, agentIDs)
	if err != nil {
		uc.logger.Warnw("failed to fetch rule sync statuses", "error", err, "agent_count", len(agentIDs))
		return
	}

	// Build rule -> agent IDs mapping
	ruleAgentMap := dto.CollectAllAgentIDsForRules(dtos)

	// Aggregate status for each rule
	for _, ruleDTO := range dtos {
		ruleAgentIDs := ruleAgentMap[ruleDTO.ID]
		if len(ruleAgentIDs) == 0 {
			continue
		}

		statusInfo := uc.aggregateRuleStatus(ruleDTO.ID, ruleAgentIDs, statusMap)
		ruleDTO.PopulateSyncStatus(statusInfo)
	}
}

// aggregateRuleStatus aggregates sync status from all agents for a single rule.
func (uc *ListForwardRulesUseCase) aggregateRuleStatus(
	ruleSID string,
	agentIDs []uint,
	statusMap map[uint]*dto.RuleSyncStatusQueryResult,
) *dto.RuleSyncStatusInfo {
	if len(agentIDs) == 0 {
		return nil
	}

	var syncStatuses, runStatuses []string
	var latestUpdate int64
	healthyCount := 0

	for _, agentID := range agentIDs {
		syncStatus := "pending"
		runStatus := "unknown"
		hasError := false

		// Find this rule's status from agent's reported statuses
		if queryResult, ok := statusMap[agentID]; ok {
			if queryResult.UpdatedAt > latestUpdate {
				latestUpdate = queryResult.UpdatedAt
			}
			for _, ruleStatus := range queryResult.Rules {
				if ruleStatus.RuleID == ruleSID {
					syncStatus = ruleStatus.SyncStatus
					runStatus = ruleStatus.RunStatus
					hasError = ruleStatus.ErrorMessage != ""
					break
				}
			}
		}

		syncStatuses = append(syncStatuses, syncStatus)
		runStatuses = append(runStatuses, runStatus)

		// Count healthy agents
		if syncStatus == "synced" && runStatus == "running" && !hasError {
			healthyCount++
		}
	}

	// Use current time if no updates found
	if latestUpdate == 0 {
		latestUpdate = time.Now().Unix()
	}

	return &dto.RuleSyncStatusInfo{
		SyncStatus:    dto.AggregateSyncStatus(syncStatuses),
		RunStatus:     dto.AggregateRunStatus(runStatuses),
		TotalAgents:   len(agentIDs),
		HealthyAgents: healthyCount,
		UpdatedAt:     latestUpdate,
	}
}
