package usecases

import (
	"context"
	"fmt"

	subscriptionUsecases "github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/domain/payment"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// RetrySubscriptionActivationUseCase retries subscription activation for paid non-USDT payments
// that failed to activate their subscriptions previously.
// USDT payments have their own retry mechanism in ConfirmUSDTPaymentUseCase.
type RetrySubscriptionActivationUseCase struct {
	paymentRepo            payment.PaymentRepository
	activateSubscriptionUC *subscriptionUsecases.ActivateSubscriptionUseCase
	logger                 logger.Interface
}

// NewRetrySubscriptionActivationUseCase creates a new RetrySubscriptionActivationUseCase
func NewRetrySubscriptionActivationUseCase(
	paymentRepo payment.PaymentRepository,
	activateSubscriptionUC *subscriptionUsecases.ActivateSubscriptionUseCase,
	logger logger.Interface,
) *RetrySubscriptionActivationUseCase {
	return &RetrySubscriptionActivationUseCase{
		paymentRepo:            paymentRepo,
		activateSubscriptionUC: activateSubscriptionUC,
		logger:                 logger,
	}
}

// Execute retries subscription activation for paid payments that previously failed activation
func (uc *RetrySubscriptionActivationUseCase) Execute(ctx context.Context) (int, error) {
	// Get paid non-USDT payments with pending subscription activation
	pendingPayments, err := uc.paymentRepo.GetPaidPaymentsNeedingActivation(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get payments needing activation: %w", err)
	}

	if len(pendingPayments) == 0 {
		return 0, nil
	}

	uc.logger.Infow("retrying subscription activations for non-USDT payments", "count", len(pendingPayments))

	successCount := 0
	for _, p := range pendingPayments {
		activateCmd := subscriptionUsecases.ActivateSubscriptionCommand{
			SubscriptionID: p.SubscriptionID(),
		}

		if err := uc.activateSubscriptionUC.Execute(ctx, activateCmd); err != nil {
			uc.logger.Warnw("retry activation failed",
				"payment_id", p.ID(),
				"subscription_id", p.SubscriptionID(),
				"error", err,
			)
			continue
		}

		// Clear the pending flag
		p.SetMetadata("subscription_activation_pending", false)
		if updateErr := uc.paymentRepo.Update(ctx, p); updateErr != nil {
			uc.logger.Warnw("failed to clear activation pending flag after retry",
				"payment_id", p.ID(),
				"error", updateErr,
			)
		}
		successCount++
	}

	if successCount > 0 {
		uc.logger.Infow("non-USDT subscription activations retried",
			"success", successCount,
			"total", len(pendingPayments),
		)
	}

	return successCount, nil
}
