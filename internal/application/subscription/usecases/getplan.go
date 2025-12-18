package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type GetPlanUseCase struct {
	planRepo subscription.PlanRepository
	logger   logger.Interface
}

func NewGetPlanUseCase(
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *GetPlanUseCase {
	return &GetPlanUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

func (uc *GetPlanUseCase) ExecuteByID(ctx context.Context, planID uint) (*dto.PlanDTO, error) {
	plan, err := uc.planRepo.GetByID(ctx, planID)
	if err != nil {
		uc.logger.Errorw("failed to get plan by ID", "error", err, "plan_id", planID)
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found: %d", planID)
	}

	return dto.ToPlanDTO(plan), nil
}

func (uc *GetPlanUseCase) ExecuteBySlug(ctx context.Context, slug string) (*dto.PlanDTO, error) {
	plan, err := uc.planRepo.GetBySlug(ctx, slug)
	if err != nil {
		uc.logger.Errorw("failed to get plan by slug", "error", err, "slug", slug)
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found: %s", slug)
	}

	return dto.ToPlanDTO(plan), nil
}
