package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/payment/paymentgateway"
	"github.com/orris-inc/orris/internal/domain/payment"
	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/domain/subscription"
	subscriptionVO "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
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

// USDTGatewayProvider provides access to the USDT gateway
type USDTGatewayProvider interface {
	IsEnabled() bool
	GetUSDTGateway() *paymentgateway.USDTGateway
}

type CreatePaymentUseCase struct {
	paymentRepo         payment.PaymentRepository
	subscriptionRepo    subscription.SubscriptionRepository
	planRepo            subscription.PlanRepository
	pricingRepo         subscription.PlanPricingRepository
	gateway             paymentgateway.PaymentGateway
	usdtGatewayProvider USDTGatewayProvider
	txMgr               *db.TransactionManager
	logger              logger.Interface
	config              PaymentConfig
}

type PaymentConfig struct {
	NotifyURL string
}

const (
	// maxPendingUSDTPerUser limits the number of pending USDT payments per user
	// to prevent suffix exhaustion attacks
	maxPendingUSDTPerUser = 10
)

func NewCreatePaymentUseCase(
	paymentRepo payment.PaymentRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	pricingRepo subscription.PlanPricingRepository,
	gateway paymentgateway.PaymentGateway,
	txMgr *db.TransactionManager,
	logger logger.Interface,
	config PaymentConfig,
) *CreatePaymentUseCase {
	return &CreatePaymentUseCase{
		paymentRepo:      paymentRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		pricingRepo:      pricingRepo,
		gateway:          gateway,
		txMgr:            txMgr,
		logger:           logger,
		config:           config,
	}
}

