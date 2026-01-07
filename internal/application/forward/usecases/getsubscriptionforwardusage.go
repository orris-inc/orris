package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetSubscriptionForwardUsageQuery represents the input for getting subscription forward usage.
type GetSubscriptionForwardUsageQuery struct {
	SubscriptionID uint
}

// GetSubscriptionForwardUsageResult represents the subscription's forward rule usage and quota information.
type GetSubscriptionForwardUsageResult struct {
	RuleCount    int      `json:"rule_count"`
	RuleLimit    int      `json:"rule_limit"`
	TrafficUsed  uint64   `json:"traffic_used"`  // in bytes
	TrafficLimit uint64   `json:"traffic_limit"` // in bytes, 0 means unlimited
	AllowedTypes []string `json:"allowed_types"`
}

// GetSubscriptionForwardUsageUseCase handles getting subscription forward usage.
type GetSubscriptionForwardUsageUseCase struct {
	repo             forward.Repository
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	usageRepo        subscription.SubscriptionUsageRepository
	logger           logger.Interface
}

// NewGetSubscriptionForwardUsageUseCase creates a new GetSubscriptionForwardUsageUseCase.
func NewGetSubscriptionForwardUsageUseCase(
	repo forward.Repository,
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	usageRepo subscription.SubscriptionUsageRepository,
	logger logger.Interface,
) *GetSubscriptionForwardUsageUseCase {
	return &GetSubscriptionForwardUsageUseCase{
		repo:             repo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		usageRepo:        usageRepo,
		logger:           logger,
	}
}

// Execute retrieves forward rule usage statistics for a specific subscription.
func (uc *GetSubscriptionForwardUsageUseCase) Execute(ctx context.Context, query GetSubscriptionForwardUsageQuery) (*GetSubscriptionForwardUsageResult, error) {
	uc.logger.Infow("executing get subscription forward usage use case", "subscription_id", query.SubscriptionID)

	// Validate subscription ID
	if query.SubscriptionID == 0 {
		return nil, errors.NewValidationError("subscription_id is required")
	}

	// Get the subscription
	sub, err := uc.subscriptionRepo.GetByID(ctx, query.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "subscription_id", query.SubscriptionID, "error", err)
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return nil, errors.NewNotFoundError("subscription", fmt.Sprintf("%d", query.SubscriptionID))
	}

	// Get the subscription's plan
	plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get plan", "plan_id", sub.PlanID(), "error", err)
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return nil, errors.NewNotFoundError("plan", fmt.Sprintf("%d", sub.PlanID()))
	}

	// Check if plan is forward type
	if !plan.PlanType().IsForward() {
		uc.logger.Warnw("subscription plan does not support forward rules",
			"subscription_id", query.SubscriptionID,
			"plan_type", plan.PlanType().String(),
		)
		return nil, errors.NewValidationError("subscription plan does not support forward rules")
	}

	planFeatures := plan.Features()
	if planFeatures == nil {
		return nil, errors.NewValidationError("plan features not configured")
	}

	// Get rule limit
	ruleLimit, err := planFeatures.GetRuleLimit()
	if err != nil {
		uc.logger.Warnw("failed to get rule limit", "subscription_id", query.SubscriptionID, "error", err)
		ruleLimit = 0 // Default to unlimited on error
	}

	// Get traffic limit
	trafficLimit, err := planFeatures.GetTrafficLimit()
	if err != nil {
		uc.logger.Warnw("failed to get traffic limit", "subscription_id", query.SubscriptionID, "error", err)
		trafficLimit = 0 // Default to unlimited on error
	}

	// Get allowed rule types
	allowedTypes, err := planFeatures.GetRuleTypes()
	if err != nil {
		uc.logger.Warnw("failed to get rule types", "subscription_id", query.SubscriptionID, "error", err)
		allowedTypes = []string{"direct", "entry", "chain", "direct_chain"} // Default to all types on error
	}
	// Empty means all types allowed
	if len(allowedTypes) == 0 {
		allowedTypes = []string{"direct", "entry", "chain", "direct_chain"}
	}

	// Count current rules for this subscription
	ruleCount, err := uc.repo.CountBySubscriptionID(ctx, query.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to count subscription forward rules", "subscription_id", query.SubscriptionID, "error", err)
		return nil, fmt.Errorf("failed to get rule count: %w", err)
	}

	// Query traffic usage from subscription_usages table
	var trafficUsed uint64
	periodStart := sub.CurrentPeriodStart()
	periodEnd := biztime.EndOfDayUTC(sub.CurrentPeriodEnd())

	resourceType := string(subscription.ResourceTypeForwardRule)
	usageSummary, err := uc.usageRepo.GetTotalUsageBySubscriptionIDs(
		ctx, resourceType, []uint{query.SubscriptionID}, periodStart, periodEnd,
	)
	if err != nil {
		uc.logger.Warnw("failed to get forward traffic usage", "subscription_id", query.SubscriptionID, "error", err)
	} else if usageSummary != nil {
		trafficUsed = usageSummary.Total
	}

	result := &GetSubscriptionForwardUsageResult{
		RuleCount:    int(ruleCount),
		RuleLimit:    ruleLimit,
		TrafficUsed:  trafficUsed,
		TrafficLimit: trafficLimit,
		AllowedTypes: allowedTypes,
	}

	uc.logger.Infow("subscription forward usage retrieved successfully",
		"subscription_id", query.SubscriptionID,
		"rule_count", ruleCount,
		"rule_limit", ruleLimit,
		"traffic_used", trafficUsed,
		"traffic_limit", trafficLimit,
		"allowed_types", allowedTypes,
	)

	return result, nil
}
