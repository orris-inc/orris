package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/subscription/dto"
	"orris/internal/domain/subscription"
	vo "orris/internal/domain/subscription/value_objects"
	"orris/internal/shared/logger"
)

type UpdateSubscriptionPlanCommand struct {
	PlanID         uint
	Description    *string
	Price          *uint64
	Currency       *string
	Features       []string
	Limits         map[string]interface{}
	APIRateLimit   *uint
	MaxUsers       *uint
	MaxProjects    *uint
	SortOrder      *int
	IsPublic       *bool
}

type UpdateSubscriptionPlanUseCase struct {
	planRepo subscription.SubscriptionPlanRepository
	logger   logger.Interface
}

func NewUpdateSubscriptionPlanUseCase(
	planRepo subscription.SubscriptionPlanRepository,
	logger logger.Interface,
) *UpdateSubscriptionPlanUseCase {
	return &UpdateSubscriptionPlanUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

func (uc *UpdateSubscriptionPlanUseCase) Execute(
	ctx context.Context,
	cmd UpdateSubscriptionPlanCommand,
) (*dto.SubscriptionPlanDTO, error) {
	plan, err := uc.planRepo.GetByID(ctx, cmd.PlanID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan", "error", err, "plan_id", cmd.PlanID)
		return nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("subscription plan not found: %d", cmd.PlanID)
	}

	if cmd.Description != nil {
		plan.UpdateDescription(*cmd.Description)
	}

	if cmd.Price != nil && cmd.Currency != nil {
		if err := plan.UpdatePrice(*cmd.Price, *cmd.Currency); err != nil {
			uc.logger.Errorw("failed to update price", "error", err)
			return nil, fmt.Errorf("failed to update price: %w", err)
		}
	}

	if len(cmd.Features) > 0 || cmd.Limits != nil {
		features := vo.NewPlanFeatures(cmd.Features, cmd.Limits)
		if err := plan.UpdateFeatures(features); err != nil {
			uc.logger.Errorw("failed to update features", "error", err)
			return nil, fmt.Errorf("failed to update features: %w", err)
		}
	}

	if cmd.APIRateLimit != nil {
		if err := plan.SetAPIRateLimit(*cmd.APIRateLimit); err != nil {
			uc.logger.Errorw("failed to set API rate limit", "error", err)
			return nil, fmt.Errorf("failed to set API rate limit: %w", err)
		}
	}

	if cmd.MaxUsers != nil {
		plan.SetMaxUsers(*cmd.MaxUsers)
	}

	if cmd.MaxProjects != nil {
		plan.SetMaxProjects(*cmd.MaxProjects)
	}

	if cmd.SortOrder != nil {
		plan.SetSortOrder(*cmd.SortOrder)
	}

	if cmd.IsPublic != nil {
		plan.SetPublic(*cmd.IsPublic)
	}

	if err := uc.planRepo.Update(ctx, plan); err != nil {
		uc.logger.Errorw("failed to update subscription plan", "error", err, "plan_id", cmd.PlanID)
		return nil, fmt.Errorf("failed to update subscription plan: %w", err)
	}

	uc.logger.Infow("subscription plan updated successfully", "plan_id", plan.ID())

	return uc.toDTO(plan), nil
}

func (uc *UpdateSubscriptionPlanUseCase) toDTO(plan *subscription.SubscriptionPlan) *dto.SubscriptionPlanDTO {
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