// SetUSDTGatewayProvider sets the USDT gateway provider
func (uc *CreatePaymentUseCase) SetUSDTGatewayProvider(provider USDTGatewayProvider) {
	uc.usdtGatewayProvider = provider
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
		return nil, errors.NewForbiddenError("permission denied: you don't own this subscription")
	}

	if sub.Status() != subscriptionVO.StatusInactive && sub.Status() != subscriptionVO.StatusPendingPayment {
		return nil, errors.NewValidationError("subscription status invalid for payment")
	}

	plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get plan", "error", err, "plan_id", sub.PlanID())
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	existingPayment, err := uc.paymentRepo.GetPendingBySubscriptionID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to check existing payment", "error", err, "subscription_id", cmd.SubscriptionID)
		return nil, fmt.Errorf("failed to check existing payment: %w", err)
	}
	if existingPayment != nil {
		return nil, errors.NewConflictError("pending payment already exists")
	}

	// BillingCycle is required to determine the price
	if cmd.BillingCycle == "" {
		return nil, errors.NewValidationError("billing cycle is required")
	}

	// Parse and validate the billing cycle
	billingCycle, err := subscriptionVO.ParseBillingCycle(cmd.BillingCycle)
	if err != nil {
		uc.logger.Warnw("invalid billing cycle", "error", err, "billing_cycle", cmd.BillingCycle)
		return nil, errors.NewValidationError("invalid billing cycle")
	}

	// Validate billing cycle consistency with subscription
	if subBC := sub.BillingCycle(); subBC != nil {
		if !subBC.Equals(billingCycle) {
			uc.logger.Warnw("billing cycle mismatch",
				"subscription_id", cmd.SubscriptionID,
				"subscription_billing_cycle", subBC.String(),
				"payment_billing_cycle", billingCycle.String(),
			)
			return nil, errors.NewValidationError("billing cycle does not match subscription billing cycle")
		}
	} else {
		// Legacy subscription without billing_cycle, allow any cycle but log warning
		uc.logger.Warnw("subscription has no billing cycle set, allowing payment with any cycle",
			"subscription_id", cmd.SubscriptionID,
			"payment_billing_cycle", billingCycle.String(),
		)
	}

	// Get pricing for the specified billing cycle
	pricing, err := uc.pricingRepo.GetByPlanAndCycle(ctx, sub.PlanID(), billingCycle)
	if err != nil {
		uc.logger.Warnw("failed to get pricing", "error", err, "plan_id", sub.PlanID(), "billing_cycle", billingCycle)
		return nil, fmt.Errorf("failed to get pricing: %w", err)
	}

	if pricing == nil {
		return nil, errors.NewNotFoundError("pricing not found for selected billing cycle")
	}

	// Safe conversion: cap uint64 price to math.MaxInt64 to prevent overflow
	amount := vo.NewMoney(utils.SafeUint64ToInt64(pricing.Price()), pricing.Currency())
	method, err := vo.NewPaymentMethod(cmd.PaymentMethod)
	if err != nil {
		return nil, errors.NewValidationError("invalid payment method")
	}

	paymentOrder, err := payment.NewPayment(cmd.SubscriptionID, cmd.UserID, amount, method)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Handle USDT payments separately
	if method.IsUSDT() {
		return uc.createUSDTPayment(ctx, paymentOrder, amount, method, plan.Name())
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

// createUSDTPayment handles USDT-specific payment creation
// Uses database transaction to ensure atomicity of payment creation and suffix allocation
func (uc *CreatePaymentUseCase) createUSDTPayment(ctx context.Context, paymentOrder *payment.Payment, amount vo.Money, method vo.PaymentMethod, planName string) (*CreatePaymentResult, error) {
	if uc.usdtGatewayProvider == nil || !uc.usdtGatewayProvider.IsEnabled() {
		return nil, errors.NewBadRequestError("USDT payment is not enabled")
	}

	usdtGateway := uc.usdtGatewayProvider.GetUSDTGateway()
	if usdtGateway == nil {
		return nil, errors.NewInternalError("USDT payment temporarily unavailable")
	}

	var result *CreatePaymentResult
	var usdtInfo *paymentgateway.USDTPaymentInfo

	// Execute all operations in a transaction to prevent race conditions
	txErr := uc.txMgr.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Check pending USDT payment count inside transaction to prevent race condition
		// This ensures concurrent requests cannot bypass the limit
		pendingCount, countErr := uc.paymentRepo.CountPendingUSDTPaymentsByUser(txCtx, paymentOrder.UserID())
		if countErr != nil {
			uc.logger.Errorw("failed to count pending USDT payments", "error", countErr, "user_id", paymentOrder.UserID())
			return fmt.Errorf("failed to check pending payments: %w", countErr)
		}
		if pendingCount >= maxPendingUSDTPerUser {
			uc.logger.Warnw("user exceeded pending USDT payment limit",
				"user_id", paymentOrder.UserID(),
				"pending_count", pendingCount,
				"limit", maxPendingUSDTPerUser,
			)
			return errors.NewValidationError("too many pending USDT payments, please complete or cancel existing payments first")
		}

		// Save the payment to get an ID
		if createErr := uc.paymentRepo.Create(txCtx, paymentOrder); createErr != nil {
			uc.logger.Errorw("failed to save payment", "error", createErr)
			return fmt.Errorf("failed to save payment: %w", createErr)
		}

		// Create USDT payment info (allocates suffix within transaction)
		var usdtErr error
		usdtInfo, usdtErr = usdtGateway.CreateUSDTPayment(txCtx, paymentOrder.ID(), amount.AmountInCents(), method)
		if usdtErr != nil {
			uc.logger.Errorw("failed to create USDT payment", "error", usdtErr)
			return fmt.Errorf("failed to create USDT payment: %w", usdtErr)
		}

		// Set USDT-specific info on the payment (using raw uint64 amount)
		paymentOrder.SetUSDTInfo(usdtInfo.ChainType, usdtInfo.USDTAmountRaw, usdtInfo.ReceivingAddress, usdtInfo.ExchangeRate)

		// Update the payment with USDT info
		if updateErr := uc.paymentRepo.Update(txCtx, paymentOrder); updateErr != nil {
			uc.logger.Errorw("failed to update payment with USDT info", "error", updateErr)
			return fmt.Errorf("failed to update payment with USDT info: %w", updateErr)
		}

		result = &CreatePaymentResult{
			Payment: paymentOrder,
		}
		return nil
	})

	if txErr != nil {
		// Transaction was rolled back, no need to release suffix manually
		return nil, txErr
	}

	uc.logger.Infow("USDT payment created successfully",
		"payment_id", paymentOrder.ID(),
		"order_no", paymentOrder.OrderNo(),
		"chain_type", usdtInfo.ChainType,
		"usdt_amount_raw", usdtInfo.USDTAmountRaw,
		"usdt_amount", usdtInfo.USDTAmountFloat(),
		"receiving_address", usdtInfo.ReceivingAddress,
	)

	return result, nil
}
