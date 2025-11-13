package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/subscription/dto"
	"orris/internal/domain/subscription"
	vo "orris/internal/domain/subscription/value_objects"
	"orris/internal/shared/logger"
)

type CreateSubscriptionPlanCommand struct {
	Name         string
	Slug         string
	Description  string
	Price        uint64
	Currency     string
	BillingCycle string
	TrialDays    int
	Features     []string
	Limits       map[string]interface{}
	APIRateLimit uint
	MaxUsers     uint
	MaxProjects  uint
	IsPublic     bool
	SortOrder    int
}

type CreateSubscriptionPlanUseCase struct {
	planRepo subscription.SubscriptionPlanRepository
	logger   logger.Interface
}

func NewCreateSubscriptionPlanUseCase(
	planRepo subscription.SubscriptionPlanRepository,
	logger logger.Interface,
) *CreateSubscriptionPlanUseCase {
	return &CreateSubscriptionPlanUseCase{
		planRepo: planRepo,
		logger:   logger,
	}
}

func (uc *CreateSubscriptionPlanUseCase) Execute(
	ctx context.Context,
	cmd CreateSubscriptionPlanCommand,
) (*dto.SubscriptionPlanDTO, error) {
	exists, err := uc.planRepo.ExistsBySlug(ctx, cmd.Slug)
	if err != nil {
		uc.logger.Errorw("failed to check slug existence", "error", err, "slug", cmd.Slug)
		return nil, fmt.Errorf("failed to check slug existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("subscription plan with slug %s already exists", cmd.Slug)
	}

	billingCycle, err := vo.NewBillingCycle(cmd.BillingCycle)
	if err != nil {
		uc.logger.Errorw("invalid billing cycle", "error", err, "billing_cycle", cmd.BillingCycle)
		return nil, fmt.Errorf("invalid billing cycle: %w", err)
	}

	plan, err := subscription.NewSubscriptionPlan(
		cmd.Name,
		cmd.Slug,
		cmd.Description,
		cmd.Price,
		cmd.Currency,
		*billingCycle,
		cmd.TrialDays,
	)
	if err != nil {
		uc.logger.Errorw("failed to create subscription plan", "error", err)
		return nil, fmt.Errorf("failed to create subscription plan: %w", err)
	}

	if len(cmd.Features) > 0 || cmd.Limits != nil {
		features := vo.NewPlanFeatures(cmd.Features, cmd.Limits)
		if err := plan.UpdateFeatures(features); err != nil {
			uc.logger.Errorw("failed to set plan features", "error", err)
			return nil, fmt.Errorf("failed to set plan features: %w", err)
		}
	}

	if cmd.APIRateLimit > 0 {
		if err := plan.SetAPIRateLimit(cmd.APIRateLimit); err != nil {
			uc.logger.Errorw("failed to set API rate limit", "error", err)
			return nil, fmt.Errorf("failed to set API rate limit: %w", err)
		}
	}

	if cmd.MaxUsers > 0 {
		plan.SetMaxUsers(cmd.MaxUsers)
	}

	if cmd.MaxProjects > 0 {
		plan.SetMaxProjects(cmd.MaxProjects)
	}

	plan.SetPublic(cmd.IsPublic)

	if cmd.SortOrder != 0 {
		plan.SetSortOrder(cmd.SortOrder)
	}

	if err := uc.planRepo.Create(ctx, plan); err != nil {
		uc.logger.Errorw("failed to persist subscription plan", "error", err)
		return nil, fmt.Errorf("failed to persist subscription plan: %w", err)
	}

	uc.logger.Infow("subscription plan created successfully", "plan_id", plan.ID(), "slug", plan.Slug())

	return uc.toDTO(plan), nil
}

func (uc *CreateSubscriptionPlanUseCase) toDTO(plan *subscription.SubscriptionPlan) *dto.SubscriptionPlanDTO {
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
