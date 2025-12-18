package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ListPlansQuery struct {
	Status       *string
	IsPublic     *bool
	BillingCycle *string
	Page         int
	PageSize     int
}

type ListPlansResult struct {
	Plans []*dto.PlanDTO `json:"plans"`
	Total int64          `json:"total"`
}

type ListPlansUseCase struct {
	planRepo subscription.PlanRepository
	logger   logger.Interface
}

func NewListPlansUseCase(
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *ListPlansUseCase {
	return &ListPlansUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

func (uc *ListPlansUseCase) Execute(
	ctx context.Context,
	query ListPlansQuery,
) (*ListPlansResult, error) {
	filter := subscription.PlanFilter{
		Status:       query.Status,
		IsPublic:     query.IsPublic,
		BillingCycle: query.BillingCycle,
		Page:         query.Page,
		PageSize:     query.PageSize,
	}

	plans, total, err := uc.planRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list plans", "error", err)
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}

	planDTOs := make([]*dto.PlanDTO, 0, len(plans))
	for _, plan := range plans {
		planDTOs = append(planDTOs, dto.ToPlanDTO(plan))
	}

	return &ListPlansResult{
		Plans: planDTOs,
		Total: total,
	}, nil
}
