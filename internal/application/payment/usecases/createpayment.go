package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/payment/paymentgateway"
	"github.com/orris-inc/orris/internal/domain/payment"
	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/domain/subscription"
	subscriptionVO "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type CreatePaymentCommand struct {
	SubscriptionID uint
	UserID         uint
	BillingCycle   string // Required: billing cycle to determine price
	PaymentMethod  string
	ReturnURL      string
}

type CreatePaymentResult struct {
	Payment    *payment.Payment
	PaymentURL string
	QRCode     string
}

type CreatePaymentUseCase struct {
	paymentRepo      payment.PaymentRepository
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	pricingRepo      subscription.PlanPricingRepository
	gateway          paymentgateway.PaymentGateway
	logger           logger.Interface
	config           PaymentConfig
}

type PaymentConfig struct {
	NotifyURL string
}

func NewCreatePaymentUseCase(
	paymentRepo payment.PaymentRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	pricingRepo subscription.PlanPricingRepository,
	gateway paymentgateway.PaymentGateway,
	logger logger.Interface,
	config PaymentConfig,
) *CreatePaymentUseCase {
	return &CreatePaymentUseCase{
		paymentRepo:      paymentRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		pricingRepo:      pricingRepo,
		gateway:          gateway,
		logger:           logger,
		config:           config,
	}
}

func (uc *CreatePaymentUseCase) Execute(ctx context.Context, cmd CreatePaymentCommand) (*CreatePaymentResult, error) {
	sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Verify user owns this subscription
	if sub.UserID() != cmd.UserID {
		uc.logger.Warnw("unauthorized payment attempt", "subscription_id", cmd.SubscriptionID, "user_id", cmd.UserID, "owner_id", sub.UserID())
		return nil, fmt.Errorf("permission denied: you don't own this subscription")
	}

	if sub.Status() != subscriptionVO.StatusInactive && sub.Status() != subscriptionVO.StatusPendingPayment {
		return nil, fmt.Errorf("subscription status invalid for payment: %s", sub.Status())
	}

	plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get plan", "error", err, "plan_id", sub.PlanID())
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	existingPayment, _ := uc.paymentRepo.GetPendingBySubscriptionID(ctx, cmd.SubscriptionID)
	if existingPayment != nil {
		return nil, fmt.Errorf("pending payment already exists")
	}

	// BillingCycle is required to determine the price
	if cmd.BillingCycle == "" {
		return nil, fmt.Errorf("billing cycle is required")
	}

	// Parse and validate the billing cycle
	billingCycle, err := subscriptionVO.ParseBillingCycle(cmd.BillingCycle)
	if err != nil {
		uc.logger.Warnw("invalid billing cycle", "error", err, "billing_cycle", cmd.BillingCycle)
		return nil, fmt.Errorf("invalid billing cycle: %w", err)
	}

	// Get pricing for the specified billing cycle
	pricing, err := uc.pricingRepo.GetByPlanAndCycle(ctx, sub.PlanID(), billingCycle)
	if err != nil {
		uc.logger.Warnw("failed to get pricing", "error", err, "plan_id", sub.PlanID(), "billing_cycle", billingCycle)
		return nil, fmt.Errorf("failed to get pricing: %w", err)
	}

	if pricing == nil {
		return nil, fmt.Errorf("pricing not found for selected billing cycle")
	}

	amount := vo.NewMoney(int64(pricing.Price()), pricing.Currency())
	method, err := vo.NewPaymentMethod(cmd.PaymentMethod)
	if err != nil {
		return nil, fmt.Errorf("invalid payment method: %w", err)
	}

	paymentOrder, err := payment.NewPayment(cmd.SubscriptionID, cmd.UserID, amount, method)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	gatewayReq := paymentgateway.CreatePaymentRequest{
		OrderNo:   paymentOrder.OrderNo(),
		Amount:    amount.AmountInCents(),
		Currency:  amount.Currency(),
		Subject:   fmt.Sprintf("Subscription - %s", plan.Name()),
		Body:      fmt.Sprintf("Purchase %s subscription", plan.Name()),
		ReturnURL: cmd.ReturnURL,
		NotifyURL: uc.config.NotifyURL,
	}

	gatewayResp, err := uc.gateway.CreatePayment(ctx, gatewayReq)
	if err != nil {
		uc.logger.Errorw("failed to create payment in gateway", "error", err)
		return nil, fmt.Errorf("failed to create payment in gateway: %w", err)
	}

	paymentOrder.SetGatewayInfo(gatewayResp.GatewayOrderNo, gatewayResp.PaymentURL, gatewayResp.QRCode)

	if err := uc.paymentRepo.Create(ctx, paymentOrder); err != nil {
		uc.logger.Errorw("failed to save payment", "error", err)
		return nil, fmt.Errorf("failed to save payment: %w", err)
	}

	uc.logger.Infow("payment created successfully",
		"payment_id", paymentOrder.ID(),
		"order_no", paymentOrder.OrderNo(),
		"subscription_id", cmd.SubscriptionID,
		"amount", amount.AmountInCents())

	return &CreatePaymentResult{
		Payment:    paymentOrder,
		PaymentURL: gatewayResp.PaymentURL,
		QRCode:     gatewayResp.QRCode,
	}, nil
}
