package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type RenewSubscriptionCommand struct {
	SubscriptionID uint
	BillingCycle   string // Required: billing cycle for renewal period
	IsAutoRenew    bool
}

type RenewSubscriptionUseCase struct {
	subscriptionRepo     subscription.SubscriptionRepository
	planRepo             subscription.PlanRepository
	pricingRepo          subscription.PlanPricingRepository
	subscriptionNotifier SubscriptionChangeNotifier // Optional: for notifying node agents
	logger               logger.Interface
}

func NewRenewSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	pricingRepo subscription.PlanPricingRepository,
	logger logger.Interface,
) *RenewSubscriptionUseCase {
	return &RenewSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		pricingRepo:      pricingRepo,
		logger:           logger,
	}
}

// SetSubscriptionNotifier sets the subscription change notifier (optional).
func (uc *RenewSubscriptionUseCase) SetSubscriptionNotifier(notifier SubscriptionChangeNotifier) {
	uc.subscriptionNotifier = notifier
}

func (uc *RenewSubscriptionUseCase) Execute(ctx context.Context, cmd RenewSubscriptionCommand) error {
	sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get plan", "error", err, "plan_id", sub.PlanID())
		return fmt.Errorf("failed to get plan: %w", err)
	}

	if !plan.IsActive() {
		return fmt.Errorf("plan is not active")
	}

	// BillingCycle is required for renewal
	if cmd.BillingCycle == "" {
		return fmt.Errorf("billing cycle is required for renewal")
	}

	// Parse and validate the billing cycle
	billingCycle, err := vo.ParseBillingCycle(cmd.BillingCycle)
	if err != nil {
		uc.logger.Warnw("invalid billing cycle for renewal", "error", err, "billing_cycle", cmd.BillingCycle)
		return fmt.Errorf("invalid billing cycle: %w", err)
	}

	// Verify that pricing exists for this plan and billing cycle
	pricing, err := uc.pricingRepo.GetByPlanAndCycle(ctx, sub.PlanID(), billingCycle)
	if err != nil {
		uc.logger.Warnw("failed to get pricing for billing cycle", "error", err, "plan_id", sub.PlanID(), "billing_cycle", billingCycle)
		return fmt.Errorf("pricing not available for selected billing cycle: %w", err)
	}

	if pricing == nil {
		uc.logger.Warnw("pricing not found for billing cycle", "plan_id", sub.PlanID(), "billing_cycle", billingCycle)
		return fmt.Errorf("pricing not found for selected billing cycle")
	}

	newEndDate := uc.calculateNewEndDate(sub.EndDate(), billingCycle)

	if err := sub.Renew(newEndDate); err != nil {
		uc.logger.Errorw("failed to renew subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to renew subscription: %w", err)
	}

	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		uc.logger.Errorw("failed to update subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	uc.logger.Infow("subscription renewed successfully",
		"subscription_id", cmd.SubscriptionID,
		"new_end_date", newEndDate,
		"status", sub.Status(),
	)

	// Notify node agents about the renewed subscription (updated expiry)
	if uc.subscriptionNotifier != nil {
		notifyCtx := context.Background()
		if err := uc.subscriptionNotifier.NotifySubscriptionUpdate(notifyCtx, sub); err != nil {
			// Log error but don't fail the renewal
			uc.logger.Warnw("failed to notify nodes of subscription renewal",
				"subscription_id", cmd.SubscriptionID,
				"error", err,
			)
		}
	}

	return nil
}

func (uc *RenewSubscriptionUseCase) calculateNewEndDate(currentEndDate time.Time, billingCycle vo.BillingCycle) time.Time {
	switch billingCycle {
	case vo.BillingCycleMonthly:
		return currentEndDate.AddDate(0, 1, 0)
	case vo.BillingCycleQuarterly:
		return currentEndDate.AddDate(0, 3, 0)
	case vo.BillingCycleYearly:
		return currentEndDate.AddDate(1, 0, 0)
	default:
		return currentEndDate.AddDate(0, 1, 0)
	}
}
