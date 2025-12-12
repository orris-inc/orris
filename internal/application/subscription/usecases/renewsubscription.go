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
	IsAutoRenew    bool
}

type RenewSubscriptionUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.SubscriptionPlanRepository
	logger           logger.Interface
}

func NewRenewSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.SubscriptionPlanRepository,
	logger logger.Interface,
) *RenewSubscriptionUseCase {
	return &RenewSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}

func (uc *RenewSubscriptionUseCase) Execute(ctx context.Context, cmd RenewSubscriptionCommand) error {
	sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan", "error", err, "plan_id", sub.PlanID())
		return fmt.Errorf("failed to get subscription plan: %w", err)
	}

	if !plan.IsActive() {
		return fmt.Errorf("subscription plan is not active")
	}

	newEndDate := uc.calculateNewEndDate(sub.EndDate(), plan.BillingCycle())

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
