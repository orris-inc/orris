package adapters

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ForwardTrafficRecorder defines the interface for recording forward rule traffic
type ForwardTrafficRecorder interface {
	// RecordForwardTraffic records forward rule traffic to subscription_usages table.
	// If subscriptionID is provided, records only to that subscription.
	// If subscriptionID is nil (admin rule), skip recording.
	RecordForwardTraffic(ctx context.Context, ruleID uint, subscriptionID *uint, upload, download int64) error
}

// ForwardTrafficRecorderAdapter records forward rule traffic to subscription_usages table.
// It records usage directly to the subscription bound to the rule.
type ForwardTrafficRecorderAdapter struct {
	subscriptionUsageRepo subscription.SubscriptionUsageRepository
	logger                logger.Interface
}

// NewForwardTrafficRecorderAdapter creates a new forward traffic recorder adapter
func NewForwardTrafficRecorderAdapter(
	subscriptionUsageRepo subscription.SubscriptionUsageRepository,
	logger logger.Interface,
) ForwardTrafficRecorder {
	return &ForwardTrafficRecorderAdapter{
		subscriptionUsageRepo: subscriptionUsageRepo,
		logger:                logger,
	}
}

// RecordForwardTraffic records forward rule traffic to subscription_usages table.
// It records usage directly to the subscription bound to the rule.
// If subscriptionID is nil (admin rule), no usage is recorded.
func (a *ForwardTrafficRecorderAdapter) RecordForwardTraffic(ctx context.Context, ruleID uint, subscriptionID *uint, upload, download int64) error {
	// Skip if no subscription (admin rule)
	if subscriptionID == nil {
		return nil
	}

	// Skip zero traffic
	if upload == 0 && download == 0 {
		return nil
	}

	// Truncate to hour in business timezone, then convert to UTC for storage
	period := biztime.TruncateToHourInBiz(biztime.NowUTC())

	// Create subscription usage record for the rule's subscription
	usage, err := subscription.NewSubscriptionUsage(
		subscription.ResourceTypeForwardRule.String(),
		ruleID,
		subscriptionID,
		period,
	)
	if err != nil {
		a.logger.Errorw("failed to create subscription usage entity",
			"error", err,
			"rule_id", ruleID,
			"subscription_id", *subscriptionID,
		)
		return err
	}

	// Accumulate usage
	if err := usage.Accumulate(uint64(upload), uint64(download)); err != nil {
		a.logger.Errorw("failed to accumulate usage",
			"error", err,
			"rule_id", ruleID,
			"subscription_id", *subscriptionID,
		)
		return err
	}

	// Record in repository
	if err := a.subscriptionUsageRepo.RecordUsage(ctx, usage); err != nil {
		a.logger.Errorw("failed to record forward traffic to subscription usage",
			"error", err,
			"rule_id", ruleID,
			"subscription_id", *subscriptionID,
		)
		return err
	}

	a.logger.Debugw("forward traffic recorded to subscription_usages",
		"rule_id", ruleID,
		"subscription_id", *subscriptionID,
		"upload", upload,
		"download", download,
	)

	return nil
}
