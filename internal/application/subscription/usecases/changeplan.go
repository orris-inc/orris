package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	apperrors "github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ChangeType string

const (
	ChangeTypeUpgrade   ChangeType = "upgrade"
	ChangeTypeDowngrade ChangeType = "downgrade"
)

type EffectiveDate string

const (
	EffectiveDateImmediate EffectiveDate = "immediate"
	EffectiveDatePeriodEnd EffectiveDate = "period_end"
)

type ChangePlanCommand struct {
	SubscriptionID uint
	NewPlanID      uint   // Internal plan ID (used if NewPlanSID is empty)
	NewPlanSID     string // Stripe-style plan SID (takes precedence over NewPlanID)
	ChangeType     ChangeType
	EffectiveDate  EffectiveDate
}

type ChangePlanUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	logger           logger.Interface
}

func NewChangePlanUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *ChangePlanUseCase {
	return &ChangePlanUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}

func (uc *ChangePlanUseCase) Execute(ctx context.Context, cmd ChangePlanCommand) error {
	sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return apperrors.NewNotFoundError("subscription not found")
	}

	oldPlan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get old plan", "error", err, "plan_id", sub.PlanID())
		return fmt.Errorf("failed to get old plan: %w", err)
	}

	// Resolve new plan: prefer SID over internal ID
	var newPlan *subscription.Plan
	newPlanID := cmd.NewPlanID

	if cmd.NewPlanSID != "" {
		newPlan, err = uc.planRepo.GetBySID(ctx, cmd.NewPlanSID)
		if err != nil {
			uc.logger.Errorw("failed to get new plan by SID", "error", err, "plan_sid", cmd.NewPlanSID)
			return fmt.Errorf("failed to get new plan: %w", err)
		}
		if newPlan == nil {
			uc.logger.Warnw("new plan not found by SID", "plan_sid", cmd.NewPlanSID)
			return apperrors.NewNotFoundError("new plan not found")
		}
		newPlanID = newPlan.ID()
	} else {
		newPlan, err = uc.planRepo.GetByID(ctx, cmd.NewPlanID)
		if err != nil {
			uc.logger.Errorw("failed to get new plan", "error", err, "plan_id", cmd.NewPlanID)
			return fmt.Errorf("failed to get new plan: %w", err)
		}
		if newPlan == nil {
			return apperrors.NewNotFoundError("new plan not found")
		}
	}

	if !newPlan.IsActive() {
		return apperrors.NewValidationError("new plan is not active")
	}

	// Note: Change type validation is removed as pricing is now flexible per billing cycle

	if cmd.EffectiveDate == EffectiveDatePeriodEnd {
		metadata := sub.Metadata()
		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		metadata["pending_plan_change"] = map[string]interface{}{
			"new_plan_id":    newPlanID,
			"change_type":    string(cmd.ChangeType),
			"effective_date": string(cmd.EffectiveDate),
		}

		uc.logger.Infow("plan change scheduled for period end",
			"subscription_id", cmd.SubscriptionID,
			"old_plan_id", sub.PlanID(),
			"new_plan_id", newPlanID,
			"change_type", cmd.ChangeType,
		)
	} else {
		if err := uc.applyPlanChange(sub, newPlanID, cmd.ChangeType); err != nil {
			uc.logger.Errorw("failed to apply plan change", "error", err)
			return err
		}

		uc.logger.Infow("plan changed immediately",
			"subscription_id", cmd.SubscriptionID,
			"old_plan_id", oldPlan.ID(),
			"new_plan_id", newPlanID,
			"change_type", cmd.ChangeType,
		)
	}

	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		uc.logger.Errorw("failed to update subscription", "error", err)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

func (uc *ChangePlanUseCase) applyPlanChange(sub *subscription.Subscription, newPlanID uint, changeType ChangeType) error {
	return sub.ChangePlan(newPlanID)
}
