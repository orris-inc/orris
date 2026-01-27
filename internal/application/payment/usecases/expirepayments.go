package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/payment"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ExpirePaymentsUseCase struct {
	paymentRepo      payment.PaymentRepository
	subscriptionRepo subscription.SubscriptionRepository
	logger           logger.Interface
}

func NewExpirePaymentsUseCase(
	paymentRepo payment.PaymentRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	logger logger.Interface,
) *ExpirePaymentsUseCase {
	return &ExpirePaymentsUseCase{
		paymentRepo:      paymentRepo,
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

func (uc *ExpirePaymentsUseCase) Execute(ctx context.Context) (int, error) {
	expiredPayments, err := uc.paymentRepo.GetExpiredPayments(ctx)
	if err != nil {
		uc.logger.Errorw("failed to get expired payments", "error", err)
		return 0, fmt.Errorf("failed to get expired payments: %w", err)
	}

	if len(expiredPayments) == 0 {
		uc.logger.Debugw("no expired payments found")
		return 0, nil
	}

	uc.logger.Infow("processing expired payments", "count", len(expiredPayments))

	expiredCount := 0
	for _, p := range expiredPayments {
		if err := p.MarkAsExpired(); err != nil {
			uc.logger.Errorw("failed to mark payment as expired",
				"error", err,
				"payment_id", p.ID(),
				"order_no", p.OrderNo())
			continue
		}

		if err := uc.paymentRepo.Update(ctx, p); err != nil {
			uc.logger.Errorw("failed to update expired payment",
				"error", err,
				"payment_id", p.ID(),
				"order_no", p.OrderNo())
			continue
		}

		// Record payment expiration time on subscription for auto-cancel grace period
		if err := uc.markSubscriptionPaymentExpired(ctx, p.SubscriptionID()); err != nil {
			uc.logger.Warnw("failed to mark subscription payment expired",
				"error", err,
				"subscription_id", p.SubscriptionID())
			// Continue processing other payments even if this fails
		}

		expiredCount++
		uc.logger.Infow("payment marked as expired",
			"payment_id", p.ID(),
			"order_no", p.OrderNo(),
			"subscription_id", p.SubscriptionID())
	}

	uc.logger.Infow("expired payments processed",
		"total", len(expiredPayments),
		"expired", expiredCount)

	return expiredCount, nil
}

// markSubscriptionPaymentExpired records the payment expiration time on the subscription
// This is used by the auto-cancel logic to determine the grace period
func (uc *ExpirePaymentsUseCase) markSubscriptionPaymentExpired(ctx context.Context, subscriptionID uint) error {
	sub, err := uc.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return fmt.Errorf("subscription not found: %d", subscriptionID)
	}

	// Record the payment expiration time for grace period calculation
	sub.SetMetadata("payment_expired_at", biztime.FormatMetadataTime(biztime.NowUTC()))

	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}
