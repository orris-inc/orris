package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ExpireSubscriptionsUseCase handles marking expired subscriptions.
// This is a background job that runs periodically to ensure database consistency.
// Note: The display layer uses EffectiveStatus() for real-time accuracy,
// this job is mainly for data consistency in reports and statistics.
type ExpireSubscriptionsUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	logger           logger.Interface
}

// NewExpireSubscriptionsUseCase creates a new ExpireSubscriptionsUseCase
func NewExpireSubscriptionsUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	logger logger.Interface,
) *ExpireSubscriptionsUseCase {
	return &ExpireSubscriptionsUseCase{
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

// Execute finds and marks all expired subscriptions.
// Returns the number of subscriptions marked as expired.
func (uc *ExpireSubscriptionsUseCase) Execute(ctx context.Context) (int, error) {
	// Find subscriptions that are past end_date but still in active/trialing/past_due status
	expiredSubs, err := uc.subscriptionRepo.FindExpiredSubscriptions(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to find expired subscriptions: %w", err)
	}

	if len(expiredSubs) == 0 {
		return 0, nil
	}

	uc.logger.Infow("found expired subscriptions to process", "count", len(expiredSubs))

	markedCount := 0
	for _, sub := range expiredSubs {
		if err := sub.MarkAsExpired(); err != nil {
			uc.logger.Warnw("failed to mark subscription as expired",
				"subscription_id", sub.ID(),
				"subscription_sid", sub.SID(),
				"current_status", sub.Status().String(),
				"error", err,
			)
			continue
		}

		if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
			uc.logger.Errorw("failed to update expired subscription",
				"subscription_id", sub.ID(),
				"subscription_sid", sub.SID(),
				"error", err,
			)
			continue
		}

		markedCount++
		uc.logger.Debugw("subscription marked as expired",
			"subscription_id", sub.ID(),
			"subscription_sid", sub.SID(),
		)
	}

	return markedCount, nil
}
