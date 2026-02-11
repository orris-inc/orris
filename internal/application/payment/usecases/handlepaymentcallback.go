package usecases

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/orris-inc/orris/internal/application/payment/paymentgateway"
	subscriptionUsecases "github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/domain/payment"
	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
	apperrors "github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AdminPaymentNotifier is the interface for notifying admins about payment success
type AdminPaymentNotifier interface {
	NotifyPaymentSuccess(ctx context.Context, cmd AdminPaymentCommand) error
}

// AdminPaymentCommand contains data for payment success notification
type AdminPaymentCommand struct {
	PaymentID      uint
	PaymentSID     string
	UserID         uint
	UserSID        string
	UserEmail      string
	SubscriptionID uint
	PlanName       string
	Amount         float64
	Currency       string
	PaymentMethod  string
	TransactionID  string
	PaidAt         time.Time
}

// PaymentUserInfoProvider provides user info for payment notifications
type PaymentUserInfoProvider interface {
	GetUserSIDAndEmail(ctx context.Context, userID uint) (sid, email string, err error)
}

// PaymentPlanInfoProvider provides plan info for payment notifications
type PaymentPlanInfoProvider interface {
	GetPlanName(ctx context.Context, subscriptionID uint) (string, error)
}

type HandlePaymentCallbackUseCase struct {
	paymentRepo            payment.PaymentRepository
	activateSubscriptionUC *subscriptionUsecases.ActivateSubscriptionUseCase
	gateway                paymentgateway.PaymentGateway
	adminNotifier          AdminPaymentNotifier    // Optional
	userInfoProvider       PaymentUserInfoProvider // Optional
	planInfoProvider       PaymentPlanInfoProvider // Optional
	logger                 logger.Interface
}

func NewHandlePaymentCallbackUseCase(
	paymentRepo payment.PaymentRepository,
	activateSubscriptionUC *subscriptionUsecases.ActivateSubscriptionUseCase,
	gateway paymentgateway.PaymentGateway,
	logger logger.Interface,
) *HandlePaymentCallbackUseCase {
	return &HandlePaymentCallbackUseCase{
		paymentRepo:            paymentRepo,
		activateSubscriptionUC: activateSubscriptionUC,
		gateway:                gateway,
		logger:                 logger,
	}
}

// SetAdminNotifier sets the admin notifier (optional dependency injection)
func (uc *HandlePaymentCallbackUseCase) SetAdminNotifier(notifier AdminPaymentNotifier) {
	uc.adminNotifier = notifier
}

// SetUserInfoProvider sets the user info provider (optional dependency injection)
func (uc *HandlePaymentCallbackUseCase) SetUserInfoProvider(provider PaymentUserInfoProvider) {
	uc.userInfoProvider = provider
}

// SetPlanInfoProvider sets the plan info provider (optional dependency injection)
func (uc *HandlePaymentCallbackUseCase) SetPlanInfoProvider(provider PaymentPlanInfoProvider) {
	uc.planInfoProvider = provider
}

func (uc *HandlePaymentCallbackUseCase) Execute(ctx context.Context, req *http.Request) error {
	callbackData, err := uc.gateway.VerifyCallback(req)
	if err != nil {
		uc.logger.Warnw("invalid payment callback signature", "error", err)
		return apperrors.NewValidationError("invalid payment callback", err.Error())
	}

	paymentOrder, err := uc.paymentRepo.GetByGatewayOrderNo(ctx, callbackData.GatewayOrderNo)
	if err != nil {
		uc.logger.Warnw("payment order not found", "gateway_order_no", callbackData.GatewayOrderNo, "error", err)
		return fmt.Errorf("payment not found: %w", err)
	}

	if paymentOrder.Status() == vo.PaymentStatusPaid {
		uc.logger.Infow("payment already processed", "payment_id", paymentOrder.ID())
		return nil
	}

	if callbackData.Status == "TRADE_SUCCESS" || callbackData.Status == "success" {
		return uc.handlePaymentSuccess(ctx, paymentOrder, callbackData)
	} else {
		return uc.handlePaymentFailure(ctx, paymentOrder, callbackData)
	}
}

