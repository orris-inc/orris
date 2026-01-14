package services

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SubscriptionDeactivationNotifier defines the interface for notifying subscription deactivation.
type SubscriptionDeactivationNotifier interface {
	NotifySubscriptionDeactivation(ctx context.Context, sub *subscription.Subscription) error
}

// NodeTrafficLimitEnforcementService enforces traffic limits for node subscriptions.
// When a subscription's traffic exceeds its plan limit, this service automatically
// suspends the subscription.
type NodeTrafficLimitEnforcementService struct {
	subscriptionRepo     subscription.SubscriptionRepository
	usageStatsRepo       subscription.SubscriptionUsageStatsRepository
	hourlyTrafficCache   cache.HourlyTrafficCache
	planRepo             subscription.PlanRepository
	quotaCache           cache.SubscriptionQuotaCache
	deactivationNotifier SubscriptionDeactivationNotifier
	logger               logger.Interface
}

// NewNodeTrafficLimitEnforcementService creates a new node traffic limit enforcement service.
func NewNodeTrafficLimitEnforcementService(
	subscriptionRepo subscription.SubscriptionRepository,
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyTrafficCache cache.HourlyTrafficCache,
	planRepo subscription.PlanRepository,
	quotaCache cache.SubscriptionQuotaCache,
	logger logger.Interface,
) *NodeTrafficLimitEnforcementService {
	return &NodeTrafficLimitEnforcementService{
		subscriptionRepo:   subscriptionRepo,
		usageStatsRepo:     usageStatsRepo,
		hourlyTrafficCache: hourlyTrafficCache,
		planRepo:           planRepo,
		quotaCache:         quotaCache,
		logger:             logger,
	}
}

// SetDeactivationNotifier sets the notifier for subscription deactivation events.
func (s *NodeTrafficLimitEnforcementService) SetDeactivationNotifier(notifier SubscriptionDeactivationNotifier) {
	s.deactivationNotifier = notifier
}

// CheckAndEnforceLimitForNode checks if a node subscription has exceeded its traffic limit
// and suspends it if necessary. Only applies to node-type subscriptions.
func (s *NodeTrafficLimitEnforcementService) CheckAndEnforceLimitForNode(ctx context.Context, subscriptionID uint) error {
	s.logger.Debugw("checking node traffic limit enforcement",
		"subscription_id", subscriptionID,
	)

	// Get subscription
	sub, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		s.logger.Errorw("failed to get subscription",
			"subscription_id", subscriptionID,
			"error", err,
		)
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if sub == nil {
		s.logger.Warnw("subscription not found",
			"subscription_id", subscriptionID,
		)
		return nil
	}

	// Skip if subscription is already suspended
	if sub.Status() == vo.StatusSuspended {
		s.logger.Debugw("subscription already suspended, skipping",
			"subscription_id", subscriptionID,
		)
		return nil
	}

	// Get plan
	plan, err := s.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		s.logger.Errorw("failed to get plan",
			"subscription_id", subscriptionID,
			"plan_id", sub.PlanID(),
			"error", err,
		)
		return fmt.Errorf("failed to get plan: %w", err)
	}

	if plan == nil {
		s.logger.Warnw("plan not found",
			"subscription_id", subscriptionID,
			"plan_id", sub.PlanID(),
		)
		return nil
	}

	// Only check node-type plans (skip forward and hybrid)
	if !plan.PlanType().IsNode() {
		s.logger.Debugw("skipping non-node plan",
			"subscription_id", subscriptionID,
			"plan_type", plan.PlanType().String(),
		)
		return nil
	}

	// Check if plan has unlimited traffic
	if plan.IsUnlimitedTraffic() {
		s.logger.Debugw("plan has unlimited traffic, skipping",
			"subscription_id", subscriptionID,
		)
		return nil
	}

	// Get traffic limit
	trafficLimit, err := plan.GetTrafficLimit()
	if err != nil {
		s.logger.Errorw("failed to get traffic limit",
			"subscription_id", subscriptionID,
			"error", err,
		)
		return fmt.Errorf("failed to get traffic limit: %w", err)
	}

	// If limit is 0, it means unlimited
	if trafficLimit == 0 {
		s.logger.Debugw("plan has unlimited traffic (limit=0), skipping",
			"subscription_id", subscriptionID,
		)
		return nil
	}

	// Get total traffic for this subscription
	usedTraffic, err := s.getTotalTrafficForSubscription(ctx, subscriptionID)
	if err != nil {
		s.logger.Errorw("failed to get total traffic",
			"subscription_id", subscriptionID,
			"error", err,
		)
		return fmt.Errorf("failed to get total traffic: %w", err)
	}

	// Check if traffic exceeds limit
	if usedTraffic <= trafficLimit {
		s.logger.Debugw("traffic within limit",
			"subscription_id", subscriptionID,
			"traffic_used", usedTraffic,
			"traffic_limit", trafficLimit,
		)
		return nil
	}

	// Traffic exceeded - suspend the subscription
	s.logger.Warnw("node subscription traffic limit exceeded, suspending",
		"subscription_id", subscriptionID,
		"traffic_used", usedTraffic,
		"traffic_limit", trafficLimit,
	)

	// Suspend the subscription
	reason := fmt.Sprintf("traffic limit exceeded: used %d bytes, limit %d bytes", usedTraffic, trafficLimit)
	if err := sub.Suspend(reason); err != nil {
		s.logger.Errorw("failed to suspend subscription",
			"subscription_id", subscriptionID,
			"error", err,
		)
		return fmt.Errorf("failed to suspend subscription: %w", err)
	}

	if err := s.subscriptionRepo.Update(ctx, sub); err != nil {
		s.logger.Errorw("failed to update suspended subscription",
			"subscription_id", subscriptionID,
			"error", err,
		)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	// Update quota cache to mark as suspended
	if s.quotaCache != nil {
		if err := s.quotaCache.SetSuspended(ctx, subscriptionID, true); err != nil {
			s.logger.Warnw("failed to update quota cache suspended status",
				"subscription_id", subscriptionID,
				"error", err,
			)
			// Don't fail the operation for cache update failure
		}
	}

	// Notify node agents about subscription deactivation
	if s.deactivationNotifier != nil {
		notifyCtx := context.Background()
		if err := s.deactivationNotifier.NotifySubscriptionDeactivation(notifyCtx, sub); err != nil {
			s.logger.Warnw("failed to notify nodes of subscription suspension",
				"subscription_id", subscriptionID,
				"error", err,
			)
			// Don't fail the operation for notification failure
		}
	}

	s.logger.Infow("node subscription suspended due to traffic limit",
		"subscription_id", subscriptionID,
		"traffic_used", usedTraffic,
		"traffic_limit", trafficLimit,
	)

	return nil
}

