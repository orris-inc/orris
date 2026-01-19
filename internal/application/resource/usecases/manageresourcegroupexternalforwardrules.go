package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/domain/externalforward"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ManageResourceGroupExternalForwardRulesUseCase handles adding/removing external forward rules from resource groups
type ManageResourceGroupExternalForwardRulesUseCase struct {
	resourceGroupRepo resource.Repository
	ruleRepo          externalforward.Repository
	planRepo          subscription.PlanRepository
	logger            logger.Interface
}

// NewManageResourceGroupExternalForwardRulesUseCase creates a new ManageResourceGroupExternalForwardRulesUseCase
func NewManageResourceGroupExternalForwardRulesUseCase(
	resourceGroupRepo resource.Repository,
	ruleRepo externalforward.Repository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *ManageResourceGroupExternalForwardRulesUseCase {
	return &ManageResourceGroupExternalForwardRulesUseCase{
		resourceGroupRepo: resourceGroupRepo,
		ruleRepo:          ruleRepo,
		planRepo:          planRepo,
		logger:            logger,
	}
}

// AddRules adds external forward rules to a resource group by its internal ID
func (uc *ManageResourceGroupExternalForwardRulesUseCase) AddRules(ctx context.Context, groupID uint, ruleSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeAddRules(ctx, group, ruleSIDs)
}

// AddRulesBySID adds external forward rules to a resource group by its Stripe-style SID
func (uc *ManageResourceGroupExternalForwardRulesUseCase) AddRulesBySID(ctx context.Context, groupSID string, ruleSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "group_sid", groupSID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeAddRules(ctx, group, ruleSIDs)
}

// executeAddRules performs the actual add rules logic
func (uc *ManageResourceGroupExternalForwardRulesUseCase) executeAddRules(ctx context.Context, group *resource.ResourceGroup, ruleSIDs []string) (*dto.BatchOperationResult, error) {
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	// Verify the plan type supports external forward rules binding (node and hybrid only, not forward)
	plan, err := uc.planRepo.GetByID(ctx, group.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get plan", "error", err, "plan_id", group.PlanID())
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found for resource group")
	}
	if plan.PlanType().IsForward() {
		uc.logger.Warnw("attempted to add external forward rules to forward plan resource group",
			"group_id", group.ID(),
			"plan_id", group.PlanID(),
			"plan_type", plan.PlanType().String())
		return nil, resource.ErrForwardPlanCannotBindRules
	}

	groupID := group.ID()

	result := &dto.BatchOperationResult{
		Succeeded: make([]string, 0),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	// Batch fetch all rules by SIDs to reduce N+1 queries
	rulesMap, err := uc.ruleRepo.GetBySIDs(ctx, ruleSIDs)
	if err != nil {
		uc.logger.Errorw("failed to batch get external forward rules", "error", err)
		return nil, fmt.Errorf("failed to get external forward rules: %w", err)
	}

	for _, ruleSID := range ruleSIDs {
		// Validate the SID format (efr_xxx)
		if err := id.ValidatePrefix(ruleSID, id.PrefixExternalForwardRule); err != nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     ruleSID,
				Reason: "invalid external forward rule ID format",
			})
			continue
		}

		rule, exists := rulesMap[ruleSID]
		if !exists || rule == nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     ruleSID,
				Reason: "external forward rule not found",
			})
			continue
		}

		// Use atomic operation to add group ID
		added, err := uc.ruleRepo.AddGroupIDAtomically(ctx, rule.ID(), groupID)
		if err != nil {
			uc.logger.Errorw("failed to add group ID to external forward rule atomically", "error", err, "rule_sid", ruleSID, "group_id", groupID)
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     ruleSID,
				Reason: "failed to update external forward rule",
			})
			continue
		}

		// added=false means it already had this group ID, still count as success
		_ = added
		result.Succeeded = append(result.Succeeded, ruleSID)
	}

	uc.logger.Infow("added external forward rules to resource group",
		"group_id", groupID,
		"group_sid", group.SID(),
		"succeeded_count", len(result.Succeeded),
		"failed_count", len(result.Failed))

	return result, nil
}

// RemoveRules removes external forward rules from a resource group by its internal ID
func (uc *ManageResourceGroupExternalForwardRulesUseCase) RemoveRules(ctx context.Context, groupID uint, ruleSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeRemoveRules(ctx, group, ruleSIDs)
}

// RemoveRulesBySID removes external forward rules from a resource group by its Stripe-style SID
func (uc *ManageResourceGroupExternalForwardRulesUseCase) RemoveRulesBySID(ctx context.Context, groupSID string, ruleSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "group_sid", groupSID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeRemoveRules(ctx, group, ruleSIDs)
}

