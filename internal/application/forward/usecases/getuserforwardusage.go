package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetUserForwardUsageQuery represents the input for getting user forward usage.
type GetUserForwardUsageQuery struct {
	UserID uint
}

// GetUserForwardUsageResult represents the user's forward rule usage and quota information.
type GetUserForwardUsageResult struct {
	RuleCount    int      `json:"rule_count"`
	RuleLimit    int      `json:"rule_limit"`
	TrafficUsed  uint64   `json:"traffic_used"`  // in bytes
	TrafficLimit uint64   `json:"traffic_limit"` // in bytes, 0 means unlimited
	AllowedTypes []string `json:"allowed_types"`
}

// GetUserForwardUsageUseCase handles getting user forward usage.
type GetUserForwardUsageUseCase struct {
	repo             forward.Repository
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	usageRepo        subscription.SubscriptionUsageRepository
	usageStatsRepo   subscription.SubscriptionUsageStatsRepository
	hourlyCache      cache.HourlyTrafficCache
	logger           logger.Interface
}

// NewGetUserForwardUsageUseCase creates a new GetUserForwardUsageUseCase.
func NewGetUserForwardUsageUseCase(
	repo forward.Repository,
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	usageRepo subscription.SubscriptionUsageRepository,
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyCache cache.HourlyTrafficCache,
	logger logger.Interface,
) *GetUserForwardUsageUseCase {
	return &GetUserForwardUsageUseCase{
		repo:             repo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		usageRepo:        usageRepo,
		usageStatsRepo:   usageStatsRepo,
		hourlyCache:      hourlyCache,
		logger:           logger,
	}
}

