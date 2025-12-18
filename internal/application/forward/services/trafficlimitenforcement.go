package services

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// TrafficLimitEnforcementService enforces traffic limits for user forward rules.
// When a user's total forward traffic exceeds their subscription plan limit,
// this service automatically disables all of their forward rules.
type TrafficLimitEnforcementService struct {
	forwardRuleRepo  forward.Repository
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	logger           logger.Interface
}

// NewTrafficLimitEnforcementService creates a new traffic limit enforcement service.
func NewTrafficLimitEnforcementService(
	forwardRuleRepo forward.Repository,
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *TrafficLimitEnforcementService {
	return &TrafficLimitEnforcementService{
		forwardRuleRepo:  forwardRuleRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}

// CheckAndEnforceLimit checks if a user has exceeded their traffic limit
// and disables all forward rules if necessary.
// Returns an error if the check fails, but not if rules are disabled successfully.
func (s *TrafficLimitEnforcementService) CheckAndEnforceLimit(ctx context.Context, userID uint) error {
	s.logger.Debugw("checking traffic limit enforcement",
		"user_id", userID,
	)

	// Get user's total forward traffic
	totalTraffic, err := s.forwardRuleRepo.GetTotalTrafficByUserID(ctx, userID)
	if err != nil {
		s.logger.Errorw("failed to get total traffic for user",
			"user_id", userID,
			"error", err,
		)
		return fmt.Errorf("failed to get total traffic: %w", err)
	}

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

	// Find the highest traffic limit across all subscriptions
	trafficLimit, hasLimit, err := s.getHighestTrafficLimit(ctx, activeSubscriptions)
	if err != nil {
		s.logger.Errorw("failed to determine traffic limit",
			"user_id", userID,
			"error", err,
		)
		return fmt.Errorf("failed to determine traffic limit: %w", err)
	}

	// If any subscription has unlimited traffic (limit = 0), don't enforce
	if !hasLimit {
		s.logger.Debugw("user has unlimited traffic subscription, skipping limit enforcement",
			"user_id", userID,
		)
		return nil
	}

	// Check if traffic exceeds limit
	// Convert to uint64 for safe comparison (totalTraffic should never be negative)
	var usedTraffic uint64
	if totalTraffic > 0 {
		usedTraffic = uint64(totalTraffic)
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
			"rule_short_id", rule.ShortID(),
		)
	}

	s.logger.Warnw("traffic limit enforcement completed",
		"user_id", userID,
		"rules_disabled", disabledCount,
		"traffic_used", totalTraffic,
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

// getHighestTrafficLimit returns the highest traffic limit across all subscriptions.
// Returns (limit, hasLimit, error) where hasLimit is false if any subscription has unlimited traffic.
func (s *TrafficLimitEnforcementService) getHighestTrafficLimit(ctx context.Context, subscriptions []*subscription.Subscription) (uint64, bool, error) {
	var highestLimit uint64
	hasLimit := false

	// Collect all unique plan IDs
	planIDs := make([]uint, 0, len(subscriptions))
	planIDSet := make(map[uint]bool)
	for _, sub := range subscriptions {
		if !planIDSet[sub.PlanID()] {
			planIDs = append(planIDs, sub.PlanID())
			planIDSet[sub.PlanID()] = true
		}
	}

	// Fetch all plans
	for _, planID := range planIDs {
		plan, err := s.planRepo.GetByID(ctx, planID)
		if err != nil {
			s.logger.Warnw("failed to get plan for subscription",
				"plan_id", planID,
				"error", err,
			)
			continue
		}

		if plan == nil {
			s.logger.Warnw("plan not found",
				"plan_id", planID,
			)
			continue
		}

		// Check if plan has unlimited traffic
		if plan.IsUnlimitedTraffic() {
			s.logger.Debugw("plan has unlimited traffic",
				"plan_id", planID,
			)
			return 0, false, nil // Unlimited traffic - don't enforce
		}

		// Get the forward traffic limit from plan features
		limit, err := s.getForwardTrafficLimit(plan)
		if err != nil {
			s.logger.Warnw("failed to get forward traffic limit from plan",
				"plan_id", planID,
				"error", err,
			)
			continue
		}

		// If limit is 0, it means unlimited
		if limit == 0 {
			s.logger.Debugw("plan has unlimited forward traffic",
				"plan_id", planID,
			)
			return 0, false, nil
		}

		// Track the highest limit
		if !hasLimit || limit > highestLimit {
			highestLimit = limit
			hasLimit = true
		}
	}

	return highestLimit, hasLimit, nil
}

// getForwardTrafficLimit extracts the forward traffic limit from a plan.
// It first checks for forward-specific traffic limit, then falls back to general traffic limit.
// Returns 0 if unlimited or not set.
func (s *TrafficLimitEnforcementService) getForwardTrafficLimit(plan *subscription.Plan) (uint64, error) {
	features := plan.Features()
	if features == nil {
		return 0, nil // No features = unlimited
	}

	// First check for forward-specific traffic limit
	forwardLimit, err := features.GetForwardTrafficLimit()
	if err != nil {
		s.logger.Debugw("failed to get forward traffic limit, trying general traffic limit",
			"plan_id", plan.ID(),
			"error", err,
		)
		// Fall back to general traffic limit
		generalLimit, err := features.GetTrafficLimit()
		if err != nil {
			return 0, fmt.Errorf("failed to get traffic limit: %w", err)
		}
		return generalLimit, nil
	}

	// If forward traffic limit is set and non-zero, use it
	if forwardLimit > 0 {
		return forwardLimit, nil
	}

	// Otherwise fall back to general traffic limit
	generalLimit, err := features.GetTrafficLimit()
	if err != nil {
		return 0, fmt.Errorf("failed to get general traffic limit: %w", err)
	}

	return generalLimit, nil
}
