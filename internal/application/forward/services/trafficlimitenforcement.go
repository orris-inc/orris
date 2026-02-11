package services

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// TrafficLimitEnforcementService enforces traffic limits for user forward rules.
// When a user's total forward traffic exceeds their subscription plan limit,
// this service automatically disables all of their forward rules.
type TrafficLimitEnforcementService struct {
	forwardRuleRepo       forward.Repository
	subscriptionRepo      subscription.SubscriptionRepository
	subscriptionUsageRepo subscription.SubscriptionUsageRepository // Keep for backward compatibility but won't be used
	usageStatsRepo        subscription.SubscriptionUsageStatsRepository
	hourlyTrafficCache    cache.HourlyTrafficCache
	planRepo              subscription.PlanRepository
	logger                logger.Interface
}

// NewTrafficLimitEnforcementService creates a new traffic limit enforcement service.
func NewTrafficLimitEnforcementService(
	forwardRuleRepo forward.Repository,
	subscriptionRepo subscription.SubscriptionRepository,
	subscriptionUsageRepo subscription.SubscriptionUsageRepository,
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyTrafficCache cache.HourlyTrafficCache,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *TrafficLimitEnforcementService {
	return &TrafficLimitEnforcementService{
		forwardRuleRepo:       forwardRuleRepo,
		subscriptionRepo:      subscriptionRepo,
		subscriptionUsageRepo: subscriptionUsageRepo,
		usageStatsRepo:        usageStatsRepo,
		hourlyTrafficCache:    hourlyTrafficCache,
		planRepo:              planRepo,
		logger:                logger,
	}
}

// CheckAndEnforceLimit checks if a user has exceeded their traffic limit
// and disables all forward rules if necessary.
// Returns an error if the check fails, but not if rules are disabled successfully.
func (s *TrafficLimitEnforcementService) CheckAndEnforceLimit(ctx context.Context, userID uint) error {
	s.logger.Debugw("checking traffic limit enforcement",
		"user_id", userID,
	)

	// Get user's active subscriptions
	activeSubscriptions, err := s.subscriptionRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		s.logger.Errorw("failed to get active subscriptions for user",
			"user_id", userID,
			"error", err,
		)
		return fmt.Errorf("failed to get active subscriptions: %w", err)
	}

	// If no active subscriptions, don't enforce limits
	if len(activeSubscriptions) == 0 {
		s.logger.Debugw("no active subscriptions found, skipping limit enforcement",
			"user_id", userID,
		)
		return nil
	}

	// Find the highest traffic limit across all Forward-type subscriptions
	// and collect their subscription IDs for traffic query
	trafficLimit, hasLimit, forwardSubscriptionIDs, err := s.getHighestTrafficLimitAndIDs(ctx, activeSubscriptions)
	if err != nil {
		s.logger.Errorw("failed to determine traffic limit",
			"user_id", userID,
			"error", err,
		)
		return fmt.Errorf("failed to determine traffic limit: %w", err)
	}

	// If no Forward-type subscriptions found, don't enforce
	if len(forwardSubscriptionIDs) == 0 {
		s.logger.Debugw("no forward subscriptions found, skipping limit enforcement",
			"user_id", userID,
		)
		return nil
	}

	// If any subscription has unlimited traffic (limit = 0), don't enforce
	if !hasLimit {
		s.logger.Debugw("user has unlimited traffic subscription, skipping limit enforcement",
			"user_id", userID,
		)
		return nil
	}

	// Get user's total forward traffic by combining Redis (recent 24h) and MySQL (historical)
	usedTraffic, err := s.getTotalTrafficForSubscriptions(ctx, forwardSubscriptionIDs)
	if err != nil {
		s.logger.Errorw("failed to get total traffic for user",
			"user_id", userID,
			"subscription_ids", forwardSubscriptionIDs,
			"error", err,
		)
		return fmt.Errorf("failed to get total traffic: %w", err)
	}
	if usedTraffic <= trafficLimit {
		s.logger.Debugw("traffic within limit",
			"user_id", userID,
			"traffic_used", usedTraffic,
			"traffic_limit", trafficLimit,
		)
		return nil
	}

	// Traffic exceeded - disable all user's enabled forward rules
	s.logger.Warnw("user forward traffic limit exceeded, disabling all rules",
		"user_id", userID,
		"traffic_used", usedTraffic,
		"traffic_limit", trafficLimit,
	)

	// Get all enabled rules for this user using pagination to handle large rule sets
	var allEnabledRules []*forward.ForwardRule
	page := 1
	pageSize := 100
	for {
		filter := forward.ListFilter{
			UserID:   &userID,
			Status:   "enabled",
			Page:     page,
			PageSize: pageSize,
		}
		rules, total, err := s.forwardRuleRepo.ListByUserID(ctx, userID, filter)
		if err != nil {
			s.logger.Errorw("failed to get enabled rules for user",
				"user_id", userID,
				"page", page,
				"error", err,
			)
			return fmt.Errorf("failed to get enabled rules: %w", err)
		}

		allEnabledRules = append(allEnabledRules, rules...)

		// Check if we've fetched all rules
		if int64(len(allEnabledRules)) >= total || len(rules) < pageSize {
			break
		}
		page++
	}

	if len(allEnabledRules) == 0 {
		s.logger.Debugw("no enabled rules to disable",
			"user_id", userID,
		)
		return nil
	}

	// Disable each rule
	disabledCount := 0
	for _, rule := range allEnabledRules {
		if err := rule.Disable(); err != nil {
			s.logger.Warnw("failed to disable rule",
				"user_id", userID,
				"rule_id", rule.ID(),
				"error", err,
			)
			continue
		}

		if err := s.forwardRuleRepo.Update(ctx, rule); err != nil {
			s.logger.Errorw("failed to update disabled rule",
				"user_id", userID,
				"rule_id", rule.ID(),
				"error", err,
			)
			continue
		}

		disabledCount++
		s.logger.Infow("disabled forward rule due to traffic limit",
			"user_id", userID,
			"rule_id", rule.ID(),
			"rule_short_id", rule.SID(),
		)
	}

	s.logger.Infow("traffic limit enforcement completed",
		"user_id", userID,
		"rules_disabled", disabledCount,
		"traffic_used", usedTraffic,
		"traffic_limit", trafficLimit,
	)

	return nil
}

