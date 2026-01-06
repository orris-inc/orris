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
		uc.logger.Errorw("invalid payment callback signature", "error", err)
		return fmt.Errorf("invalid callback: %w", err)
	}

	paymentOrder, err := uc.paymentRepo.GetByGatewayOrderNo(ctx, callbackData.GatewayOrderNo)
	if err != nil {
		uc.logger.Errorw("payment order not found", "gateway_order_no", callbackData.GatewayOrderNo, "error", err)
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
	if err := paymentOrder.MarkAsPaid(callbackData.TransactionID); err != nil {
		return fmt.Errorf("failed to mark payment as paid: %w", err)
	}

	if err := uc.paymentRepo.Update(ctx, paymentOrder); err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	activateCmd := subscriptionUsecases.ActivateSubscriptionCommand{
		SubscriptionID: paymentOrder.SubscriptionID(),
	}

	if err := uc.activateSubscriptionUC.Execute(ctx, activateCmd); err != nil {
		uc.logger.Errorw("failed to activate subscription", "error", err, "subscription_id", paymentOrder.SubscriptionID())
		return fmt.Errorf("failed to activate subscription: %w", err)
	}

	uc.logger.Infow("payment processed successfully",
		"payment_id", paymentOrder.ID(),
		"subscription_id", paymentOrder.SubscriptionID(),
		"transaction_id", callbackData.TransactionID)

	// Notify admins about payment success (async, non-blocking)
	if uc.adminNotifier != nil {
		go func() {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			paidAt := time.Now()
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
		}()
	}

	return nil
}

func (uc *HandlePaymentCallbackUseCase) handlePaymentFailure(
	ctx context.Context,
	paymentOrder *payment.Payment,
	callbackData *paymentgateway.CallbackData,
) error {
	if err := paymentOrder.MarkAsFailed(callbackData.Status); err != nil {
		return fmt.Errorf("failed to mark payment as failed: %w", err)
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
