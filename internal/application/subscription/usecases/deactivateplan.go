package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type DeactivatePlanUseCase struct {
	planRepo subscription.PlanRepository
	logger   logger.Interface
}

func NewDeactivatePlanUseCase(
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *DeactivatePlanUseCase {
	return &DeactivatePlanUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

func (uc *DeactivatePlanUseCase) Execute(ctx context.Context, planID uint) error {
	plan, err := uc.planRepo.GetByID(ctx, planID)
	if err != nil {
		uc.logger.Errorw("failed to get plan", "error", err, "plan_id", planID)
		return fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return fmt.Errorf("plan not found: %d", planID)
	}

	if err := plan.Deactivate(); err != nil {
		uc.logger.Errorw("failed to deactivate plan", "error", err, "plan_id", planID)
		return fmt.Errorf("failed to deactivate plan: %w", err)
	}

	if err := uc.planRepo.Update(ctx, plan); err != nil {
		uc.logger.Errorw("failed to persist deactivation", "error", err, "plan_id", planID)
		return fmt.Errorf("failed to persist deactivation: %w", err)
	}

	uc.logger.Infow("plan deactivated successfully", "plan_id", planID)
	return nil
}
