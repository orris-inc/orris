package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SuspendSubscriptionCommand represents the command to suspend a subscription
type SuspendSubscriptionCommand struct {
	SubscriptionID uint
	Reason         string
}

// SuspendSubscriptionUseCase handles suspending subscriptions
type SuspendSubscriptionUseCase struct {
	subscriptionRepo     subscription.SubscriptionRepository
	subscriptionNotifier SubscriptionChangeNotifier
	logger               logger.Interface
}

// NewSuspendSubscriptionUseCase creates a new instance of SuspendSubscriptionUseCase
func NewSuspendSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	logger logger.Interface,
) *SuspendSubscriptionUseCase {
	return &SuspendSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

// SetSubscriptionNotifier sets the subscription change notifier (optional).
func (uc *SuspendSubscriptionUseCase) SetSubscriptionNotifier(notifier SubscriptionChangeNotifier) {
	uc.subscriptionNotifier = notifier
}

// Execute suspends a subscription
func (uc *SuspendSubscriptionUseCase) Execute(ctx context.Context, cmd SuspendSubscriptionCommand) error {
	sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return fmt.Errorf("subscription not found")
	}

	if err := sub.Suspend(cmd.Reason); err != nil {
		uc.logger.Errorw("failed to suspend subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to suspend subscription: %w", err)
	}

	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		uc.logger.Errorw("failed to update subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	uc.logger.Infow("subscription suspended successfully",
		"subscription_id", cmd.SubscriptionID,
		"reason", cmd.Reason,
		"status", sub.Status(),
	)

	// Notify node agents when subscription is suspended
	if uc.subscriptionNotifier != nil {
		notifyCtx := context.Background()
		if err := uc.subscriptionNotifier.NotifySubscriptionDeactivation(notifyCtx, sub); err != nil {
			uc.logger.Warnw("failed to notify nodes of subscription suspension",
				"subscription_id", cmd.SubscriptionID,
				"error", err,
			)
		}
	}

	return nil
}

// UnsuspendSubscriptionCommand represents the command to unsuspend a subscription
type UnsuspendSubscriptionCommand struct {
	SubscriptionID uint
}

// UnsuspendSubscriptionUseCase handles unsuspending (reactivating) subscriptions
type UnsuspendSubscriptionUseCase struct {
	subscriptionRepo     subscription.SubscriptionRepository
	subscriptionNotifier SubscriptionChangeNotifier
	quotaCacheManager    QuotaCacheManager
	logger               logger.Interface
}

// NewUnsuspendSubscriptionUseCase creates a new instance of UnsuspendSubscriptionUseCase
func NewUnsuspendSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	logger logger.Interface,
) *UnsuspendSubscriptionUseCase {
	return &UnsuspendSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

// SetSubscriptionNotifier sets the subscription change notifier (optional).
func (uc *UnsuspendSubscriptionUseCase) SetSubscriptionNotifier(notifier SubscriptionChangeNotifier) {
	uc.subscriptionNotifier = notifier
}

// SetQuotaCacheManager sets the quota cache manager (optional).
func (uc *UnsuspendSubscriptionUseCase) SetQuotaCacheManager(manager QuotaCacheManager) {
	uc.quotaCacheManager = manager
}

// Execute unsuspends a subscription
func (uc *UnsuspendSubscriptionUseCase) Execute(ctx context.Context, cmd UnsuspendSubscriptionCommand) error {
	sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return fmt.Errorf("subscription not found")
	}

	if err := sub.Unsuspend(); err != nil {
		uc.logger.Errorw("failed to unsuspend subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to unsuspend subscription: %w", err)
	}

	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		uc.logger.Errorw("failed to update subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	uc.logger.Infow("subscription unsuspended successfully",
		"subscription_id", cmd.SubscriptionID,
		"status", sub.Status(),
	)

	// Update quota cache to reflect unsuspended state (only update suspended flag)
	if uc.quotaCacheManager != nil {
		if err := uc.quotaCacheManager.SetSuspended(ctx, cmd.SubscriptionID, false); err != nil {
			uc.logger.Warnw("failed to update quota cache suspended status",
				"subscription_id", cmd.SubscriptionID,
				"error", err,
			)
		}
	}

	// Notify node agents when subscription is reactivated
	if uc.subscriptionNotifier != nil {
		notifyCtx := context.Background()
		if err := uc.subscriptionNotifier.NotifySubscriptionActivation(notifyCtx, sub); err != nil {
			uc.logger.Warnw("failed to notify nodes of subscription reactivation",
				"subscription_id", cmd.SubscriptionID,
				"error", err,
			)
		}
	}

	return nil
}