func (uc *HandlePaymentCallbackUseCase) handlePaymentSuccess(
	ctx context.Context,
	paymentOrder *payment.Payment,
	callbackData *paymentgateway.CallbackData,
) error {
	// Validate callback amount and currency match the payment
	if err := paymentOrder.ValidateCallbackAmount(callbackData.Amount, callbackData.Currency); err != nil {
		uc.logger.Errorw("callback amount/currency mismatch",
			"payment_id", paymentOrder.ID(),
			"error", err,
			"expected_amount", paymentOrder.Amount().AmountInCents(),
			"callback_amount", callbackData.Amount,
			"expected_currency", paymentOrder.Amount().Currency(),
			"callback_currency", callbackData.Currency,
		)
		// Mark payment as failed due to amount mismatch
		if markErr := paymentOrder.MarkAsFailed(fmt.Sprintf("amount/currency mismatch: %s", err.Error())); markErr != nil {
			uc.logger.Errorw("failed to mark payment as failed after amount mismatch", "error", markErr)
			return markErr
		}
		if updateErr := uc.paymentRepo.Update(ctx, paymentOrder); updateErr != nil {
			uc.logger.Errorw("failed to update payment after amount mismatch", "error", updateErr)
			return fmt.Errorf("failed to update payment after amount mismatch: %w", updateErr)
		}
		// Acknowledge callback to avoid repeated retries for a known mismatch
		return nil
	}

	// Pre-set activation_pending flag before marking as paid.
	// This ensures the flag is persisted together with paid status in a single update,
	// so if activation fails later, we have a reliable marker for retry.
	paymentOrder.SetMetadata("subscription_activation_pending", true)

	if err := paymentOrder.MarkAsPaid(callbackData.TransactionID); err != nil {
		return err
	}

	if err := uc.paymentRepo.Update(ctx, paymentOrder); err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	// Now try to activate the subscription
	activateCmd := subscriptionUsecases.ActivateSubscriptionCommand{
		SubscriptionID: paymentOrder.SubscriptionID(),
	}

	if err := uc.activateSubscriptionUC.Execute(ctx, activateCmd); err != nil {
		uc.logger.Errorw("failed to activate subscription after payment, will retry later",
			"error", err,
			"payment_id", paymentOrder.ID(),
			"subscription_id", paymentOrder.SubscriptionID(),
		)
		// Update metadata with error details for debugging
		paymentOrder.SetMetadata("subscription_activation_error", err.Error())
		if updateErr := uc.paymentRepo.Update(ctx, paymentOrder); updateErr != nil {
			uc.logger.Warnw("failed to update payment with activation error details",
				"payment_id", paymentOrder.ID(),
				"error", updateErr,
			)
		}
		// Return nil to acknowledge the callback (payment is already recorded with pending flag)
		// The scheduler will retry subscription activation later
	} else {
		// Activation succeeded, clear the pending flag
		paymentOrder.SetMetadata("subscription_activation_pending", false)
		paymentOrder.SetMetadata("subscription_activation_error", nil)
		if updateErr := uc.paymentRepo.Update(ctx, paymentOrder); updateErr != nil {
			// Return error to trigger callback retry, ensuring the pending flag gets cleared
			// This prevents the scheduler from continuously retrying activation unnecessarily
			uc.logger.Errorw("failed to clear activation pending flag, triggering callback retry",
				"payment_id", paymentOrder.ID(),
				"error", updateErr,
			)
			return fmt.Errorf("failed to clear activation pending flag: %w", updateErr)
		}
	}

	uc.logger.Infow("payment processed successfully",
		"payment_id", paymentOrder.ID(),
		"subscription_id", paymentOrder.SubscriptionID(),
		"transaction_id", callbackData.TransactionID)

	// Notify admins about payment success (async, non-blocking)
	if uc.adminNotifier != nil {
		goroutine.SafeGo(uc.logger, "payment-callback-notify-admins", func() {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			paidAt := biztime.NowUTC()
			if paymentOrder.PaidAt() != nil {
				paidAt = *paymentOrder.PaidAt()
			}
			cmd := AdminPaymentCommand{
				PaymentID:      paymentOrder.ID(),
				PaymentSID:     paymentOrder.OrderNo(), // Use OrderNo as SID
				UserID:         paymentOrder.UserID(),
				SubscriptionID: paymentOrder.SubscriptionID(),
				Amount:         paymentOrder.Amount().AmountInYuan(),
				Currency:       paymentOrder.Amount().Currency(),
				PaymentMethod:  paymentOrder.PaymentMethod().String(),
				TransactionID:  callbackData.TransactionID,
				PaidAt:         paidAt,
			}

			// Try to get user info
			if uc.userInfoProvider != nil {
				if sid, email, err := uc.userInfoProvider.GetUserSIDAndEmail(notifyCtx, paymentOrder.UserID()); err == nil {
					cmd.UserSID = sid
					cmd.UserEmail = email
				}
			}

			// Try to get plan name
			if uc.planInfoProvider != nil {
				if planName, err := uc.planInfoProvider.GetPlanName(notifyCtx, paymentOrder.SubscriptionID()); err == nil {
					cmd.PlanName = planName
				}
			}

			if err := uc.adminNotifier.NotifyPaymentSuccess(notifyCtx, cmd); err != nil {
				uc.logger.Warnw("failed to notify admins about payment success", "payment_id", paymentOrder.ID(), "error", err)
			}
		})
	}

	return nil
}

func (uc *HandlePaymentCallbackUseCase) handlePaymentFailure(
	ctx context.Context,
	paymentOrder *payment.Payment,
	callbackData *paymentgateway.CallbackData,
) error {
	if err := paymentOrder.MarkAsFailed(callbackData.Status); err != nil {
		return err
	}

	if err := uc.paymentRepo.Update(ctx, paymentOrder); err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	uc.logger.Infow("payment failed",
		"payment_id", paymentOrder.ID(),
		"subscription_id", paymentOrder.SubscriptionID(),
		"reason", callbackData.Status)

	return nil
}
