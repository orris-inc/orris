package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ListPlansQuery struct {
	Status   *string
	IsPublic *bool
	PlanType *string // Optional: filter by plan type ("node" or "forward")
	Page     int
	PageSize int
}

type ListPlansResult struct {
	Plans []*dto.PlanDTO `json:"plans"`
	Total int64          `json:"total"`
}

type ListPlansUseCase struct {
	planRepo    subscription.PlanRepository
	pricingRepo subscription.PlanPricingRepository
	logger      logger.Interface
}

func NewListPlansUseCase(
	planRepo subscription.PlanRepository,
	pricingRepo subscription.PlanPricingRepository,
	logger logger.Interface,
) *ListPlansUseCase {
	return &ListPlansUseCase{
		planRepo:    planRepo,
		pricingRepo: pricingRepo,
		logger:      logger,
	}
}

func (uc *ListPlansUseCase) Execute(
	ctx context.Context,
	query ListPlansQuery,
) (*ListPlansResult, error) {
	filter := subscription.PlanFilter{
		Status:   query.Status,
		IsPublic: query.IsPublic,
		PlanType: query.PlanType,
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	plans, total, err := uc.planRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list plans", "error", err)
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}

	// Extract plan IDs for batch pricing query
	planIDs := make([]uint, 0, len(plans))
	for _, plan := range plans {
		planIDs = append(planIDs, plan.ID())
	}

	// Batch fetch all pricings in a single query (solves N+1 problem)
	pricingsByPlanID, err := uc.pricingRepo.GetActivePricingsByPlanIDs(ctx, planIDs)
	if err != nil {
		uc.logger.Warnw("failed to get pricings for plans", "plan_count", len(planIDs), "error", err)
		// Graceful degradation: continue without pricings
		pricingsByPlanID = make(map[uint][]*vo.PlanPricing)
	}

	// Build DTOs with pricings from the batch result
	planDTOs := make([]*dto.PlanDTO, 0, len(plans))
	for _, plan := range plans {
		pricings := pricingsByPlanID[plan.ID()]
		if len(pricings) > 0 {
			planDTOs = append(planDTOs, dto.ToPlanDTOWithPricings(plan, pricings))
		} else {
			planDTOs = append(planDTOs, dto.ToPlanDTO(plan))
		}
	}

	return &ListPlansResult{
		Plans: planDTOs,
		Total: total,
	}, nil
}