// Execute retrieves forward rule usage statistics for a user.
func (uc *GetUserForwardUsageUseCase) Execute(ctx context.Context, query GetUserForwardUsageQuery) (*GetUserForwardUsageResult, error) {
	uc.logger.Infow("executing get user forward usage use case", "user_id", query.UserID)

	// Validate user ID
	if query.UserID == 0 {
		return nil, errors.NewValidationError("user_id is required")
	}

	// Get user's active subscriptions
	subscriptions, err := uc.subscriptionRepo.GetActiveByUserID(ctx, query.UserID)
	if err != nil {
		uc.logger.Errorw("failed to get active subscriptions", "user_id", query.UserID, "error", err)
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}

	// Initialize limits - use flags to track unlimited state
	// maxRuleLimit: 0 means unlimited, >0 means limited
	maxRuleLimit := 0
	hasUnlimitedRules := false
	// maxTrafficLimit: 0 means unlimited, >0 means limited
	var maxTrafficLimit uint64
	hasUnlimitedTraffic := false
	allowedTypesSet := make(map[string]bool)

	// Collect forward subscription IDs and their period ranges for traffic query
	var forwardSubscriptionIDs []uint
	var earliestFrom, latestTo time.Time
	firstSub := true

	// Find the highest limits among all active subscriptions
	for _, sub := range subscriptions {
		plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
		if err != nil {
			uc.logger.Warnw("failed to get plan for subscription", "subscription_id", sub.ID(), "plan_id", sub.PlanID(), "error", err)
			continue
		}

		if plan == nil {
			continue
		}

		// Check if plan is forward type
		if !plan.PlanType().IsForward() {
			continue
		}

		// Collect forward subscription ID and period range
		forwardSubscriptionIDs = append(forwardSubscriptionIDs, sub.ID())
		periodStart := sub.CurrentPeriodStart()
		periodEnd := sub.CurrentPeriodEnd()
		if firstSub || periodStart.Before(earliestFrom) {
			earliestFrom = periodStart
		}
		if firstSub || periodEnd.After(latestTo) {
			latestTo = periodEnd
		}
		firstSub = false

		planFeatures := plan.Features()
		if planFeatures == nil {
			continue
		}

		// Get rule limit using unified key
		limit, err := planFeatures.GetRuleLimit()
		if err != nil {
			uc.logger.Warnw("failed to get rule limit", "subscription_id", sub.ID(), "error", err)
			continue
		}

		// 0 means unlimited - once found, it cannot be overridden
		if limit == 0 {
			hasUnlimitedRules = true
		} else if !hasUnlimitedRules && limit > maxRuleLimit {
			// Only update if no unlimited found and this limit is higher
			maxRuleLimit = limit
		}

		// Get traffic limit using unified key
		trafficLimit, err := planFeatures.GetTrafficLimit()
		if err != nil {
			uc.logger.Warnw("failed to get traffic limit", "subscription_id", sub.ID(), "error", err)
			continue
		}

		// 0 means unlimited - once found, it cannot be overridden
		if trafficLimit == 0 {
			hasUnlimitedTraffic = true
		} else if !hasUnlimitedTraffic && trafficLimit > maxTrafficLimit {
			// Only update if no unlimited found and this limit is higher
			maxTrafficLimit = trafficLimit
		}

		// Collect allowed rule types using unified key
		types, err := planFeatures.GetRuleTypes()
		if err != nil {
			uc.logger.Warnw("failed to get rule types", "subscription_id", sub.ID(), "error", err)
			continue
		}

		// Empty means all types allowed
		if len(types) == 0 {
			allowedTypesSet["direct"] = true
			allowedTypesSet["entry"] = true
			allowedTypesSet["chain"] = true
			allowedTypesSet["direct_chain"] = true
		} else {
			for _, t := range types {
				allowedTypesSet[t] = true
			}
		}
	}

	// Apply unlimited flags - 0 represents unlimited in the result
	if hasUnlimitedRules {
		maxRuleLimit = 0
	}
	if hasUnlimitedTraffic {
		maxTrafficLimit = 0
	}

	// Get rule count only (no need to fetch all rules)
	_, ruleCount, err := uc.repo.ListByUserID(ctx, query.UserID, forward.ListFilter{
		Page:     1,
		PageSize: 1, // Only need count, not actual rules
	})
	if err != nil {
		uc.logger.Errorw("failed to count user forward rules", "user_id", query.UserID, "error", err)
		return nil, fmt.Errorf("failed to get user rules count: %w", err)
	}

	// Query traffic usage from Redis (last 24h) + MySQL stats (historical)
	var trafficUsed uint64
	if len(forwardSubscriptionIDs) > 0 {
		// Adjust latestTo to end of day
		latestTo = biztime.EndOfDayUTC(latestTo)
		resourceType := string(subscription.ResourceTypeForwardRule)

		now := biztime.NowUTC()
		dayAgo := now.Add(-24 * time.Hour)

		// Determine time boundary for recent data (last 24 hours)
		recentFrom := earliestFrom
		if recentFrom.Before(dayAgo) {
			recentFrom = dayAgo
		}

		// Get recent traffic from Redis (last 24h)
		if recentFrom.Before(latestTo) {
			recentTraffic, err := uc.hourlyCache.GetTotalTrafficBySubscriptionIDs(
				ctx, forwardSubscriptionIDs, resourceType, recentFrom, latestTo,
			)
			if err != nil {
				uc.logger.Warnw("failed to get recent traffic from Redis", "error", err)
			} else {
				for _, t := range recentTraffic {
					trafficUsed += t.Total
				}
			}
		}

		// Get historical traffic from MySQL stats (before 24h ago)
		if earliestFrom.Before(dayAgo) {
			historicalTo := dayAgo
			if historicalTo.After(latestTo) {
				historicalTo = latestTo
			}
			historicalTraffic, err := uc.usageStatsRepo.GetTotalBySubscriptionIDs(
				ctx, forwardSubscriptionIDs, &resourceType, subscription.GranularityDaily, earliestFrom, historicalTo,
			)
			if err != nil {
				uc.logger.Warnw("failed to get historical traffic from stats", "error", err)
			} else if historicalTraffic != nil {
				trafficUsed += historicalTraffic.Total
			}
		}
	}

	// Convert allowed types set to slice
	allowedTypes := make([]string, 0, len(allowedTypesSet))
	for t := range allowedTypesSet {
		allowedTypes = append(allowedTypes, t)
	}

	result := &GetUserForwardUsageResult{
		RuleCount:    int(ruleCount),
		RuleLimit:    maxRuleLimit,
		TrafficUsed:  trafficUsed,
		TrafficLimit: maxTrafficLimit,
		AllowedTypes: allowedTypes,
	}

	uc.logger.Infow("user forward usage retrieved successfully",
		"user_id", query.UserID,
		"rule_count", ruleCount,
		"rule_limit", maxRuleLimit,
		"traffic_used", trafficUsed,
		"traffic_limit", maxTrafficLimit,
		"allowed_types", allowedTypes,
	)

	return result, nil
}
