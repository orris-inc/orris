package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ListSubscriptionForwardRulesQuery represents the input for listing a subscription's forward rules.
type ListSubscriptionForwardRulesQuery struct {
	SubscriptionID uint
	Page           int
	PageSize       int
	Name           string
	Protocol       string
	Status         string
	OrderBy        string
	Order          string
}

// ListSubscriptionForwardRulesResult represents the output of listing a subscription's forward rules.
type ListSubscriptionForwardRulesResult struct {
	Rules []*dto.ForwardRuleDTO `json:"rules"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Pages int                   `json:"pages"`
}

// ListSubscriptionForwardRulesUseCase handles listing forward rules for a specific subscription.
// It supports resource group priority mode: if the subscription's plan has active resource groups
// with forward rules, those rules are returned; otherwise, direct subscription-bound rules are returned.
type ListSubscriptionForwardRulesUseCase struct {
	repo              forward.Repository
	agentRepo         forward.AgentRepository
	nodeRepo          node.NodeRepository
	subscriptionRepo  subscription.SubscriptionRepository
	resourceGroupRepo resource.Repository
	statusQuerier     RuleSyncStatusBatchQuerier
	logger            logger.Interface
}

// NewListSubscriptionForwardRulesUseCase creates a new ListSubscriptionForwardRulesUseCase.
func NewListSubscriptionForwardRulesUseCase(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	resourceGroupRepo resource.Repository,
	statusQuerier RuleSyncStatusBatchQuerier,
	logger logger.Interface,
) *ListSubscriptionForwardRulesUseCase {
	return &ListSubscriptionForwardRulesUseCase{
		repo:              repo,
		agentRepo:         agentRepo,
		nodeRepo:          nodeRepo,
		subscriptionRepo:  subscriptionRepo,
		resourceGroupRepo: resourceGroupRepo,
		statusQuerier:     statusQuerier,
		logger:            logger,
	}
}

// Execute retrieves a list of forward rules for a specific subscription.
// Resource group priority mode: if the plan has active resource groups with rules, return those;
// otherwise, return direct subscription-bound rules.
func (uc *ListSubscriptionForwardRulesUseCase) Execute(ctx context.Context, query ListSubscriptionForwardRulesQuery) (*ListSubscriptionForwardRulesResult, error) {
	uc.logger.Infow("executing list subscription forward rules use case",
		"subscription_id", query.SubscriptionID,
		"page", query.Page,
		"page_size", query.PageSize,
	)

	// Validate subscription ID
	if query.SubscriptionID == 0 {
		return nil, errors.NewValidationError("subscription_id is required")
	}

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

	// Try to get rules from resource groups first (resource group priority mode)
	rules, err := uc.getRulesWithResourceGroupPriority(ctx, query.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to list subscription forward rules", "subscription_id", query.SubscriptionID, "error", err)
		return nil, fmt.Errorf("failed to list subscription forward rules: %w", err)
	}

	// Apply filters manually since ListBySubscriptionID doesn't support filtering
	filteredRules := uc.applyFilters(rules, query)

	// Calculate total and pagination
	total := int64(len(filteredRules))
	pages := int(total) / query.PageSize
	if int(total)%query.PageSize > 0 {
		pages++
	}

	// Apply pagination
	start := (query.Page - 1) * query.PageSize
	end := start + query.PageSize
	if start > len(filteredRules) {
		filteredRules = []*forward.ForwardRule{}
	} else if end > len(filteredRules) {
		filteredRules = filteredRules[start:]
	} else {
		filteredRules = filteredRules[start:end]
	}

	// Convert to DTOs
	dtos := dto.ToForwardRuleDTOs(filteredRules)

	// Populate agent info (AgentID and ExitAgentID)
	agentIDs := dto.CollectAgentIDs(dtos)
	if len(agentIDs) > 0 && uc.agentRepo != nil {
		agentShortIDs, err := uc.agentRepo.GetSIDsByIDs(ctx, agentIDs)
		if err != nil {
			uc.logger.Warnw("failed to fetch agent short IDs", "subscription_id", query.SubscriptionID, "error", err)
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
			uc.logger.Warnw("failed to fetch target nodes", "subscription_id", query.SubscriptionID, "error", err)
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

	uc.logger.Infow("subscription forward rules listed successfully",
		"subscription_id", query.SubscriptionID,
		"total", total,
	)

	return &ListSubscriptionForwardRulesResult{
		Rules: dtos,
		Total: total,
		Page:  query.Page,
		Pages: pages,
	}, nil
}

// applyFilters applies query filters to the rules list.
func (uc *ListSubscriptionForwardRulesUseCase) applyFilters(rules []*forward.ForwardRule, query ListSubscriptionForwardRulesQuery) []*forward.ForwardRule {
	var filtered []*forward.ForwardRule

	for _, rule := range rules {
		// Filter by name (case-insensitive contains)
		if query.Name != "" {
			if !containsIgnoreCase(rule.Name(), query.Name) {
				continue
			}
		}

		// Filter by protocol
		if query.Protocol != "" {
			if rule.Protocol().String() != query.Protocol {
				continue
			}
		}

		// Filter by status
		if query.Status != "" {
			if rule.Status().String() != query.Status {
				continue
			}
		}

		filtered = append(filtered, rule)
	}

	return filtered
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// populateSyncStatus populates runtime sync status for all rules.
func (uc *ListSubscriptionForwardRulesUseCase) populateSyncStatus(ctx context.Context, dtos []*dto.ForwardRuleDTO, agentIDs []uint) {
	if len(agentIDs) == 0 {
		return
	}

	statusMap, err := uc.statusQuerier.GetMultipleRuleStatus(ctx, agentIDs)
	if err != nil {
		uc.logger.Warnw("failed to fetch rule sync statuses", "error", err, "agent_count", len(agentIDs))
		return
	}

	ruleAgentMap := dto.CollectAllAgentIDsForRules(dtos)

	for _, ruleDTO := range dtos {
		// Skip disabled rules - they are not synced to agents
		if ruleDTO.Status == "disabled" {
			continue
		}

		ruleAgentIDs := ruleAgentMap[ruleDTO.ID]
		if len(ruleAgentIDs) == 0 {
			continue
		}

		statusInfo := uc.aggregateRuleStatus(ruleDTO.ID, ruleAgentIDs, statusMap)
		ruleDTO.PopulateSyncStatus(statusInfo)
	}
}

// aggregateRuleStatus aggregates sync status from all agents for a single rule.
func (uc *ListSubscriptionForwardRulesUseCase) aggregateRuleStatus(
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

		if syncStatus == "synced" && runStatus == "running" && !hasError {
			healthyCount++
		}
	}

	if latestUpdate == 0 {
		latestUpdate = biztime.NowUTC().Unix()
	}

	return &dto.RuleSyncStatusInfo{
		SyncStatus:    dto.AggregateSyncStatus(syncStatuses),
		RunStatus:     dto.AggregateRunStatus(runStatuses),
		TotalAgents:   len(agentIDs),
		HealthyAgents: healthyCount,
		UpdatedAt:     latestUpdate,
	}
}

// getRulesWithResourceGroupPriority implements resource group priority mode:
// 1. Get subscription's planID
// 2. Get active resource groups for the plan
// 3. If resource groups have rules, return those rules
// 4. Otherwise, return direct subscription-bound rules
func (uc *ListSubscriptionForwardRulesUseCase) getRulesWithResourceGroupPriority(ctx context.Context, subscriptionID uint) ([]*forward.ForwardRule, error) {
	// Get subscription to find planID
	sub, err := uc.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get active resource groups for the plan
	groups, err := uc.resourceGroupRepo.GetByPlanID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Warnw("failed to get resource groups, falling back to subscription rules",
			"subscription_id", subscriptionID,
			"plan_id", sub.PlanID(),
			"error", err)
		// Fall back to subscription-bound rules
		return uc.repo.ListBySubscriptionID(ctx, subscriptionID)
	}

	// Collect active group IDs
	var activeGroupIDs []uint
	for _, g := range groups {
		if g.Status() == resource.GroupStatusActive {
			activeGroupIDs = append(activeGroupIDs, g.ID())
		}
	}

	// If no active resource groups, return subscription-bound rules
	if len(activeGroupIDs) == 0 {
		uc.logger.Debugw("no active resource groups, using subscription rules",
			"subscription_id", subscriptionID,
			"plan_id", sub.PlanID())
		return uc.repo.ListBySubscriptionID(ctx, subscriptionID)
	}

	// Get rules from all active resource groups
	var allRules []*forward.ForwardRule
	seenRuleIDs := make(map[uint]bool)

	for _, groupID := range activeGroupIDs {
		// Use page=0 and pageSize=0 to get all rules without pagination
		rules, _, err := uc.repo.ListByGroupID(ctx, groupID, 0, 0)
		if err != nil {
			uc.logger.Warnw("failed to get rules for resource group",
				"group_id", groupID,
				"error", err)
			continue
		}

		// Deduplicate rules (a rule can belong to multiple groups)
		for _, rule := range rules {
			if !seenRuleIDs[rule.ID()] {
				seenRuleIDs[rule.ID()] = true
				allRules = append(allRules, rule)
			}
		}
	}

	// If resource groups have rules, return them
	if len(allRules) > 0 {
		uc.logger.Debugw("returning rules from resource groups",
			"subscription_id", subscriptionID,
			"plan_id", sub.PlanID(),
			"group_count", len(activeGroupIDs),
			"rule_count", len(allRules))
		return allRules, nil
	}

	// No rules in resource groups, fall back to subscription-bound rules
	uc.logger.Debugw("no rules in resource groups, using subscription rules",
		"subscription_id", subscriptionID,
		"plan_id", sub.PlanID(),
		"group_count", len(activeGroupIDs))
	return uc.repo.ListBySubscriptionID(ctx, subscriptionID)
}
