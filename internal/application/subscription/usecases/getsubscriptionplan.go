package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/subscription/dto"
	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"
)

type GetSubscriptionPlanUseCase struct {
	planRepo subscription.SubscriptionPlanRepository
	logger   logger.Interface
}

func NewGetSubscriptionPlanUseCase(
	planRepo subscription.SubscriptionPlanRepository,
	logger logger.Interface,
) *GetSubscriptionPlanUseCase {
	return &GetSubscriptionPlanUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

func (uc *GetSubscriptionPlanUseCase) ExecuteByID(ctx context.Context, planID uint) (*dto.SubscriptionPlanDTO, error) {
	plan, err := uc.planRepo.GetByID(ctx, planID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan by ID", "error", err, "plan_id", planID)
		return nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("subscription plan not found: %d", planID)
	}

	return uc.toDTO(plan), nil
}

func (uc *GetSubscriptionPlanUseCase) ExecuteBySlug(ctx context.Context, slug string) (*dto.SubscriptionPlanDTO, error) {
	plan, err := uc.planRepo.GetBySlug(ctx, slug)
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan by slug", "error", err, "slug", slug)
		return nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("subscription plan not found: %s", slug)
	}

	return uc.toDTO(plan), nil
}

func (uc *GetSubscriptionPlanUseCase) toDTO(plan *subscription.SubscriptionPlan) *dto.SubscriptionPlanDTO {
	result := &dto.SubscriptionPlanDTO{
		ID:             plan.ID(),
		Name:           plan.Name(),
		Slug:           plan.Slug(),
		Description:    plan.Description(),
		Price:          plan.Price(),
		Currency:       plan.Currency(),
		BillingCycle:   plan.BillingCycle().String(),
		TrialDays:      plan.TrialDays(),
		Status:         string(plan.Status()),
		APIRateLimit:   plan.APIRateLimit(),
		MaxUsers:       plan.MaxUsers(),
		MaxProjects:    plan.MaxProjects(),
		IsPublic:       plan.IsPublic(),
		SortOrder:      plan.SortOrder(),
		CreatedAt:      plan.CreatedAt(),
		UpdatedAt:      plan.UpdatedAt(),
	}

	if plan.Features() != nil {
		result.Features = plan.Features().Features
		result.Limits = plan.Features().Limits
	}

	return result
}
