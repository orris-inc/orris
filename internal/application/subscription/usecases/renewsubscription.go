package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type RenewSubscriptionCommand struct {
	SubscriptionID uint
	BillingCycle   string // Optional: billing cycle for renewal period. If empty, uses current subscription's billing cycle.
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

	// Determine billing cycle: use provided value or fall back to subscription's current billing cycle
	var billingCycle vo.BillingCycle
	if cmd.BillingCycle != "" {
		// Use explicitly provided billing cycle
		parsed, err := vo.ParseBillingCycle(cmd.BillingCycle)
		if err != nil {
			uc.logger.Warnw("invalid billing cycle for renewal", "error", err, "billing_cycle", cmd.BillingCycle)
			return fmt.Errorf("invalid billing cycle: %w", err)
		}
		billingCycle = parsed
	} else if sub.BillingCycle() != nil {
		// Fall back to subscription's current billing cycle
		billingCycle = *sub.BillingCycle()
		uc.logger.Debugw("using subscription's current billing cycle for renewal",
			"subscription_id", cmd.SubscriptionID,
			"billing_cycle", billingCycle,
		)
	} else {
		// Legacy subscription without billing cycle - default to monthly
		billingCycle = vo.BillingCycleMonthly
		uc.logger.Infow("using default monthly billing cycle for legacy subscription",
			"subscription_id", cmd.SubscriptionID,
		)
	}

	// Lifetime subscriptions cannot be renewed
	if billingCycle.IsLifetime() {
		uc.logger.Warnw("attempted to renew lifetime subscription", "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("lifetime subscriptions cannot be renewed")
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

	newEndDate := billingCycle.NextBillingDate(sub.EndDate())

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