// OnTrafficUpdate is called when traffic is updated for a forward rule.
// It checks if the user has exceeded their limit and enforces it if necessary.
func (s *TrafficLimitEnforcementService) OnTrafficUpdate(ctx context.Context, ruleID uint, uploadDelta, downloadDelta int64) error {
	s.logger.Debugw("traffic update received, checking limit enforcement",
		"rule_id", ruleID,
		"upload_delta", uploadDelta,
		"download_delta", downloadDelta,
	)

	// Get the rule to determine user_id
	rule, err := s.forwardRuleRepo.GetByID(ctx, ruleID)
	if err != nil {
		s.logger.Errorw("failed to get rule for traffic update",
			"rule_id", ruleID,
			"error", err,
		)
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if rule == nil {
		s.logger.Warnw("rule not found for traffic update",
			"rule_id", ruleID,
		)
		return forward.ErrRuleNotFound
	}

	// If rule has no user_id (admin-created), don't enforce limits
	if rule.UserID() == nil {
		s.logger.Debugw("rule has no user_id, skipping limit enforcement",
			"rule_id", ruleID,
		)
		return nil
	}

	userID := *rule.UserID()

	// Check and enforce limit for this user
	return s.CheckAndEnforceLimit(ctx, userID)
}

// getHighestTrafficLimitAndIDs returns the highest traffic limit across all Forward-type subscriptions
// and collects their subscription IDs for traffic query.
// Returns (limit, hasLimit, subscriptionIDs, error) where hasLimit is false if any subscription has unlimited traffic.
// Only considers subscriptions with PlanType = "forward".
func (s *TrafficLimitEnforcementService) getHighestTrafficLimitAndIDs(ctx context.Context, subscriptions []*subscription.Subscription) (uint64, bool, []uint, error) {
	var highestLimit uint64
	hasLimit := false
	var forwardSubscriptionIDs []uint

	// Collect plan IDs for batch query
	planIDs := make([]uint, 0, len(subscriptions))
	for _, sub := range subscriptions {
		planIDs = append(planIDs, sub.PlanID())
	}

	// Batch fetch all plans
	plansList, err := s.planRepo.GetByIDs(ctx, planIDs)
	if err != nil {
		s.logger.Errorw("failed to batch fetch plans", "error", err)
		return 0, false, nil, err
	}

	// Convert to map for quick lookup
	plans := make(map[uint]*subscription.Plan, len(plansList))
	for _, p := range plansList {
		plans[p.ID()] = p
	}

	// Process each subscription
	for _, sub := range subscriptions {
		plan, ok := plans[sub.PlanID()]
		if !ok || plan == nil {
			s.logger.Warnw("plan not found",
				"subscription_id", sub.ID(),
				"plan_id", sub.PlanID(),
			)
			continue
		}

		// Only check Forward-type plans for forward traffic limits
		if !plan.PlanType().IsForward() {
			s.logger.Debugw("skipping non-forward plan for traffic limit check",
				"subscription_id", sub.ID(),
				"plan_id", sub.PlanID(),
				"plan_type", plan.PlanType().String(),
			)
			continue
		}

		// Collect Forward-type subscription ID
		forwardSubscriptionIDs = append(forwardSubscriptionIDs, sub.ID())

		// Check if plan has unlimited traffic
		if plan.IsUnlimitedTraffic() {
			s.logger.Debugw("forward plan has unlimited traffic",
				"subscription_id", sub.ID(),
				"plan_id", sub.PlanID(),
			)
			return 0, false, forwardSubscriptionIDs, nil // Unlimited traffic - don't enforce
		}

		// Get the traffic limit from plan features
		limit, err := s.getForwardTrafficLimit(plan)
		if err != nil {
			s.logger.Warnw("failed to get traffic limit from plan",
				"subscription_id", sub.ID(),
				"plan_id", sub.PlanID(),
				"error", err,
			)
			continue
		}

		// If limit is 0, it means unlimited
		if limit == 0 {
			s.logger.Debugw("forward plan has unlimited traffic",
				"subscription_id", sub.ID(),
				"plan_id", sub.PlanID(),
			)
			return 0, false, forwardSubscriptionIDs, nil
		}

		// Track the highest limit
		if !hasLimit || limit > highestLimit {
			highestLimit = limit
			hasLimit = true
		}
	}

	return highestLimit, hasLimit, forwardSubscriptionIDs, nil
}

// getForwardTrafficLimit extracts the traffic limit from a plan.
// Returns 0 if unlimited or not set.
func (s *TrafficLimitEnforcementService) getForwardTrafficLimit(plan *subscription.Plan) (uint64, error) {
	features := plan.Features()
	if features == nil {
		return 0, nil // No features = unlimited
	}

	// Directly use GetTrafficLimit() - no fallback needed as limits are now unified
	return features.GetTrafficLimit()
}

// getTotalTrafficForSubscriptions calculates total traffic for given subscription IDs
// by combining data from two sources:
// - Last 24 hours: from Redis HourlyTrafficCache
// - Before 24 hours: from MySQL subscription_usage_stats table
func (s *TrafficLimitEnforcementService) getTotalTrafficForSubscriptions(ctx context.Context, subscriptionIDs []uint) (uint64, error) {
	if len(subscriptionIDs) == 0 {
		return 0, nil
	}

	now := biztime.NowUTC()

	// Use start of yesterday's business day as batch/speed boundary (Lambda architecture)
	// MySQL: complete days before yesterday; Redis: yesterday + today (within 48h TTL)
	recentBoundary := biztime.StartOfDayUTC(now.AddDate(0, 0, -1))

	var total uint64

	// Get recent traffic from Redis (yesterday + today, filter by forward_rule type)
	resourceType := subscription.ResourceTypeForwardRule.String()
	recentTraffic, err := s.hourlyTrafficCache.GetTotalTrafficBySubscriptionIDs(
		ctx, subscriptionIDs, resourceType, recentBoundary, now,
	)
	if err != nil {
		// Log warning but don't fail - Redis unavailability shouldn't block limit checks
		s.logger.Warnw("failed to get recent traffic from Redis, falling back to stats only",
			"subscription_ids", subscriptionIDs,
			"error", err,
		)
		// Continue with historical data only
	} else {
		// Sum traffic from Redis
		for _, traffic := range recentTraffic {
			total += traffic.Total
		}
		s.logger.Debugw("got recent 24h traffic from Redis",
			"subscription_ids_count", len(subscriptionIDs),
			"recent_total", total,
		)
	}

	// Get historical traffic from MySQL subscription_usage_stats (complete days before yesterday)
	// Use daily granularity for historical aggregation, filter by forward_rule type
	historicalTraffic, err := s.usageStatsRepo.GetTotalBySubscriptionIDs(
		ctx, subscriptionIDs, &resourceType, subscription.GranularityDaily, time.Time{}, recentBoundary.Add(-time.Second),
	)
	if err != nil {
		s.logger.Warnw("failed to get historical traffic from stats, using Redis data only",
			"subscription_ids", subscriptionIDs,
			"error", err,
		)
		// Continue with Redis data only if available
		if total == 0 {
			return 0, fmt.Errorf("failed to get traffic data from both sources: %w", err)
		}
		return total, nil
	}

	if historicalTraffic != nil {
		total += historicalTraffic.Total
		s.logger.Debugw("got historical traffic from MySQL stats",
			"subscription_ids_count", len(subscriptionIDs),
			"historical_total", historicalTraffic.Total,
			"combined_total", total,
		)
	}

	return total, nil
}
