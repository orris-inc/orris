package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils/setutil"
)

// ManageResourceGroupForwardRulesUseCase handles adding/removing forward rules from resource groups
type ManageResourceGroupForwardRulesUseCase struct {
	resourceGroupRepo resource.Repository
	ruleRepo          forward.Repository
	planRepo          subscription.PlanRepository
	logger            logger.Interface
}

// NewManageResourceGroupForwardRulesUseCase creates a new ManageResourceGroupForwardRulesUseCase
func NewManageResourceGroupForwardRulesUseCase(
	resourceGroupRepo resource.Repository,
	ruleRepo forward.Repository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *ManageResourceGroupForwardRulesUseCase {
	return &ManageResourceGroupForwardRulesUseCase{
		resourceGroupRepo: resourceGroupRepo,
		ruleRepo:          ruleRepo,
		planRepo:          planRepo,
		logger:            logger,
	}
}

// AddRules adds forward rules to a resource group by its internal ID
func (uc *ManageResourceGroupForwardRulesUseCase) AddRules(ctx context.Context, groupID uint, ruleSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeAddRules(ctx, group, ruleSIDs)
}

// AddRulesBySID adds forward rules to a resource group by its Stripe-style SID
func (uc *ManageResourceGroupForwardRulesUseCase) AddRulesBySID(ctx context.Context, groupSID string, ruleSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "group_sid", groupSID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeAddRules(ctx, group, ruleSIDs)
}

// executeAddRules performs the actual add rules logic
func (uc *ManageResourceGroupForwardRulesUseCase) executeAddRules(ctx context.Context, group *resource.ResourceGroup, ruleSIDs []string) (*dto.BatchOperationResult, error) {
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	// Verify the plan type supports forward rules binding (node and hybrid only, not forward)
	plan, err := uc.planRepo.GetByID(ctx, group.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get plan", "error", err, "plan_id", group.PlanID())
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found for resource group")
	}
	if plan.PlanType().IsForward() {
		uc.logger.Warnw("attempted to add forward rules to forward plan resource group",
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
		uc.logger.Errorw("failed to batch get forward rules", "error", err)
		return nil, fmt.Errorf("failed to get forward rules: %w", err)
	}

	// Collect valid rule IDs for batch update
	validRuleIDs := make([]uint, 0, len(ruleSIDs))
	sidToID := make(map[uint]string) // For mapping back to SIDs

	for _, ruleSID := range ruleSIDs {
		// Validate the SID format (fr_xxx)
		if err := id.ValidatePrefix(ruleSID, id.PrefixForwardRule); err != nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     ruleSID,
				Reason: "invalid forward rule ID format",
			})
			continue
		}

		rule, exists := rulesMap[ruleSID]
		if !exists || rule == nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     ruleSID,
				Reason: "forward rule not found",
			})
			continue
		}

		validRuleIDs = append(validRuleIDs, rule.ID())
		sidToID[rule.ID()] = ruleSID
	}

	// Batch add group ID to all valid rules
	if len(validRuleIDs) > 0 {
		_, err := uc.ruleRepo.BatchAddGroupID(ctx, validRuleIDs, groupID)
		if err != nil {
			uc.logger.Errorw("failed to batch add group ID to forward rules", "error", err, "group_id", groupID)
			// Mark all as failed
			for _, ruleID := range validRuleIDs {
				result.Failed = append(result.Failed, dto.BatchOperationErr{
					ID:     sidToID[ruleID],
					Reason: "failed to update forward rule",
				})
			}
		} else {
			// All valid rules succeeded (including those that already had the group ID)
			for _, ruleID := range validRuleIDs {
				result.Succeeded = append(result.Succeeded, sidToID[ruleID])
			}
		}
	}

	uc.logger.Infow("added forward rules to resource group",
		"group_id", groupID,
		"group_sid", group.SID(),
		"succeeded_count", len(result.Succeeded),
		"failed_count", len(result.Failed))

	return result, nil
}

// RemoveRules removes forward rules from a resource group by its internal ID
func (uc *ManageResourceGroupForwardRulesUseCase) RemoveRules(ctx context.Context, groupID uint, ruleSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeRemoveRules(ctx, group, ruleSIDs)
}

// RemoveRulesBySID removes forward rules from a resource group by its Stripe-style SID
func (uc *ManageResourceGroupForwardRulesUseCase) RemoveRulesBySID(ctx context.Context, groupSID string, ruleSIDs []string) (*dto.BatchOperationResult, error) {
	group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "group_sid", groupSID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeRemoveRules(ctx, group, ruleSIDs)
}

