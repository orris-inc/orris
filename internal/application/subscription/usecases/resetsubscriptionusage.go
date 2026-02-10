package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ResetSubscriptionUsageCommand represents the command to reset subscription usage
type ResetSubscriptionUsageCommand struct {
	SubscriptionID uint
}

// ResetSubscriptionUsageUseCase handles resetting subscription usage
type ResetSubscriptionUsageUseCase struct {
	subscriptionRepo     subscription.SubscriptionRepository
	subscriptionNotifier SubscriptionChangeNotifier
	quotaCacheManager    QuotaCacheManager
	logger               logger.Interface
}

// NewResetSubscriptionUsageUseCase creates a new instance of ResetSubscriptionUsageUseCase
func NewResetSubscriptionUsageUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	logger logger.Interface,
) *ResetSubscriptionUsageUseCase {
	return &ResetSubscriptionUsageUseCase{
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

// SetSubscriptionNotifier sets the subscription change notifier (optional).
func (uc *ResetSubscriptionUsageUseCase) SetSubscriptionNotifier(notifier SubscriptionChangeNotifier) {
	uc.subscriptionNotifier = notifier
}

// SetQuotaCacheManager sets the quota cache manager (optional).
func (uc *ResetSubscriptionUsageUseCase) SetQuotaCacheManager(manager QuotaCacheManager) {
	uc.quotaCacheManager = manager
}

// Execute resets a subscription's usage by updating the period start time
func (uc *ResetSubscriptionUsageUseCase) Execute(ctx context.Context, cmd ResetSubscriptionUsageCommand) error {
	sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return fmt.Errorf("subscription not found")
	}

	// Track if subscription was suspended before reset (for notification purposes)
	wasSuspended := sub.Status().String() == "suspended"

	if err := sub.ResetUsage(); err != nil {
		uc.logger.Errorw("failed to reset subscription usage", "error", err, "subscription_id", cmd.SubscriptionID)
		return err
	}

	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		uc.logger.Errorw("failed to update subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	uc.logger.Infow("subscription usage reset successfully",
		"subscription_id", cmd.SubscriptionID,
		"status", sub.Status(),
		"was_suspended", wasSuspended,
	)

	// Sync quota cache to reflect new period and unsuspended state
	if uc.quotaCacheManager != nil {
		if err := uc.quotaCacheManager.SyncQuotaFromSubscription(ctx, sub); err != nil {
			uc.logger.Warnw("failed to sync quota cache after usage reset",
				"subscription_id", cmd.SubscriptionID,
				"error", err,
			)
		}
	}

	// Notify node agents when subscription usage is reset
	// If subscription was suspended and is now active, notify activation
	if uc.subscriptionNotifier != nil && wasSuspended {
		notifyCtx := context.Background()
		if err := uc.subscriptionNotifier.NotifySubscriptionActivation(notifyCtx, sub); err != nil {
			uc.logger.Warnw("failed to notify nodes of subscription activation after usage reset",
				"subscription_id", cmd.SubscriptionID,
				"error", err,
			)
		}
	}

	return nil
}
