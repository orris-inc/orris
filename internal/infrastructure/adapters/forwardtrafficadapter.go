package adapters

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ForwardTrafficRecorder defines the interface for recording forward rule traffic
type ForwardTrafficRecorder interface {
	// RecordForwardTraffic records forward rule traffic to Redis HourlyTrafficCache.
	// If subscriptionID is provided, records to that subscription's hourly bucket.
	// If subscriptionID is nil (admin rule), skip recording.
	RecordForwardTraffic(ctx context.Context, ruleID uint, subscriptionID *uint, upload, download int64) error
}

// ForwardTrafficRecorderAdapter records forward rule traffic to Redis HourlyTrafficCache.
// It records usage directly to the subscription bound to the rule.
type ForwardTrafficRecorderAdapter struct {
	hourlyTrafficCache cache.HourlyTrafficCache
	logger             logger.Interface
}

// NewForwardTrafficRecorderAdapter creates a new forward traffic recorder adapter
func NewForwardTrafficRecorderAdapter(
	hourlyTrafficCache cache.HourlyTrafficCache,
	logger logger.Interface,
) ForwardTrafficRecorder {
	return &ForwardTrafficRecorderAdapter{
		hourlyTrafficCache: hourlyTrafficCache,
		logger:             logger,
	}
}

// RecordForwardTraffic records forward rule traffic to Redis HourlyTrafficCache.
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

	// Use the constant for forward rule resource type
	resourceType := subscription.ResourceTypeForwardRule.String()

	// Write to Redis HourlyTrafficCache
	if err := a.hourlyTrafficCache.IncrementHourlyTraffic(ctx, *subscriptionID, resourceType, ruleID, upload, download); err != nil {
		a.logger.Errorw("failed to record forward traffic to hourly cache",
			"error", err,
			"rule_id", ruleID,
			"subscription_id", *subscriptionID,
		)
		return err
	}

	a.logger.Debugw("forward traffic recorded to hourly cache",
		"rule_id", ruleID,
		"subscription_id", *subscriptionID,
		"upload", upload,
		"download", download,
	)

	return nil
}
