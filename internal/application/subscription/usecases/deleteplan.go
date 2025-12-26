package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type DeletePlanUseCase struct {
	planRepo         subscription.PlanRepository
	subscriptionRepo subscription.SubscriptionRepository
	pricingRepo      subscription.PlanPricingRepository
	txMgr            *db.TransactionManager
	logger           logger.Interface
}

func NewDeletePlanUseCase(
	planRepo subscription.PlanRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	pricingRepo subscription.PlanPricingRepository,
	txMgr *db.TransactionManager,
	logger logger.Interface,
) *DeletePlanUseCase {
	return &DeletePlanUseCase{
		planRepo:         planRepo,
		subscriptionRepo: subscriptionRepo,
		pricingRepo:      pricingRepo,
		txMgr:            txMgr,
		logger:           logger,
	}
}

func (uc *DeletePlanUseCase) Execute(ctx context.Context, planSID string) error {
	// Get the plan by SID
	plan, err := uc.planRepo.GetBySID(ctx, planSID)
	if err != nil {
		uc.logger.Errorw("failed to get plan", "error", err, "plan_sid", planSID)
		return fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return errors.NewNotFoundError("plan not found")
	}

	// Check if there are any subscriptions using this plan
	count, err := uc.subscriptionRepo.CountByPlanID(ctx, plan.ID())
	if err != nil {
		uc.logger.Errorw("failed to count subscriptions", "error", err, "plan_id", plan.ID())
		return fmt.Errorf("failed to check plan usage: %w", err)
	}
	if count > 0 {
		return errors.NewConflictError(fmt.Sprintf("cannot delete plan: %d subscriptions are using this plan", count))
	}

	// Delete pricings and plan in a transaction
	planID := plan.ID()
	err = uc.txMgr.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Delete associated pricings first
		if err := uc.pricingRepo.DeleteByPlanID(txCtx, planID); err != nil {
			uc.logger.Errorw("failed to delete plan pricings", "error", err, "plan_id", planID)
			return fmt.Errorf("failed to delete plan pricings: %w", err)
		}

		// Delete the plan
		if err := uc.planRepo.Delete(txCtx, planID); err != nil {
			uc.logger.Errorw("failed to delete plan", "error", err, "plan_id", planID)
			return fmt.Errorf("failed to delete plan: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	uc.logger.Infow("plan deleted successfully", "plan_id", planID, "plan_sid", planSID)
	return nil
}