// getTotalTrafficForSubscription calculates total traffic for a subscription
// by combining data from two sources:
// - Last 24 hours: from Redis HourlyTrafficCache
// - Before 24 hours: from MySQL subscription_usage_stats table
func (s *NodeTrafficLimitEnforcementService) getTotalTrafficForSubscription(ctx context.Context, subscriptionID uint) (uint64, error) {
	now := biztime.NowUTC()
	dayAgo := now.Add(-24 * time.Hour)

	var total uint64

	// Get recent 24h traffic from Redis (filter by node type)
	resourceType := subscription.ResourceTypeNode.String()
	recentTraffic, err := s.hourlyTrafficCache.GetTotalTrafficBySubscriptionIDs(
		ctx, []uint{subscriptionID}, resourceType, dayAgo, now,
	)
	if err != nil {
		// Log warning but don't fail - Redis unavailability shouldn't block limit checks
		s.logger.Warnw("failed to get recent traffic from Redis, falling back to stats only",
			"subscription_id", subscriptionID,
			"error", err,
		)
		// Continue with historical data only
	} else {
		// Sum traffic from Redis
		for _, traffic := range recentTraffic {
			total += traffic.Total
		}
		s.logger.Debugw("got recent 24h traffic from Redis",
			"subscription_id", subscriptionID,
			"recent_total", total,
		)
	}

	// Get historical traffic from MySQL subscription_usage_stats (before 24 hours ago)
	// Use daily granularity for historical aggregation, filter by node type
	historicalTraffic, err := s.usageStatsRepo.GetTotalBySubscriptionIDs(
		ctx, []uint{subscriptionID}, &resourceType, subscription.GranularityDaily, time.Time{}, dayAgo,
	)
	if err != nil {
		s.logger.Warnw("failed to get historical traffic from stats, using Redis data only",
			"subscription_id", subscriptionID,
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
			"subscription_id", subscriptionID,
			"historical_total", historicalTraffic.Total,
			"combined_total", total,
		)
	}

	return total, nil
}
