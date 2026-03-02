package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	apperrors "github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdateSubscriptionCommand represents the command to update a subscription's overridable fields.
// All fields are pointers; nil means "no change".
type UpdateSubscriptionCommand struct {
	SubscriptionID uint
	StartDate      *time.Time // Update start date
	EndDate        *time.Time // Update end date
	DataLimitBytes *uint64    // Override traffic limit (nil in command = no change; value 0 handled separately to clear)
	ClearDataLimit bool       // If true, clear the traffic limit override
	DataUsedBytes  *uint64    // Target used bytes; UseCase computes adjustment = target - actual
}

// UpdateSubscriptionUseCase handles admin subscription editing.
type UpdateSubscriptionUseCase struct {
	subscriptionRepo     subscription.SubscriptionRepository
	planRepo             subscription.PlanRepository
	quotaService         QuotaService
	quotaCacheManager    QuotaCacheManager
	subscriptionNotifier SubscriptionChangeNotifier
	logger               logger.Interface
}

// NewUpdateSubscriptionUseCase creates a new UpdateSubscriptionUseCase.
func NewUpdateSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	quotaService QuotaService,
	logger logger.Interface,
) *UpdateSubscriptionUseCase {
	return &UpdateSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		quotaService:     quotaService,
		logger:           logger,
	}
}

// SetQuotaService sets the quota service for usage calculation (optional, late-bound).
func (uc *UpdateSubscriptionUseCase) SetQuotaService(qs QuotaService) {
	uc.quotaService = qs
}

// SetQuotaCacheManager sets the quota cache manager (optional).
func (uc *UpdateSubscriptionUseCase) SetQuotaCacheManager(manager QuotaCacheManager) {
	uc.quotaCacheManager = manager
}

// SetSubscriptionNotifier sets the subscription change notifier (optional).
func (uc *UpdateSubscriptionUseCase) SetSubscriptionNotifier(notifier SubscriptionChangeNotifier) {
	uc.subscriptionNotifier = notifier
}

// Execute performs the subscription update.
func (uc *UpdateSubscriptionUseCase) Execute(ctx context.Context, cmd UpdateSubscriptionCommand) error {
	sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return apperrors.NewNotFoundError("subscription not found")
	}

	// Apply date changes
	if cmd.StartDate != nil || cmd.EndDate != nil {
		if err := sub.UpdateDates(cmd.StartDate, cmd.EndDate); err != nil {
			uc.logger.Warnw("failed to update dates", "error", err, "subscription_id", cmd.SubscriptionID)
			return apperrors.NewValidationError("failed to update dates", err.Error())
		}
	}

	// Apply traffic limit override
	if cmd.ClearDataLimit {
		sub.ClearTrafficLimitOverride()
	} else if cmd.DataLimitBytes != nil {
		sub.SetTrafficLimitOverride(*cmd.DataLimitBytes)
	}

	// Apply traffic used adjustment
	if cmd.DataUsedBytes != nil {
		targetUsed := *cmd.DataUsedBytes

		plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
		if err != nil {
			uc.logger.Errorw("failed to get plan", "error", err, "plan_id", sub.PlanID())
			return fmt.Errorf("failed to get plan: %w", err)
		}
		if plan == nil {
			return apperrors.NewNotFoundError("plan not found")
		}

		// Resolve traffic period
		period := subscription.ResolveTrafficPeriod(plan, sub)

		// GetCurrentPeriodUsage returns raw usage (without adjustment applied),
		// so actualUsage IS the raw actual usage — no need to subtract existing adjustment.
		actualUsage, err := uc.quotaService.GetCurrentPeriodUsage(ctx, sub.ID(), period.Start, period.End)
		if err != nil {
			uc.logger.Errorw("failed to get current usage", "error", err, "subscription_id", cmd.SubscriptionID)
			return fmt.Errorf("failed to get current usage: %w", err)
		}

		// Compute adjustment so that displayed usage = target: adjustment = target - rawActual
		adjustment := int64(targetUsed) - actualUsage
		sub.SetTrafficUsedAdjustment(adjustment)

		uc.logger.Infow("traffic used adjustment computed",
			"subscription_id", cmd.SubscriptionID,
			"target_used", targetUsed,
			"raw_actual", actualUsage,
			"adjustment", adjustment,
		)
	}

	// Persist changes
	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		uc.logger.Errorw("failed to update subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	// Invalidate quota cache
	if uc.quotaCacheManager != nil {
		if err := uc.quotaCacheManager.InvalidateQuota(ctx, sub.ID()); err != nil {
			uc.logger.Warnw("failed to invalidate quota cache",
				"subscription_id", cmd.SubscriptionID,
				"error", err,
			)
		}
	}

	// Notify node agents of subscription update
	if uc.subscriptionNotifier != nil {
		notifyCtx := context.Background()
		if err := uc.subscriptionNotifier.NotifySubscriptionUpdate(notifyCtx, sub); err != nil {
			uc.logger.Warnw("failed to notify nodes of subscription update",
				"subscription_id", cmd.SubscriptionID,
				"error", err,
			)
		}
	}

	uc.logger.Infow("subscription updated successfully", "subscription_id", cmd.SubscriptionID)
	return nil
}
