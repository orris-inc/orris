package usecases

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/orris-inc/orris/internal/application/externalforward/dto"
	"github.com/orris-inc/orris/internal/domain/externalforward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ListExternalForwardRulesQuery represents the input for listing external forward rules.
type ListExternalForwardRulesQuery struct {
	SubscriptionID  uint
	SubscriptionSID string
	Page            int
	PageSize        int
	Status          string
	OrderBy         string
	Order           string
}

// ListExternalForwardRulesResult represents the output of listing external forward rules.
type ListExternalForwardRulesResult struct {
	Rules []*dto.ExternalForwardRuleDTO `json:"rules"`
	Total int64                         `json:"total"`
}

// ListExternalForwardRulesUseCase handles listing external forward rules.
// It supports resource group priority mode: if the subscription's plan has active resource groups
// with external forward rules, those rules are returned; otherwise, direct subscription-bound rules are returned.
type ListExternalForwardRulesUseCase struct {
	repo              externalforward.Repository
	subscriptionRepo  subscription.SubscriptionRepository
	resourceGroupRepo resource.Repository
	nodeRepo          node.NodeRepository
	logger            logger.Interface
}

// NewListExternalForwardRulesUseCase creates a new use case.
func NewListExternalForwardRulesUseCase(
	repo externalforward.Repository,
	subscriptionRepo subscription.SubscriptionRepository,
	resourceGroupRepo resource.Repository,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *ListExternalForwardRulesUseCase {
	return &ListExternalForwardRulesUseCase{
		repo:              repo,
		subscriptionRepo:  subscriptionRepo,
		resourceGroupRepo: resourceGroupRepo,
		nodeRepo:          nodeRepo,
		logger:            logger,
	}
}

// Execute lists external forward rules for a subscription.
// Resource group priority mode: if the plan has active resource groups with rules, return those;
// otherwise, return direct subscription-bound rules.
func (uc *ListExternalForwardRulesUseCase) Execute(ctx context.Context, query ListExternalForwardRulesQuery) (*ListExternalForwardRulesResult, error) {
	uc.logger.Infow("executing list external forward rules use case",
		"subscription_id", query.SubscriptionID,
		"page", query.Page,
		"page_size", query.PageSize,
	)

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
		uc.logger.Errorw("failed to list external forward rules", "subscription_id", query.SubscriptionID, "error", err)
		return nil, fmt.Errorf("failed to list external forward rules: %w", err)
	}

	// Apply filters
	filteredRules := uc.applyFilters(rules, query)

	// Sort rules
	uc.sortRules(filteredRules, query.OrderBy, query.Order)

	// Calculate total before pagination
	total := int64(len(filteredRules))

	// Apply pagination
	start := (query.Page - 1) * query.PageSize
	end := start + query.PageSize
	if start > len(filteredRules) {
		filteredRules = []*externalforward.ExternalForwardRule{}
	} else if end > len(filteredRules) {
		filteredRules = filteredRules[start:]
	} else {
		filteredRules = filteredRules[start:end]
	}

	// Collect node IDs and batch fetch node info
	nodeIDToInfo := make(map[uint]*dto.NodeInfo)
	var nodeIDs []uint
	for _, rule := range filteredRules {
		if rule.NodeID() != nil {
			nodeIDs = append(nodeIDs, *rule.NodeID())
		}
	}
	if len(nodeIDs) > 0 {
		nodes, err := uc.nodeRepo.GetByIDs(ctx, nodeIDs)
		if err != nil {
			uc.logger.Warnw("failed to get nodes", "error", err)
			// Continue without node info rather than failing the request
		} else {
			for _, n := range nodes {
				info := &dto.NodeInfo{
					SID:           n.SID(),
					ServerAddress: n.ServerAddress().Value(),
				}
				if n.PublicIPv4() != nil {
					info.PublicIPv4 = *n.PublicIPv4()
				}
				if n.PublicIPv6() != nil {
					info.PublicIPv6 = *n.PublicIPv6()
				}
				nodeIDToInfo[n.ID()] = info
			}
		}
	}

	return &ListExternalForwardRulesResult{
		Rules: dto.FromDomainList(filteredRules, query.SubscriptionSID, nodeIDToInfo),
		Total: total,
	}, nil
}

