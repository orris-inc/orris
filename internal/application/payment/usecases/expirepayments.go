package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/payment"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ExpirePaymentsUseCase struct {
	paymentRepo payment.PaymentRepository
	logger      logger.Interface
}

func NewExpirePaymentsUseCase(
	paymentRepo payment.PaymentRepository,
	logger logger.Interface,
) *ExpirePaymentsUseCase {
	return &ExpirePaymentsUseCase{
		paymentRepo: paymentRepo,
		logger:      logger,
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
