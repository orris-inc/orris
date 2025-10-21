package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"
)

type DeactivateSubscriptionPlanUseCase struct {
	planRepo subscription.SubscriptionPlanRepository
	logger   logger.Interface
}

func NewDeactivateSubscriptionPlanUseCase(
	planRepo subscription.SubscriptionPlanRepository,
	logger logger.Interface,
) *DeactivateSubscriptionPlanUseCase {
	return &DeactivateSubscriptionPlanUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

func (uc *DeactivateSubscriptionPlanUseCase) Execute(ctx context.Context, planID uint) error {
	plan, err := uc.planRepo.GetByID(ctx, planID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan", "error", err, "plan_id", planID)
		return fmt.Errorf("failed to get subscription plan: %w", err)
	}
	if plan == nil {
		return fmt.Errorf("subscription plan not found: %d", planID)
	}

	if err := plan.Deactivate(); err != nil {
		uc.logger.Errorw("failed to deactivate subscription plan", "error", err, "plan_id", planID)
		return fmt.Errorf("failed to deactivate subscription plan: %w", err)
	}

	if err := uc.planRepo.Update(ctx, plan); err != nil {
		uc.logger.Errorw("failed to persist deactivation", "error", err, "plan_id", planID)
		return fmt.Errorf("failed to persist deactivation: %w", err)
	}

	uc.logger.Infow("subscription plan deactivated successfully", "plan_id", planID)
	return nil
}
