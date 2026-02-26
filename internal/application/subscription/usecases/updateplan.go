package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type UpdatePlanCommand struct {
	PlanSID     string
	Description *string
	Limits      *map[string]interface{}
	NodeLimit   *int // Maximum number of user nodes (nil or 0 = unlimited)
	SortOrder   *int
	IsPublic    *bool
	Pricings    *[]dto.PricingOptionInput // Optional: update pricing options
}

// PlanChangeNotifier notifies nodes when plan features change
type PlanChangeNotifier interface {
	NotifyPlanFeaturesChanged(ctx context.Context, planID uint) error
}

type UpdatePlanUseCase struct {
	planRepo           subscription.PlanRepository
	pricingRepo        subscription.PlanPricingRepository
	planChangeNotifier PlanChangeNotifier
	logger             logger.Interface
}

// SetPlanChangeNotifier sets the notifier for plan feature changes.
func (uc *UpdatePlanUseCase) SetPlanChangeNotifier(notifier PlanChangeNotifier) {
	uc.planChangeNotifier = notifier
}

func NewUpdatePlanUseCase(
	planRepo subscription.PlanRepository,
	pricingRepo subscription.PlanPricingRepository,
	logger logger.Interface,
) *UpdatePlanUseCase {
	return &UpdatePlanUseCase{
		planRepo:    planRepo,
		pricingRepo: pricingRepo,
		logger:      logger,
	}
}

func (uc *UpdatePlanUseCase) Execute(
	ctx context.Context,
	cmd UpdatePlanCommand,
) (*dto.PlanDTO, error) {
	plan, err := uc.planRepo.GetBySID(ctx, cmd.PlanSID)
	if err != nil {
		uc.logger.Errorw("failed to get plan", "error", err, "plan_sid", cmd.PlanSID)
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found: %s", cmd.PlanSID)
	}

	if cmd.Description != nil {
		plan.UpdateDescription(*cmd.Description)
	}

	if cmd.Limits != nil {
		features, err := vo.NewPlanFeaturesWithValidation(*cmd.Limits)
		if err != nil {
			uc.logger.Warnw("invalid plan limits", "error", err)
			return nil, err
		}
		if err := plan.UpdateFeatures(features); err != nil {
			uc.logger.Errorw("failed to update features", "error", err)
			return nil, err
		}
	}

	if cmd.SortOrder != nil {
		plan.SetSortOrder(*cmd.SortOrder)
	}

	if cmd.IsPublic != nil {
		plan.SetPublic(*cmd.IsPublic)
	}

	if cmd.NodeLimit != nil {
		plan.SetNodeLimit(cmd.NodeLimit)
	}

	if err := uc.planRepo.Update(ctx, plan); err != nil {
		uc.logger.Errorw("failed to update plan", "error", err, "plan_id", plan.ID())
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}

	planID := plan.ID()

	// Notify affected nodes when plan features (e.g. device_limit) change
	if cmd.Limits != nil && uc.planChangeNotifier != nil {
		if err := uc.planChangeNotifier.NotifyPlanFeaturesChanged(ctx, planID); err != nil {
			uc.logger.Warnw("failed to notify nodes of plan features change",
				"plan_id", planID,
				"error", err,
			)
			// Don't fail the update operation
		}
	}

	// Sync pricing options if provided (delete old, create new)
	if cmd.Pricings != nil {
		uc.logger.Infow("syncing pricing options", "plan_id", planID, "count", len(*cmd.Pricings))

		// Delete all existing pricings for this plan
		if err := uc.pricingRepo.DeleteByPlanID(ctx, planID); err != nil {
			uc.logger.Errorw("failed to delete existing pricings",
				"error", err,
				"plan_id", planID)
			return nil, fmt.Errorf("failed to delete existing pricings: %w", err)
		}

		// Create new pricings
		for _, pricingInput := range *cmd.Pricings {
			// Validate billing cycle
			cycle, err := vo.NewBillingCycle(pricingInput.BillingCycle)
			if err != nil {
				uc.logger.Warnw("invalid billing cycle in pricing",
					"error", err,
					"billing_cycle", pricingInput.BillingCycle,
					"plan_id", planID)
				return nil, err
			}

			// Create pricing value object
			pricing, err := vo.NewPlanPricing(planID, *cycle, pricingInput.Price, pricingInput.Currency)
			if err != nil {
				uc.logger.Errorw("failed to create pricing",
					"error", err,
					"plan_id", planID,
					"billing_cycle", pricingInput.BillingCycle)
				return nil, err
			}

			// Set active status if explicitly set to false
			if !pricingInput.IsActive {
				pricing.Deactivate()
			}

			// Persist pricing
			if err := uc.pricingRepo.Create(ctx, pricing); err != nil {
				uc.logger.Errorw("failed to persist pricing",
					"error", err,
					"plan_id", planID,
					"billing_cycle", pricingInput.BillingCycle)
				return nil, fmt.Errorf("failed to persist pricing: %w", err)
			}
		}

		uc.logger.Infow("pricing options synced successfully",
			"plan_id", planID,
			"count", len(*cmd.Pricings))
	}

	// Reload the plan from database to get the accurate state after update
	updatedPlan, err := uc.planRepo.GetByID(ctx, planID)
	if err != nil {
		uc.logger.Errorw("failed to reload updated plan", "error", err, "plan_id", planID)
		return nil, fmt.Errorf("failed to reload updated plan: %w", err)
	}

	uc.logger.Infow("plan updated successfully", "plan_id", updatedPlan.ID())

	// Fetch pricings to include in response
	pricings, err := uc.pricingRepo.GetByPlanID(ctx, updatedPlan.ID())
	if err != nil {
		uc.logger.Warnw("failed to fetch pricings for response",
			"error", err,
			"plan_id", updatedPlan.ID())
		// Don't fail the request, just return plan without pricings
		return dto.ToPlanDTO(updatedPlan), nil
	}

	return dto.ToPlanDTOWithPricings(updatedPlan, pricings), nil
}
