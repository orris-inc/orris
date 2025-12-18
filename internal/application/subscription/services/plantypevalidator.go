package services

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// PlanTypeValidator validates plan type consistency for resource associations
type PlanTypeValidator struct {
	planRepo subscription.PlanRepository
	logger   logger.Interface
}

// NewPlanTypeValidator creates a new PlanTypeValidator
func NewPlanTypeValidator(planRepo subscription.PlanRepository) *PlanTypeValidator {
	return &PlanTypeValidator{
		planRepo: planRepo,
		logger:   logger.NewLogger(),
	}
}

// ValidateNodePlanAssociation validates that planIDs contain only node-type plans
func (v *PlanTypeValidator) ValidateNodePlanAssociation(ctx context.Context, planIDs []uint) error {
	for _, planID := range planIDs {
		plan, err := v.planRepo.GetByID(ctx, planID)
		if err != nil {
			return fmt.Errorf("failed to get plan %d: %w", planID, err)
		}
		if plan == nil {
			return fmt.Errorf("plan %d not found", planID)
		}
		if !plan.PlanType().IsNode() {
			return fmt.Errorf("plan %d is type '%s', cannot associate with node (requires 'node' type)", planID, plan.PlanType())
		}
	}
	return nil
}

// ValidateForwardPlanAssociation validates that planIDs contain only forward-type plans
func (v *PlanTypeValidator) ValidateForwardPlanAssociation(ctx context.Context, planIDs []uint) error {
	for _, planID := range planIDs {
		plan, err := v.planRepo.GetByID(ctx, planID)
		if err != nil {
			return fmt.Errorf("failed to get plan %d: %w", planID, err)
		}
		if plan == nil {
			return fmt.Errorf("plan %d not found", planID)
		}
		if !plan.PlanType().IsForward() {
			return fmt.Errorf("plan %d is type '%s', cannot associate with forward agent (requires 'forward' type)", planID, plan.PlanType())
		}
	}
	return nil
}
