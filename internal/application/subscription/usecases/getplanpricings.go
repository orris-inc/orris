package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type GetPlanPricingsQuery struct {
	PlanSID string
}

type GetPlanPricingsUseCase struct {
	planRepo    subscription.PlanRepository
	pricingRepo subscription.PlanPricingRepository
	logger      logger.Interface
}

func NewGetPlanPricingsUseCase(
	planRepo subscription.PlanRepository,
	pricingRepo subscription.PlanPricingRepository,
	logger logger.Interface,
) *GetPlanPricingsUseCase {
	return &GetPlanPricingsUseCase{
		planRepo:    planRepo,
		pricingRepo: pricingRepo,
		logger:      logger,
	}
}

func (uc *GetPlanPricingsUseCase) Execute(
	ctx context.Context,
	query GetPlanPricingsQuery,
) ([]*dto.PricingOptionDTO, error) {
	// Validate plan exists
	plan, err := uc.planRepo.GetBySID(ctx, query.PlanSID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan", "error", err, "plan_sid", query.PlanSID)
		return nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found: %s", query.PlanSID)
	}

	// Check if plan is active
	if !plan.IsActive() {
		uc.logger.Warnw("subscription plan is not active", "plan_id", plan.ID())
		return nil, fmt.Errorf("subscription plan is not active")
	}

	// Retrieve all active pricing options for the plan
	pricings, err := uc.pricingRepo.GetActivePricings(ctx, plan.ID())
	if err != nil {
		uc.logger.Errorw("failed to get plan pricings", "error", err, "plan_id", plan.ID())
		return nil, fmt.Errorf("failed to get plan pricings: %w", err)
	}

	// Convert to DTO list
	result := uc.toPricingOptionDTOList(pricings)

	uc.logger.Debugw("plan pricings retrieved successfully",
		"plan_id", plan.ID(),
		"pricing_count", len(result),
	)

	return result, nil
}

// toPricingOptionDTOList converts a list of PlanPricing domain objects to DTO objects
func (uc *GetPlanPricingsUseCase) toPricingOptionDTOList(pricings []*vo.PlanPricing) []*dto.PricingOptionDTO {
	if len(pricings) == 0 {
		return []*dto.PricingOptionDTO{}
	}

	result := make([]*dto.PricingOptionDTO, 0, len(pricings))
	for _, pricing := range pricings {
		if pricing != nil {
			result = append(result, &dto.PricingOptionDTO{
				BillingCycle: pricing.BillingCycle().String(),
				Price:        pricing.Price(),
				Currency:     pricing.Currency(),
				IsActive:     pricing.IsActive(),
			})
		}
	}

	return result
}
