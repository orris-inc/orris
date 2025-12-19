package adapters

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ForwardTrafficRecorder defines the interface for recording forward rule traffic
type ForwardTrafficRecorder interface {
	// RecordForwardTraffic records forward rule traffic to subscription_usages table
	RecordForwardTraffic(ctx context.Context, ruleID uint, userID uint, upload, download int64) error
}

// ForwardTrafficRecorderAdapter records forward rule traffic to subscription_usages table
// It finds the user's active Forward-type subscriptions and records usage for each
type ForwardTrafficRecorderAdapter struct {
	subscriptionUsageRepo subscription.SubscriptionUsageRepository
	subscriptionRepo      subscription.SubscriptionRepository
	planRepo              subscription.PlanRepository
	logger                logger.Interface
}

// NewForwardTrafficRecorderAdapter creates a new forward traffic recorder adapter
func NewForwardTrafficRecorderAdapter(
	subscriptionUsageRepo subscription.SubscriptionUsageRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) ForwardTrafficRecorder {
	return &ForwardTrafficRecorderAdapter{
		subscriptionUsageRepo: subscriptionUsageRepo,
		subscriptionRepo:      subscriptionRepo,
		planRepo:              planRepo,
		logger:                logger,
	}
}

// RecordForwardTraffic records forward rule traffic to subscription_usages table
// It finds all active Forward-type subscriptions for the user and records usage for each
func (a *ForwardTrafficRecorderAdapter) RecordForwardTraffic(ctx context.Context, ruleID uint, userID uint, upload, download int64) error {
	// Skip zero traffic
	if upload == 0 && download == 0 {
		return nil
	}

	// Get user's active subscriptions
	activeSubscriptions, err := a.subscriptionRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		a.logger.Errorw("failed to get active subscriptions for forward traffic",
			"user_id", userID,
			"rule_id", ruleID,
			"error", err,
		)
		return err
	}

	if len(activeSubscriptions) == 0 {
		a.logger.Debugw("no active subscriptions found for user",
			"user_id", userID,
			"rule_id", ruleID,
		)
		return nil
	}

	// Filter Forward-type subscriptions and record usage
	period := time.Now().Truncate(time.Hour)
	recordedCount := 0

	for _, sub := range activeSubscriptions {
		// Get the plan to check its type
		plan, err := a.planRepo.GetByID(ctx, sub.PlanID())
		if err != nil {
			a.logger.Warnw("failed to get plan for subscription",
				"subscription_id", sub.ID(),
				"plan_id", sub.PlanID(),
				"error", err,
			)
			continue
		}

		if plan == nil {
			a.logger.Warnw("plan not found for subscription",
				"subscription_id", sub.ID(),
				"plan_id", sub.PlanID(),
			)
			continue
		}

		// Skip non-Forward type plans
		if !plan.PlanType().IsForward() {
			continue
		}

		// Create subscription usage record
		subID := sub.ID()
		usage, err := subscription.NewSubscriptionUsage(
			subscription.ResourceTypeForwardRule.String(),
			ruleID,
			&subID,
			period,
		)
		if err != nil {
			a.logger.Errorw("failed to create subscription usage entity",
				"error", err,
				"rule_id", ruleID,
				"subscription_id", sub.ID(),
			)
			continue
		}

		// Accumulate usage
		if err := usage.Accumulate(uint64(upload), uint64(download)); err != nil {
			a.logger.Errorw("failed to accumulate usage",
				"error", err,
				"rule_id", ruleID,
				"subscription_id", sub.ID(),
			)
			continue
		}

		// Record in repository
		if err := a.subscriptionUsageRepo.RecordUsage(ctx, usage); err != nil {
			a.logger.Errorw("failed to record forward traffic to subscription usage",
				"error", err,
				"rule_id", ruleID,
				"subscription_id", sub.ID(),
			)
			continue
		}

		recordedCount++
	}

	if recordedCount > 0 {
		a.logger.Debugw("forward traffic recorded to subscription_usages",
			"rule_id", ruleID,
			"user_id", userID,
			"upload", upload,
			"download", download,
			"subscriptions_count", recordedCount,
		)
	}

	return nil
}