// getRulesWithResourceGroupPriority implements resource group priority mode:
// 1. Get subscription's planID
// 2. Get active resource groups for the plan
// 3. If resource groups have rules, return those rules
// 4. Otherwise, return direct subscription-bound rules
func (uc *ListExternalForwardRulesUseCase) getRulesWithResourceGroupPriority(ctx context.Context, subscriptionID uint) ([]*externalforward.ExternalForwardRule, error) {
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
	var allRules []*externalforward.ExternalForwardRule
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

// applyFilters applies query filters to the rules list.
func (uc *ListExternalForwardRulesUseCase) applyFilters(rules []*externalforward.ExternalForwardRule, query ListExternalForwardRulesQuery) []*externalforward.ExternalForwardRule {
	if query.Status == "" {
		return rules
	}

	var filtered []*externalforward.ExternalForwardRule
	for _, rule := range rules {
		// Filter by status
		if query.Status != "" && rule.Status().String() != query.Status {
			continue
		}
		filtered = append(filtered, rule)
	}
	return filtered
}

// sortRules sorts the rules based on orderBy and order.
func (uc *ListExternalForwardRulesUseCase) sortRules(rules []*externalforward.ExternalForwardRule, orderBy, order string) {
	// Normalize order
	ascending := true
	if strings.ToUpper(order) == "DESC" {
		ascending = false
	}

	sort.Slice(rules, func(i, j int) bool {
		var less bool
		switch orderBy {
		case "name":
			less = rules[i].Name() < rules[j].Name()
		case "created_at":
			less = rules[i].CreatedAt().Before(rules[j].CreatedAt())
		case "updated_at":
			less = rules[i].UpdatedAt().Before(rules[j].UpdatedAt())
		case "status":
			less = rules[i].Status().String() < rules[j].Status().String()
		default: // sort_order
			if rules[i].SortOrder() == rules[j].SortOrder() {
				less = rules[i].ID() < rules[j].ID()
			} else {
				less = rules[i].SortOrder() < rules[j].SortOrder()
			}
		}

		if ascending {
			return less
		}
		return !less
	})
}

// GetExternalForwardRuleQuery represents the input for getting a single external forward rule.
type GetExternalForwardRuleQuery struct {
	SID             string
	SubscriptionID  uint
	SubscriptionSID string
}

// GetExternalForwardRuleResult represents the output of getting a single external forward rule.
type GetExternalForwardRuleResult struct {
	Rule *dto.ExternalForwardRuleDTO `json:"rule"`
}

// GetExternalForwardRuleUseCase handles getting a single external forward rule.
type GetExternalForwardRuleUseCase struct {
	repo              externalforward.Repository
	subscriptionRepo  subscription.SubscriptionRepository
	resourceGroupRepo resource.Repository
	nodeRepo          node.NodeRepository
	logger            logger.Interface
}

// NewGetExternalForwardRuleUseCase creates a new use case.
func NewGetExternalForwardRuleUseCase(
	repo externalforward.Repository,
	subscriptionRepo subscription.SubscriptionRepository,
	resourceGroupRepo resource.Repository,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *GetExternalForwardRuleUseCase {
	return &GetExternalForwardRuleUseCase{
		repo:              repo,
		subscriptionRepo:  subscriptionRepo,
		resourceGroupRepo: resourceGroupRepo,
		nodeRepo:          nodeRepo,
		logger:            logger,
	}
}

// Execute gets a single external forward rule by SID.
// The rule must either be directly bound to the subscription, or accessible via resource groups.
func (uc *GetExternalForwardRuleUseCase) Execute(ctx context.Context, query GetExternalForwardRuleQuery) (*GetExternalForwardRuleResult, error) {
	uc.logger.Infow("executing get external forward rule use case", "sid", query.SID)

	rule, err := uc.repo.GetBySID(ctx, query.SID)
	if err != nil {
		return nil, err
	}

	// Helper function to get node info
	getNodeInfo := func() *dto.NodeInfo {
		if rule.NodeID() == nil {
			return nil
		}
		nodes, err := uc.nodeRepo.GetByIDs(ctx, []uint{*rule.NodeID()})
		if err != nil || len(nodes) == 0 {
			uc.logger.Warnw("failed to get node", "node_id", *rule.NodeID(), "error", err)
			return nil
		}
		n := nodes[0]
		info := &dto.NodeInfo{
			SID:           n.SID(),
			ServerAddress: n.ServerAddress().Value(),
		}
		if n.PublicIPv4() != nil {
			info.PublicIPv4 = *n.PublicIPv4()
		}
		if n.PublicIPv6() != nil {
			info.PublicIPv6 = *n.PublicIPv6()
		}
		return info
	}

	// Check if rule belongs directly to the subscription
	if rule.SubscriptionID() != nil && *rule.SubscriptionID() == query.SubscriptionID {
		return &GetExternalForwardRuleResult{
			Rule: dto.FromDomain(rule, query.SubscriptionSID, getNodeInfo()),
		}, nil
	}

	// Check if rule is accessible via resource groups
	if len(rule.GroupIDs()) > 0 {
		accessible, err := uc.isRuleAccessibleViaResourceGroups(ctx, rule, query.SubscriptionID)
		if err != nil {
			uc.logger.Warnw("failed to check resource group access", "error", err)
			// Continue to deny access on error
		} else if accessible {
			return &GetExternalForwardRuleResult{
				Rule: dto.FromDomain(rule, query.SubscriptionSID, getNodeInfo()),
			}, nil
		}
	}

	// Rule not accessible
	uc.logger.Warnw("external forward rule not accessible by subscription",
		"rule_sid", query.SID,
		"rule_subscription_id", rule.SubscriptionID(),
		"rule_group_ids", rule.GroupIDs(),
		"requested_subscription_id", query.SubscriptionID,
	)
	return nil, errors.NewNotFoundError("external forward rule", query.SID)
}

// isRuleAccessibleViaResourceGroups checks if the rule is accessible to the subscription via resource groups.
func (uc *GetExternalForwardRuleUseCase) isRuleAccessibleViaResourceGroups(ctx context.Context, rule *externalforward.ExternalForwardRule, subscriptionID uint) (bool, error) {
	// Get subscription to find planID
	sub, err := uc.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return false, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get resource groups for the plan
	groups, err := uc.resourceGroupRepo.GetByPlanID(ctx, sub.PlanID())
	if err != nil {
		return false, fmt.Errorf("failed to get resource groups: %w", err)
	}

	// Build a set of active group IDs for the subscription's plan
	activeGroupIDs := make(map[uint]bool)
	for _, g := range groups {
		if g.Status() == resource.GroupStatusActive {
			activeGroupIDs[g.ID()] = true
		}
	}

	// Check if any of the rule's group IDs match the subscription's active groups
	for _, ruleGroupID := range rule.GroupIDs() {
		if activeGroupIDs[ruleGroupID] {
			return true, nil
		}
	}

	return false, nil
}
