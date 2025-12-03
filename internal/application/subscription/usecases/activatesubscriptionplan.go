package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ActivateSubscriptionPlanUseCase struct {
	planRepo subscription.SubscriptionPlanRepository
	logger   logger.Interface
}

func NewActivateSubscriptionPlanUseCase(
	planRepo subscription.SubscriptionPlanRepository,
	logger logger.Interface,
) *ActivateSubscriptionPlanUseCase {
	return &ActivateSubscriptionPlanUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

func (uc *ActivateSubscriptionPlanUseCase) Execute(ctx context.Context, planID uint) error {
	plan, err := uc.planRepo.GetByID(ctx, planID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan", "error", err, "plan_id", planID)
		return fmt.Errorf("failed to get subscription plan: %w", err)
	}
	if plan == nil {
		return fmt.Errorf("subscription plan not found: %d", planID)
	}

	if err := plan.Activate(); err != nil {
		uc.logger.Errorw("failed to activate subscription plan", "error", err, "plan_id", planID)
		return fmt.Errorf("failed to activate subscription plan: %w", err)
	}

	if err := uc.planRepo.Update(ctx, plan); err != nil {
		uc.logger.Errorw("failed to persist activation", "error", err, "plan_id", planID)
		return fmt.Errorf("failed to persist activation: %w", err)
	}

	uc.logger.Infow("subscription plan activated successfully", "plan_id", planID)
	return nil
}
