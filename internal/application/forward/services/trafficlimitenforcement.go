package services

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// TrafficLimitEnforcementService enforces traffic limits for user forward rules.
// When a user's total forward traffic exceeds their subscription plan limit,
// this service automatically disables all of their forward rules.
type TrafficLimitEnforcementService struct {
	forwardRuleRepo       forward.Repository
	subscriptionRepo      subscription.SubscriptionRepository
	subscriptionUsageRepo subscription.SubscriptionUsageRepository
	planRepo              subscription.PlanRepository
	logger                logger.Interface
}

// NewTrafficLimitEnforcementService creates a new traffic limit enforcement service.
func NewTrafficLimitEnforcementService(
	forwardRuleRepo forward.Repository,
	subscriptionRepo subscription.SubscriptionRepository,
	subscriptionUsageRepo subscription.SubscriptionUsageRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *TrafficLimitEnforcementService {
	return &TrafficLimitEnforcementService{
		forwardRuleRepo:       forwardRuleRepo,
		subscriptionRepo:      subscriptionRepo,
		subscriptionUsageRepo: subscriptionUsageRepo,
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

	// Get user's total forward traffic from subscription_usages table
	usageSummary, err := s.subscriptionUsageRepo.GetTotalUsageBySubscriptionIDs(
		ctx,
		subscription.ResourceTypeForwardRule.String(),
		forwardSubscriptionIDs,
		time.Time{}, // No time range limit - get all historical usage
		time.Time{},
	)
	if err != nil {
		s.logger.Errorw("failed to get total traffic for user from subscription_usages",
			"user_id", userID,
			"subscription_ids", forwardSubscriptionIDs,
			"error", err,
		)
		return fmt.Errorf("failed to get total traffic: %w", err)
	}

	usedTraffic := usageSummary.Total
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

	s.logger.Warnw("traffic limit enforcement completed",
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

	// Process each subscription
	for _, sub := range subscriptions {
		plan, err := s.planRepo.GetByID(ctx, sub.PlanID())
		if err != nil {
			s.logger.Warnw("failed to get plan for subscription",
				"subscription_id", sub.ID(),
				"plan_id", sub.PlanID(),
				"error", err,
			)
			continue
		}

		if plan == nil {
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
