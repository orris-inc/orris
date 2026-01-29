package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type CreatePlanCommand struct {
	Name        string
	Slug        string
	Description string
	PlanType    string // Required: "node" or "forward"
	Limits      map[string]interface{}
	IsPublic    bool
	SortOrder   int
	Pricings    []dto.PricingOptionInput // Required: multiple pricing options
}

type CreatePlanUseCase struct {
	planRepo    subscription.PlanRepository
	pricingRepo subscription.PlanPricingRepository
	logger      logger.Interface
}

func NewCreatePlanUseCase(
	planRepo subscription.PlanRepository,
	pricingRepo subscription.PlanPricingRepository,
	logger logger.Interface,
) *CreatePlanUseCase {
	return &CreatePlanUseCase{
		planRepo:    planRepo,
		pricingRepo: pricingRepo,
		logger:      logger,
	}
}

func (uc *CreatePlanUseCase) Execute(
	ctx context.Context,
	cmd CreatePlanCommand,
) (*dto.PlanDTO, error) {
	// Validate pricings array - at least one pricing option is required
	if len(cmd.Pricings) == 0 {
		uc.logger.Errorw("pricings array is empty", "slug", cmd.Slug)
		return nil, fmt.Errorf("at least one pricing option is required")
	}

	exists, err := uc.planRepo.ExistsBySlug(ctx, cmd.Slug)
	if err != nil {
		uc.logger.Errorw("failed to check slug existence", "error", err, "slug", cmd.Slug)
		return nil, fmt.Errorf("failed to check slug existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("plan with slug %s already exists", cmd.Slug)
	}

	planType, err := vo.NewPlanType(cmd.PlanType)
	if err != nil {
		uc.logger.Errorw("invalid plan type", "error", err, "plan_type", cmd.PlanType)
		return nil, fmt.Errorf("invalid plan type: %w", err)
	}

	plan, err := subscription.NewPlan(
		cmd.Name,
		cmd.Slug,
		cmd.Description,
		planType,
	)
	if err != nil {
		uc.logger.Errorw("failed to create plan", "error", err)
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	if cmd.Limits != nil {
		features, err := vo.NewPlanFeaturesWithValidation(cmd.Limits)
		if err != nil {
			uc.logger.Errorw("invalid plan limits", "error", err)
			return nil, fmt.Errorf("invalid plan limits: %w", err)
		}
		if err := plan.UpdateFeatures(features); err != nil {
			uc.logger.Errorw("failed to set plan features", "error", err)
			return nil, fmt.Errorf("failed to set plan features: %w", err)
		}
	}

	plan.SetPublic(cmd.IsPublic)

	if cmd.SortOrder != 0 {
		plan.SetSortOrder(cmd.SortOrder)
	}

	if err := uc.planRepo.Create(ctx, plan); err != nil {
		uc.logger.Errorw("failed to persist plan", "error", err)
		return nil, fmt.Errorf("failed to persist plan: %w", err)
	}

	// Create pricing options (guaranteed to have at least one)
	uc.logger.Infow("creating pricing options", "plan_id", plan.ID(), "count", len(cmd.Pricings))

	for _, pricingInput := range cmd.Pricings {
		// Validate billing cycle
		cycle, err := vo.NewBillingCycle(pricingInput.BillingCycle)
		if err != nil {
			uc.logger.Errorw("invalid billing cycle in pricing",
				"error", err,
				"billing_cycle", pricingInput.BillingCycle,
				"plan_id", plan.ID())
			return nil, fmt.Errorf("invalid billing cycle '%s': %w", pricingInput.BillingCycle, err)
		}

		// Create pricing value object
		pricing, err := vo.NewPlanPricing(plan.ID(), *cycle, pricingInput.Price, pricingInput.Currency)
		if err != nil {
			uc.logger.Errorw("failed to create pricing",
				"error", err,
				"plan_id", plan.ID(),
				"billing_cycle", pricingInput.BillingCycle)
			return nil, fmt.Errorf("failed to create pricing for cycle '%s': %w", pricingInput.BillingCycle, err)
		}

		// Set active status if explicitly set to false
		if !pricingInput.IsActive {
			pricing.Deactivate()
		}

		// Persist pricing
		if err := uc.pricingRepo.Create(ctx, pricing); err != nil {
			uc.logger.Errorw("failed to persist pricing",
				"error", err,
				"plan_id", plan.ID(),
				"billing_cycle", pricingInput.BillingCycle)
			return nil, fmt.Errorf("failed to persist pricing: %w", err)
		}
	}

	uc.logger.Infow("pricing options created successfully",
		"plan_id", plan.ID(),
		"count", len(cmd.Pricings))

	uc.logger.Infow("plan created successfully", "plan_id", plan.ID(), "slug", plan.Slug())

	// Fetch pricings to include in response
	pricings, err := uc.pricingRepo.GetByPlanID(ctx, plan.ID())
	if err != nil {
		uc.logger.Warnw("failed to fetch pricings for response",
			"error", err,
			"plan_id", plan.ID())
		// Don't fail the request, just return plan without pricings
		return dto.ToPlanDTO(plan), nil
	}

	return dto.ToPlanDTOWithPricings(plan, pricings), nil
}
