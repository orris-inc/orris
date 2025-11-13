package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/subscription/dto"
	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"
)

type ListSubscriptionPlansQuery struct {
	Status       *string
	IsPublic     *bool
	BillingCycle *string
	Page         int
	PageSize     int
}

type ListSubscriptionPlansResult struct {
	Plans []*dto.SubscriptionPlanDTO `json:"plans"`
	Total int64                      `json:"total"`
}

type ListSubscriptionPlansUseCase struct {
	planRepo subscription.SubscriptionPlanRepository
	logger   logger.Interface
}

func NewListSubscriptionPlansUseCase(
	planRepo subscription.SubscriptionPlanRepository,
	logger logger.Interface,
) *ListSubscriptionPlansUseCase {
	return &ListSubscriptionPlansUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

func (uc *ListSubscriptionPlansUseCase) Execute(
	ctx context.Context,
	query ListSubscriptionPlansQuery,
) (*ListSubscriptionPlansResult, error) {
	filter := subscription.PlanFilter{
		Status:       query.Status,
		IsPublic:     query.IsPublic,
		BillingCycle: query.BillingCycle,
		Page:         query.Page,
		PageSize:     query.PageSize,
	}

	plans, total, err := uc.planRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list subscription plans", "error", err)
		return nil, fmt.Errorf("failed to list subscription plans: %w", err)
	}

	planDTOs := make([]*dto.SubscriptionPlanDTO, 0, len(plans))
	for _, plan := range plans {
		planDTOs = append(planDTOs, uc.toDTO(plan))
	}

	return &ListSubscriptionPlansResult{
		Plans: planDTOs,
		Total: total,
	}, nil
}

func (uc *ListSubscriptionPlansUseCase) toDTO(plan *subscription.SubscriptionPlan) *dto.SubscriptionPlanDTO {
	result := &dto.SubscriptionPlanDTO{
		ID:           plan.ID(),
		Name:         plan.Name(),
		Slug:         plan.Slug(),
		Description:  plan.Description(),
		Price:        plan.Price(),
		Currency:     plan.Currency(),
		BillingCycle: plan.BillingCycle().String(),
		TrialDays:    plan.TrialDays(),
		Status:       string(plan.Status()),
		APIRateLimit: plan.APIRateLimit(),
		MaxUsers:     plan.MaxUsers(),
		MaxProjects:  plan.MaxProjects(),
		IsPublic:     plan.IsPublic(),
		SortOrder:    plan.SortOrder(),
		CreatedAt:    plan.CreatedAt(),
		UpdatedAt:    plan.UpdatedAt(),
	}

	if plan.Features() != nil {
		result.Features = plan.Features().Features
		result.Limits = plan.Features().Limits
	}

	return result
}
