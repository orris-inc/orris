package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"
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
	NewPlanID      uint
	ChangeType     ChangeType
	EffectiveDate  EffectiveDate
}

type ChangePlanUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.SubscriptionPlanRepository
	logger           logger.Interface
}

func NewChangePlanUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.SubscriptionPlanRepository,
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

	oldPlan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get old plan", "error", err, "plan_id", sub.PlanID())
		return fmt.Errorf("failed to get old plan: %w", err)
	}

	newPlan, err := uc.planRepo.GetByID(ctx, cmd.NewPlanID)
	if err != nil {
		uc.logger.Errorw("failed to get new plan", "error", err, "plan_id", cmd.NewPlanID)
		return fmt.Errorf("failed to get new plan: %w", err)
	}

	if !newPlan.IsActive() {
		return fmt.Errorf("new plan is not active")
	}

	actualChangeType := uc.determineChangeType(oldPlan, newPlan)
	if actualChangeType != cmd.ChangeType {
		uc.logger.Warnw("change type mismatch",
			"requested", cmd.ChangeType,
			"actual", actualChangeType,
			"old_price", oldPlan.Price(),
			"new_price", newPlan.Price(),
		)
		return fmt.Errorf("change type mismatch: requested %s but actual is %s based on price", cmd.ChangeType, actualChangeType)
	}

	if cmd.EffectiveDate == EffectiveDatePeriodEnd {
		metadata := sub.Metadata()
		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		metadata["pending_plan_change"] = map[string]interface{}{
			"new_plan_id":    cmd.NewPlanID,
			"change_type":    string(cmd.ChangeType),
			"effective_date": string(cmd.EffectiveDate),
		}

		uc.logger.Infow("plan change scheduled for period end",
			"subscription_id", cmd.SubscriptionID,
			"old_plan_id", sub.PlanID(),
			"new_plan_id", cmd.NewPlanID,
			"change_type", cmd.ChangeType,
		)
	} else {
		if err := uc.applyPlanChange(sub, cmd.NewPlanID, cmd.ChangeType); err != nil {
			uc.logger.Errorw("failed to apply plan change", "error", err)
			return fmt.Errorf("failed to apply plan change: %w", err)
		}

		uc.logger.Infow("plan changed immediately",
			"subscription_id", cmd.SubscriptionID,
			"old_plan_id", oldPlan.ID(),
			"new_plan_id", cmd.NewPlanID,
			"change_type", cmd.ChangeType,
		)
	}

	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		uc.logger.Errorw("failed to update subscription", "error", err)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

func (uc *ChangePlanUseCase) determineChangeType(oldPlan, newPlan *subscription.SubscriptionPlan) ChangeType {
	if newPlan.Price() > oldPlan.Price() {
		return ChangeTypeUpgrade
	}
	return ChangeTypeDowngrade
}

func (uc *ChangePlanUseCase) applyPlanChange(sub *subscription.Subscription, newPlanID uint, changeType ChangeType) error {
	switch changeType {
	case ChangeTypeUpgrade:
		return sub.UpgradePlan(newPlanID)
	case ChangeTypeDowngrade:
		return sub.DowngradePlan(newPlanID)
	default:
		return fmt.Errorf("invalid change type: %s", changeType)
	}
}
