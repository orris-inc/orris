package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// Grace period before auto-cancelling unpaid subscriptions
	// After a payment expires, the user has this much time to pay before the subscription is cancelled
	unpaidGracePeriod = 24 * time.Hour
)

// CancelUnpaidSubscriptionsUseCase automatically cancels subscriptions
// that have unpaid payments beyond the grace period.
//
// Fault tolerance:
// - If Cancel() succeeds but Update() fails, the subscription state in memory is changed but not persisted.
// - On the next scheduler run, the subscription will be re-processed but will fail at the Cancel() step
//   (since it's already marked as cancelled), which is safe and idempotent.
// - This design prevents data corruption while allowing the scheduler to retry on transient failures.
type CancelUnpaidSubscriptionsUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	logger           logger.Interface
}

// NewCancelUnpaidSubscriptionsUseCase creates a new CancelUnpaidSubscriptionsUseCase
func NewCancelUnpaidSubscriptionsUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	logger logger.Interface,
) *CancelUnpaidSubscriptionsUseCase {
	return &CancelUnpaidSubscriptionsUseCase{
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

// Execute checks and cancels subscriptions with expired payments beyond grace period
func (uc *CancelUnpaidSubscriptionsUseCase) Execute(ctx context.Context) (int, error) {
	// Get subscriptions that are inactive or pending_payment
	// These are candidates for auto-cancellation
	subscriptions, err := uc.subscriptionRepo.GetByStatuses(ctx, []vo.SubscriptionStatus{
		vo.StatusInactive,
		vo.StatusPendingPayment,
	})
	if err != nil {
		uc.logger.Errorw("failed to get unpaid subscriptions", "error", err)
		return 0, fmt.Errorf("failed to get unpaid subscriptions: %w", err)
	}

	if len(subscriptions) == 0 {
		return 0, nil
	}

	now := biztime.NowUTC()
	cancelledCount := 0

	for _, sub := range subscriptions {
		// Check if payment_expired_at is set in metadata
		paymentExpiredAtStr, ok := sub.Metadata()["payment_expired_at"].(string)
		if !ok || paymentExpiredAtStr == "" {
			// No payment expiration recorded, skip
			continue
		}

		paymentExpiredAt, err := biztime.ParseMetadataTime(paymentExpiredAtStr)
		if err != nil {
			uc.logger.Warnw("failed to parse payment_expired_at",
				"subscription_id", sub.ID(),
				"value", paymentExpiredAtStr,
				"error", err)
			continue
		}

		// Check if grace period has passed
		if now.Before(paymentExpiredAt.Add(unpaidGracePeriod)) {
			// Still within grace period
			continue
		}

		// Grace period has passed, cancel the subscription
		if err := sub.Cancel("auto-cancelled: payment not completed within grace period"); err != nil {
			uc.logger.Errorw("failed to cancel subscription",
				"subscription_id", sub.ID(),
				"error", err)
			continue
		}

		if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
			uc.logger.Errorw("failed to update cancelled subscription",
				"subscription_id", sub.ID(),
				"error", err)
			continue
		}

		cancelledCount++
		uc.logger.Infow("subscription auto-cancelled due to unpaid payment",
			"subscription_id", sub.ID(),
			"payment_expired_at", paymentExpiredAt,
			"grace_period", unpaidGracePeriod)
	}

	if cancelledCount > 0 {
		uc.logger.Infow("unpaid subscriptions cancelled",
			"total_checked", len(subscriptions),
			"cancelled", cancelledCount)
	}

	return cancelledCount, nil
}