// executeRemoveRules performs the actual remove rules logic
func (uc *ManageResourceGroupExternalForwardRulesUseCase) executeRemoveRules(ctx context.Context, group *resource.ResourceGroup, ruleSIDs []string) (*dto.BatchOperationResult, error) {
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	groupID := group.ID()

	result := &dto.BatchOperationResult{
		Succeeded: make([]string, 0),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	// Batch fetch all rules by SIDs to reduce N+1 queries
	rulesMap, err := uc.ruleRepo.GetBySIDs(ctx, ruleSIDs)
	if err != nil {
		uc.logger.Errorw("failed to batch get external forward rules", "error", err)
		return nil, fmt.Errorf("failed to get external forward rules: %w", err)
	}

	for _, ruleSID := range ruleSIDs {
		// Validate the SID format (efr_xxx)
		if err := id.ValidatePrefix(ruleSID, id.PrefixExternalForwardRule); err != nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     ruleSID,
				Reason: "invalid external forward rule ID format",
			})
			continue
		}

		rule, exists := rulesMap[ruleSID]
		if !exists || rule == nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     ruleSID,
				Reason: "external forward rule not found",
			})
			continue
		}

		// Use atomic operation to remove group ID
		removed, err := uc.ruleRepo.RemoveGroupIDAtomically(ctx, rule.ID(), groupID)
		if err != nil {
			uc.logger.Errorw("failed to remove group ID from external forward rule atomically", "error", err, "rule_sid", ruleSID, "group_id", groupID)
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     ruleSID,
				Reason: "failed to update external forward rule",
			})
			continue
		}

		if !removed {
			// The rule did not have this group ID
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     ruleSID,
				Reason: "external forward rule does not belong to this group",
			})
			continue
		}

		result.Succeeded = append(result.Succeeded, ruleSID)
	}

	uc.logger.Infow("removed external forward rules from resource group",
		"group_id", groupID,
		"group_sid", group.SID(),
		"succeeded_count", len(result.Succeeded),
		"failed_count", len(result.Failed))

	return result, nil
}

// ListRules lists all external forward rules in a resource group with pagination by its internal ID
func (uc *ManageResourceGroupExternalForwardRulesUseCase) ListRules(ctx context.Context, groupID uint, page, pageSize int) (*dto.ListGroupExternalForwardRulesResponse, error) {
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeListRules(ctx, group, page, pageSize)
}

// ListRulesBySID lists all external forward rules in a resource group with pagination by its Stripe-style SID
func (uc *ManageResourceGroupExternalForwardRulesUseCase) ListRulesBySID(ctx context.Context, groupSID string, page, pageSize int) (*dto.ListGroupExternalForwardRulesResponse, error) {
	group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "group_sid", groupSID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeListRules(ctx, group, page, pageSize)
}

// executeListRules performs the actual list rules logic with database-level pagination
func (uc *ManageResourceGroupExternalForwardRulesUseCase) executeListRules(ctx context.Context, group *resource.ResourceGroup, page, pageSize int) (*dto.ListGroupExternalForwardRulesResponse, error) {
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	groupID := group.ID()

	// Set pagination defaults
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	rules, total, err := uc.ruleRepo.ListByGroupID(ctx, groupID, page, pageSize)
	if err != nil {
		uc.logger.Errorw("failed to list external forward rules", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to list external forward rules: %w", err)
	}

	// Collect all unique group IDs from the rules for batch lookup
	groupIDSet := make(map[uint]bool)
	for _, rule := range rules {
		for _, gid := range rule.GroupIDs() {
			groupIDSet[gid] = true
		}
	}

	// Batch fetch all resource groups to avoid N+1 queries
	groupIDs := make([]uint, 0, len(groupIDSet))
	for gid := range groupIDSet {
		groupIDs = append(groupIDs, gid)
	}

	groupIDToSID := make(map[uint]string)
	if len(groupIDs) > 0 {
		groups, err := uc.resourceGroupRepo.GetByIDs(ctx, groupIDs)
		if err != nil {
			uc.logger.Warnw("failed to batch get resource groups", "error", err)
			// Continue without group SIDs rather than failing the whole request
		} else {
			for _, g := range groups {
				groupIDToSID[g.ID()] = g.SID()
			}
		}
	}

	// Convert to response DTOs
	items := make([]dto.ExternalForwardRuleSummaryResponse, 0, len(rules))
	for _, rule := range rules {
		groupSIDs := make([]string, 0, len(rule.GroupIDs()))
		for _, gid := range rule.GroupIDs() {
			if sid, ok := groupIDToSID[gid]; ok && sid != "" {
				groupSIDs = append(groupSIDs, sid)
			}
		}
		items = append(items, dto.ExternalForwardRuleSummaryResponse{
			ID:             rule.SID(),
			Name:           rule.Name(),
			Status:         rule.Status().String(),
			ServerAddress:  rule.ServerAddress(),
			ListenPort:     rule.ListenPort(),
			ExternalSource: rule.ExternalSource(),
			SortOrder:      rule.SortOrder(),
			GroupSIDs:      groupSIDs,
			CreatedAt:      rule.CreatedAt(),
		})
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &dto.ListGroupExternalForwardRulesResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}
