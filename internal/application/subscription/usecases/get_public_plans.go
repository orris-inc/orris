package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/subscription/dto"
	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"
)

type GetPublicPlansUseCase struct {
	planRepo subscription.SubscriptionPlanRepository
	logger   logger.Interface
}

func NewGetPublicPlansUseCase(
	planRepo subscription.SubscriptionPlanRepository,
	logger logger.Interface,
) *GetPublicPlansUseCase {
	return &GetPublicPlansUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

func (uc *GetPublicPlansUseCase) Execute(ctx context.Context) ([]*dto.SubscriptionPlanDTO, error) {
	plans, err := uc.planRepo.GetActivePublicPlans(ctx)
	if err != nil {
		uc.logger.Errorw("failed to get active public plans", "error", err)
		return nil, fmt.Errorf("failed to get active public plans: %w", err)
	}

	planDTOs := make([]*dto.SubscriptionPlanDTO, 0, len(plans))
	for _, plan := range plans {
		planDTOs = append(planDTOs, uc.toDTO(plan))
	}

	return planDTOs, nil
}

func (uc *GetPublicPlansUseCase) toDTO(plan *subscription.SubscriptionPlan) *dto.SubscriptionPlanDTO {
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
		CustomEndpoint: plan.CustomEndpoint(),
		APIRateLimit:   plan.APIRateLimit(),
		MaxUsers:       plan.MaxUsers(),
		MaxProjects:    plan.MaxProjects(),
		StorageLimit:   plan.StorageLimit(),
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
