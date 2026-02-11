package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/payment"
	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// Grace period before auto-cancelling unpaid subscriptions
	// After a payment expires, the user has this much time to pay before the subscription is cancelled
	unpaidGracePeriod = 24 * time.Hour

	// Maximum time an inactive subscription can exist without any payment activity
	// If a subscription is created but no payment is ever made within this period, it will be cancelled
	inactiveSubscriptionTimeout = 48 * time.Hour
)

// CancelUnpaidSubscriptionsUseCase automatically cancels subscriptions
// that have not been paid within the allowed time period.
//
// Two scenarios are handled:
//  1. Payment expired: If a payment was created but expired, the subscription is cancelled
//     after a 24-hour grace period (unpaidGracePeriod).
//  2. No payment initiated: If no payment was ever created for a subscription, it is cancelled
//     after 48 hours from creation (inactiveSubscriptionTimeout).
//
// Fault tolerance:
//   - If Cancel() succeeds but Update() fails, the subscription state in memory is changed but not persisted.
//   - On the next scheduler run, the subscription will be re-processed but will fail at the Cancel() step
//     (since it's already marked as cancelled), which is safe and idempotent.
//   - This design prevents data corruption while allowing the scheduler to retry on transient failures.
type CancelUnpaidSubscriptionsUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	paymentRepo      payment.PaymentRepository
	logger           logger.Interface
}

// NewCancelUnpaidSubscriptionsUseCase creates a new CancelUnpaidSubscriptionsUseCase
func NewCancelUnpaidSubscriptionsUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	paymentRepo payment.PaymentRepository,
	logger logger.Interface,
) *CancelUnpaidSubscriptionsUseCase {
	return &CancelUnpaidSubscriptionsUseCase{
		subscriptionRepo: subscriptionRepo,
		paymentRepo:      paymentRepo,
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

	// Batch fetch subscription IDs with pending payments to avoid N+1 queries
	subscriptionIDs := make([]uint, 0, len(subscriptions))
	for _, sub := range subscriptions {
		subscriptionIDs = append(subscriptionIDs, sub.ID())
	}
	subsWithPendingPayments, err := uc.paymentRepo.GetSubscriptionIDsWithPendingPayments(ctx, subscriptionIDs)
	if err != nil {
		uc.logger.Errorw("failed to batch check pending payments", "error", err)
		return 0, fmt.Errorf("failed to check pending payments: %w", err)
	}
	pendingPaymentSet := make(map[uint]bool, len(subsWithPendingPayments))
	for _, subID := range subsWithPendingPayments {
		pendingPaymentSet[subID] = true
	}

	for _, sub := range subscriptions {
		// Check if there are any pending payments for this subscription first
		// A new payment may have been created, skip cancellation
		hasPending := pendingPaymentSet[sub.ID()]

		if hasPending {
			// New pending payment exists, skip cancellation
			// Also clear payment_expired_at if it was set
			if _, ok := sub.Metadata()["payment_expired_at"].(string); ok {
				uc.logger.Debugw("skipping cancellation: new pending payment exists",
					"subscription_id", sub.ID())
				sub.DeleteMetadata("payment_expired_at")
				if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
					uc.logger.Errorw("failed to clear payment_expired_at",
						"subscription_id", sub.ID(),
						"error", err)
				}
			}
			continue
		}

		// Determine if subscription should be cancelled and the reason
		var cancelReason string

		// Check if payment_expired_at is set in metadata
		paymentExpiredAtStr, hasPaymentExpired := sub.Metadata()["payment_expired_at"].(string)
		if hasPaymentExpired && paymentExpiredAtStr != "" {
			// Scenario 1: Payment was created but expired, check grace period
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

			cancelReason = "auto-cancelled: payment not completed within grace period"
		} else {
			// Scenario 2: No payment ever made, check creation timeout
			if now.Before(sub.CreatedAt().Add(inactiveSubscriptionTimeout)) {
				// Still within timeout period
				continue
			}

			cancelReason = "auto-cancelled: no payment initiated within timeout period"
		}

		// Cancel the subscription
		if err := sub.Cancel(cancelReason); err != nil {
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
		uc.logger.Debugw("subscription auto-cancelled",
			"subscription_id", sub.ID(),
			"reason", cancelReason,
			"created_at", sub.CreatedAt())
	}

	if cancelledCount > 0 {
		uc.logger.Infow("unpaid subscriptions cancelled",
			"total_checked", len(subscriptions),
			"cancelled", cancelledCount)
	}

	return cancelledCount, nil
}
