package usecases

import (
	"context"
	"fmt"
	"math"

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
	planRepo             subscription.PlanRepository
	pricingRepo          subscription.PlanPricingRepository
	subscriptionRepo     subscription.SubscriptionRepository
	planChangeNotifier   PlanChangeNotifier
	quotaCacheManager    QuotaCacheManager
	subscriptionNotifier SubscriptionChangeNotifier
	logger               logger.Interface
}

// SetPlanChangeNotifier sets the notifier for plan feature changes.
func (uc *UpdatePlanUseCase) SetPlanChangeNotifier(notifier PlanChangeNotifier) {
	uc.planChangeNotifier = notifier
}

// SetSubscriptionRepo sets the subscription repository for cascading plan changes.
func (uc *UpdatePlanUseCase) SetSubscriptionRepo(repo subscription.SubscriptionRepository) {
	uc.subscriptionRepo = repo
}

// SetQuotaCacheManager sets the quota cache manager for invalidating cached quotas.
func (uc *UpdatePlanUseCase) SetQuotaCacheManager(manager QuotaCacheManager) {
	uc.quotaCacheManager = manager
}

// SetSubscriptionNotifier sets the notifier for subscription changes.
func (uc *UpdatePlanUseCase) SetSubscriptionNotifier(notifier SubscriptionChangeNotifier) {
	uc.subscriptionNotifier = notifier
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

	// Capture old traffic reset mode before updating features
	oldTrafficResetMode := subscription.GetTrafficResetMode(plan)

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

	// Invalidate quota cache for all active subscriptions when limits change
	if cmd.Limits != nil {
		uc.invalidateQuotaCacheForPlan(ctx, planID)
	}

	// Reset subscription usage when traffic reset mode changes
	if cmd.Limits != nil {
		newTrafficResetMode := subscription.GetTrafficResetMode(plan)
		if oldTrafficResetMode != newTrafficResetMode {
			uc.resetSubscriptionUsageForPlan(ctx, planID, oldTrafficResetMode, newTrafficResetMode)
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

			// Deactivate only when explicitly set to false (nil = active by default)
			if pricingInput.IsActive != nil && !*pricingInput.IsActive {
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

		// Migrate orphaned subscriptions whose billing cycle is no longer available
		uc.migrateOrphanedBillingCycles(ctx, planID, *cmd.Pricings)
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

// invalidateQuotaCacheForPlan invalidates Redis quota cache for all active
// subscriptions on the given plan so enforcement uses updated limits.
func (uc *UpdatePlanUseCase) invalidateQuotaCacheForPlan(ctx context.Context, planID uint) {
	if uc.subscriptionRepo == nil || uc.quotaCacheManager == nil {
		return
	}

	subs, _, err := uc.subscriptionRepo.List(ctx, subscription.SubscriptionFilter{
		PlanID:   &planID,
		Statuses: []string{string(vo.StatusActive), string(vo.StatusTrialing)},
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		uc.logger.Warnw("failed to list subscriptions for quota cache invalidation",
			"plan_id", planID, "error", err)
		return
	}

	for _, sub := range subs {
		if err := uc.quotaCacheManager.InvalidateQuota(ctx, sub.ID()); err != nil {
			uc.logger.Warnw("failed to invalidate quota cache",
				"subscription_id", sub.ID(), "error", err)
		}
	}

	if len(subs) > 0 {
		uc.logger.Infow("quota cache invalidated for plan subscriptions",
			"plan_id", planID, "count", len(subs))
	}
}

// resetSubscriptionUsageForPlan resets traffic usage for all active/trialing/suspended
// subscriptions on the given plan when the traffic reset mode changes.
// This ensures a clean start under the new mode and avoids unfair traffic accumulation
// (e.g. switching from calendar_month to billing_cycle on a lifetime plan would otherwise
// count all historical traffic since the subscription started).
func (uc *UpdatePlanUseCase) resetSubscriptionUsageForPlan(
	ctx context.Context, planID uint,
	oldMode, newMode subscription.TrafficResetMode,
) {
	if uc.subscriptionRepo == nil {
		return
	}

	subs, _, err := uc.subscriptionRepo.List(ctx, subscription.SubscriptionFilter{
		PlanID:   &planID,
		Statuses: []string{string(vo.StatusActive), string(vo.StatusTrialing), string(vo.StatusSuspended)},
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		uc.logger.Warnw("failed to list subscriptions for traffic reset mode migration",
			"plan_id", planID, "error", err)
		return
	}

	resetCount := 0
	for _, sub := range subs {
		// Lifetime subscriptions are exempt from automatic resets triggered by plan mode changes.
		// Their traffic period always spans the full subscription lifetime and is never
		// subject to calendar-month boundaries, so resetting here would cause unintended data loss.
		if sub.BillingCycle() != nil && sub.BillingCycle().IsLifetime() {
			uc.logger.Infow("skipping lifetime subscription during traffic reset mode migration",
				"subscription_id", sub.ID())
			continue
		}

		if err := sub.ResetUsage(); err != nil {
			uc.logger.Warnw("failed to reset subscription usage after traffic reset mode change",
				"subscription_id", sub.ID(), "error", err)
			continue
		}

		if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
			uc.logger.Warnw("failed to persist subscription after traffic reset mode change",
				"subscription_id", sub.ID(), "error", err)
			continue
		}

		// Invalidate quota cache
		if uc.quotaCacheManager != nil {
			_ = uc.quotaCacheManager.InvalidateQuota(ctx, sub.ID())
		}

		// Notify nodes of subscription update
		if uc.subscriptionNotifier != nil {
			_ = uc.subscriptionNotifier.NotifySubscriptionUpdate(ctx, sub)
		}

		resetCount++
	}

	if resetCount > 0 {
		uc.logger.Infow("subscription usage reset after traffic reset mode change",
			"plan_id", planID,
			"old_mode", string(oldMode),
			"new_mode", string(newMode),
			"reset_count", resetCount,
		)
	}
}

// migrateOrphanedBillingCycles migrates subscriptions whose billing cycle
// is no longer available in the new pricing options.
// e.g., plan changed from monthly to lifetime -> existing monthly subscriptions
// are migrated to lifetime with recalculated end_date.
func (uc *UpdatePlanUseCase) migrateOrphanedBillingCycles(
	ctx context.Context, planID uint, newPricings []dto.PricingOptionInput,
) {
	if uc.subscriptionRepo == nil {
		return
	}

	// Build set of active billing cycles from new pricings
	availableCycles := make(map[string]bool)
	for _, p := range newPricings {
		if p.IsActive == nil || *p.IsActive {
			availableCycles[p.BillingCycle] = true
		}
	}
	if len(availableCycles) == 0 {
		return
	}

	// Query affected subscriptions
	subs, _, err := uc.subscriptionRepo.List(ctx, subscription.SubscriptionFilter{
		PlanID:   &planID,
		Statuses: []string{string(vo.StatusActive), string(vo.StatusTrialing), string(vo.StatusSuspended)},
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		uc.logger.Warnw("failed to list subscriptions for billing cycle migration",
			"plan_id", planID, "error", err)
		return
	}

	// Find orphaned subscriptions
	migratedCount := 0
	for _, sub := range subs {
		if sub.BillingCycle() == nil {
			continue
		}
		if availableCycles[sub.BillingCycle().String()] {
			continue // billing cycle still available
		}

		oldCycle := sub.BillingCycle().String()
		targetCycle := findClosestBillingCycle(sub.BillingCycle(), availableCycles)
		newEndDate := CalculateEndDate(sub.CurrentPeriodStart(), targetCycle)

		if err := sub.ChangeBillingCycle(targetCycle, newEndDate); err != nil {
			uc.logger.Warnw("failed to change billing cycle",
				"subscription_id", sub.ID(), "old_cycle", oldCycle, "error", err)
			continue
		}

		if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
			uc.logger.Warnw("failed to persist subscription after billing cycle migration",
				"subscription_id", sub.ID(), "error", err)
			continue
		}

		// Invalidate quota cache for migrated subscription
		if uc.quotaCacheManager != nil {
			_ = uc.quotaCacheManager.InvalidateQuota(ctx, sub.ID())
		}

		// Notify nodes of subscription update
		if uc.subscriptionNotifier != nil {
			_ = uc.subscriptionNotifier.NotifySubscriptionUpdate(ctx, sub)
		}

		uc.logger.Infow("subscription billing cycle migrated",
			"subscription_id", sub.ID(),
			"subscription_sid", sub.SID(),
			"old_cycle", oldCycle,
			"new_cycle", targetCycle.String(),
			"new_end_date", newEndDate,
		)
		migratedCount++
	}

	if migratedCount > 0 {
		uc.logger.Infow("billing cycle migration completed",
			"plan_id", planID, "migrated_count", migratedCount)
	}
}

// findClosestBillingCycle finds the billing cycle from availableCycles
// that is closest in duration to the given oldCycle.
// On tie, prefers the longer duration to minimize disruption.
func findClosestBillingCycle(oldCycle *vo.BillingCycle, availableCycles map[string]bool) vo.BillingCycle {
	oldDays := 0
	if oldCycle != nil {
		oldDays = oldCycle.Days()
		if oldCycle.IsLifetime() {
			oldDays = math.MaxInt32
		}
	}

	var bestCycle vo.BillingCycle
	bestDiff := math.MaxInt32
	bestDays := 0

	for c := range availableCycles {
		parsed, err := vo.ParseBillingCycle(c)
		if err != nil {
			continue
		}
		days := parsed.Days()
		if parsed.IsLifetime() {
			days = math.MaxInt32
		}

		diff := oldDays - days
		if diff < 0 {
			diff = -diff
		}
		// Prefer closer; on tie prefer longer duration
		if diff < bestDiff || (diff == bestDiff && days > bestDays) {
			bestDiff = diff
			bestCycle = parsed
			bestDays = days
		}
	}

	return bestCycle
}