// executeRemoveRules performs the actual remove rules logic
func (uc *ManageResourceGroupForwardRulesUseCase) executeRemoveRules(ctx context.Context, group *resource.ResourceGroup, ruleSIDs []string) (*dto.BatchOperationResult, error) {
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
		uc.logger.Errorw("failed to batch get forward rules", "error", err)
		return nil, fmt.Errorf("failed to get forward rules: %w", err)
	}

	// Collect valid rule IDs for batch update
	// Only include rules that actually belong to this group
	validRuleIDs := make([]uint, 0, len(ruleSIDs))
	sidToID := make(map[uint]string) // For mapping back to SIDs

	for _, ruleSID := range ruleSIDs {
		// Validate the SID format (fr_xxx)
		if err := id.ValidatePrefix(ruleSID, id.PrefixForwardRule); err != nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     ruleSID,
				Reason: "invalid forward rule ID format",
			})
			continue
		}

		rule, exists := rulesMap[ruleSID]
		if !exists || rule == nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     ruleSID,
				Reason: "forward rule not found",
			})
			continue
		}

		// Check if the rule belongs to this group before adding to batch
		if !rule.HasGroupID(groupID) {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     ruleSID,
				Reason: "forward rule does not belong to this group",
			})
			continue
		}

		validRuleIDs = append(validRuleIDs, rule.ID())
		sidToID[rule.ID()] = ruleSID
	}

	// Batch remove group ID from all valid rules
	if len(validRuleIDs) > 0 {
		_, err := uc.ruleRepo.BatchRemoveGroupID(ctx, validRuleIDs, groupID)
		if err != nil {
			uc.logger.Errorw("failed to batch remove group ID from forward rules", "error", err, "group_id", groupID)
			// Mark all as failed
			for _, ruleID := range validRuleIDs {
				result.Failed = append(result.Failed, dto.BatchOperationErr{
					ID:     sidToID[ruleID],
					Reason: "failed to update forward rule",
				})
			}
		} else {
			// All valid rules succeeded
			for _, ruleID := range validRuleIDs {
				result.Succeeded = append(result.Succeeded, sidToID[ruleID])
			}
		}
	}

	uc.logger.Infow("removed forward rules from resource group",
		"group_id", groupID,
		"group_sid", group.SID(),
		"succeeded_count", len(result.Succeeded),
		"failed_count", len(result.Failed))

	return result, nil
}

// ListRules lists all forward rules in a resource group with pagination by its internal ID
func (uc *ManageResourceGroupForwardRulesUseCase) ListRules(ctx context.Context, groupID uint, page, pageSize int, orderBy, order string) (*dto.ListGroupForwardRulesResponse, error) {
	group, err := uc.resourceGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeListRules(ctx, group, page, pageSize, orderBy, order)
}

// ListRulesBySID lists all forward rules in a resource group with pagination by its Stripe-style SID
func (uc *ManageResourceGroupForwardRulesUseCase) ListRulesBySID(ctx context.Context, groupSID string, page, pageSize int, orderBy, order string) (*dto.ListGroupForwardRulesResponse, error) {
	group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "group_sid", groupSID)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeListRules(ctx, group, page, pageSize, orderBy, order)
}

// executeListRules performs the actual list rules logic with database-level pagination
func (uc *ManageResourceGroupForwardRulesUseCase) executeListRules(ctx context.Context, group *resource.ResourceGroup, page, pageSize int, orderBy, order string) (*dto.ListGroupForwardRulesResponse, error) {
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	groupID := group.ID()

	// Use List method with GroupIDs filter to leverage existing sorting logic
	filter := forward.ListFilter{
		Page:     page,
		PageSize: pageSize,
		GroupIDs: []uint{groupID},
		OrderBy:  orderBy,
		Order:    order,
	}
	rules, total, err := uc.ruleRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list forward rules", "error", err, "group_id", groupID)
		return nil, fmt.Errorf("failed to list forward rules: %w", err)
	}

	// Collect all unique group IDs from the rules for batch lookup
	groupIDSet := setutil.NewUintSet()
	for _, rule := range rules {
		groupIDSet.AddAll(rule.GroupIDs())
	}

	// Batch fetch all resource groups to avoid N+1 queries
	groupIDs := groupIDSet.ToSlice()

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
	items := make([]dto.ForwardRuleSummaryResponse, 0, len(rules))
	for _, rule := range rules {
		groupSIDs := make([]string, 0, len(rule.GroupIDs()))
		for _, gid := range rule.GroupIDs() {
			if sid, ok := groupIDToSID[gid]; ok && sid != "" {
				groupSIDs = append(groupSIDs, sid)
			}
		}
		items = append(items, dto.ForwardRuleSummaryResponse{
			ID:         rule.SID(),
			Name:       rule.Name(),
			Status:     rule.Status().String(),
			Protocol:   rule.Protocol().String(),
			ListenPort: rule.ListenPort(),
			SortOrder:  rule.SortOrder(),
			GroupSIDs:  groupSIDs,
			CreatedAt:  rule.CreatedAt(),
		})
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &dto.ListGroupForwardRulesResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}
