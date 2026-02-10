package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ActivatePlanUseCase struct {
	planRepo subscription.PlanRepository
	logger   logger.Interface
}

func NewActivatePlanUseCase(
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *ActivatePlanUseCase {
	return &ActivatePlanUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

func (uc *ActivatePlanUseCase) Execute(ctx context.Context, planSID string) error {
	plan, err := uc.planRepo.GetBySID(ctx, planSID)
	if err != nil {
		uc.logger.Errorw("failed to get plan", "error", err, "plan_sid", planSID)
		return fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return fmt.Errorf("plan not found: %s", planSID)
	}

	if err := plan.Activate(); err != nil {
		uc.logger.Errorw("failed to activate plan", "error", err, "plan_id", plan.ID())
		return err
	}

	if err := uc.planRepo.Update(ctx, plan); err != nil {
		uc.logger.Errorw("failed to persist activation", "error", err, "plan_id", plan.ID())
		return fmt.Errorf("failed to persist activation: %w", err)
	}

	uc.logger.Infow("plan activated successfully", "plan_id", plan.ID())
	return nil
}
