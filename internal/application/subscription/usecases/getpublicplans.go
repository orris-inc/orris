package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type GetPublicPlansUseCase struct {
	planRepo    subscription.PlanRepository
	pricingRepo subscription.PlanPricingRepository
	logger      logger.Interface
}

func NewGetPublicPlansUseCase(
	planRepo subscription.PlanRepository,
	pricingRepo subscription.PlanPricingRepository,
	logger logger.Interface,
) *GetPublicPlansUseCase {
	return &GetPublicPlansUseCase{
		planRepo:    planRepo,
		pricingRepo: pricingRepo,
		logger:      logger,
	}
}

func (uc *GetPublicPlansUseCase) Execute(ctx context.Context) ([]*dto.PlanDTO, error) {
	plans, err := uc.planRepo.GetActivePublicPlans(ctx)
	if err != nil {
		uc.logger.Errorw("failed to get active public plans", "error", err)
		return nil, fmt.Errorf("failed to get active public plans: %w", err)
	}

	// Extract plan IDs for batch pricing query
	planIDs := make([]uint, 0, len(plans))
	for _, plan := range plans {
		planIDs = append(planIDs, plan.ID())
	}

	// Batch fetch all pricings in a single query (solves N+1 problem)
	pricingsByPlanID, err := uc.pricingRepo.GetActivePricingsByPlanIDs(ctx, planIDs)
	if err != nil {
		uc.logger.Errorw("failed to get active pricings for plans", "plan_count", len(planIDs), "error", err)
		// Graceful degradation: continue without pricings
		pricingsByPlanID = make(map[uint][]*vo.PlanPricing)
	}

	// Build DTOs with pricings from the batch result
	planDTOs := make([]*dto.PlanDTO, 0, len(plans))
	for _, plan := range plans {
		pricings, hasPricings := pricingsByPlanID[plan.ID()]
		if hasPricings && len(pricings) > 0 {
			planDTOs = append(planDTOs, dto.ToPlanDTOWithPricings(plan, pricings))
		} else {
			// No pricings found for this plan
			planDTOs = append(planDTOs, dto.ToPlanDTO(plan))
		}
	}

	uc.logger.Debugw("public plans retrieved successfully",
		"plan_count", len(plans),
		"plans_with_pricings", len(pricingsByPlanID))

	return planDTOs, nil
}
