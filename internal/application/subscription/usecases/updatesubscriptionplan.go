package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type UpdateSubscriptionPlanCommand struct {
	PlanID       uint
	Description  *string
	Price        *uint64
	Currency     *string
	Features     *[]string
	Limits       *map[string]interface{}
	APIRateLimit *uint
	MaxUsers     *uint
	MaxProjects  *uint
	SortOrder    *int
	IsPublic     *bool
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

	if cmd.Features != nil || cmd.Limits != nil {
		var featuresList []string
		var limitsMap map[string]interface{}

		if cmd.Features != nil {
			featuresList = *cmd.Features
		} else {
			// Keep existing features if not provided
			if plan.Features() != nil {
				featuresList = plan.Features().Features
			}
		}

		if cmd.Limits != nil {
			limitsMap = *cmd.Limits
		} else {
			// Keep existing limits if not provided
			if plan.Features() != nil {
				limitsMap = plan.Features().Limits
			}
		}

		features := vo.NewPlanFeatures(featuresList, limitsMap)
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

	// Reload the plan from database to get the accurate state after update
	updatedPlan, err := uc.planRepo.GetByID(ctx, cmd.PlanID)
	if err != nil {
		uc.logger.Errorw("failed to reload updated plan", "error", err, "plan_id", cmd.PlanID)
		return nil, fmt.Errorf("failed to reload updated plan: %w", err)
	}

	uc.logger.Infow("subscription plan updated successfully", "plan_id", updatedPlan.ID())

	return dto.ToSubscriptionPlanDTO(updatedPlan), nil
}
