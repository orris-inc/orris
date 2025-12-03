package usecases

import (
	"context"
	"fmt"
	"net/http"

	"github.com/orris-inc/orris/internal/application/payment/payment_gateway"
	subscriptionUsecases "github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/domain/payment"
	vo "github.com/orris-inc/orris/internal/domain/payment/value_objects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type HandlePaymentCallbackUseCase struct {
	paymentRepo            payment.PaymentRepository
	activateSubscriptionUC *subscriptionUsecases.ActivateSubscriptionUseCase
	gateway                payment_gateway.PaymentGateway
	logger                 logger.Interface
}

func NewHandlePaymentCallbackUseCase(
	paymentRepo payment.PaymentRepository,
	activateSubscriptionUC *subscriptionUsecases.ActivateSubscriptionUseCase,
	gateway payment_gateway.PaymentGateway,
	logger logger.Interface,
) *HandlePaymentCallbackUseCase {
	return &HandlePaymentCallbackUseCase{
		paymentRepo:            paymentRepo,
		activateSubscriptionUC: activateSubscriptionUC,
		gateway:                gateway,
		logger:                 logger,
	}
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
	callbackData *payment_gateway.CallbackData,
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

	return nil
}

func (uc *HandlePaymentCallbackUseCase) handlePaymentFailure(
	ctx context.Context,
	paymentOrder *payment.Payment,
	callbackData *payment_gateway.CallbackData,
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
