package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ActivateSubscriptionCommand struct {
	SubscriptionID uint
}

type ActivateSubscriptionUseCase struct {
	subscriptionRepo     subscription.SubscriptionRepository
	subscriptionNotifier SubscriptionChangeNotifier // Optional: for notifying node agents
	logger               logger.Interface
}

func NewActivateSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	logger logger.Interface,
) *ActivateSubscriptionUseCase {
	return &ActivateSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

// SetSubscriptionNotifier sets the subscription change notifier (optional).
func (uc *ActivateSubscriptionUseCase) SetSubscriptionNotifier(notifier SubscriptionChangeNotifier) {
	uc.subscriptionNotifier = notifier
}

func (uc *ActivateSubscriptionUseCase) Execute(ctx context.Context, cmd ActivateSubscriptionCommand) error {
	sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if err := sub.Activate(); err != nil {
		uc.logger.Errorw("failed to activate subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to activate subscription: %w", err)
	}

	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		uc.logger.Errorw("failed to update subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	uc.logger.Infow("subscription activated successfully",
		"subscription_id", cmd.SubscriptionID,
		"status", sub.Status(),
	)

	// Notify node agents about the activated subscription
	if uc.subscriptionNotifier != nil {
		notifyCtx := context.Background()
		if err := uc.subscriptionNotifier.NotifySubscriptionActivation(notifyCtx, sub); err != nil {
			// Log error but don't fail the activation
			uc.logger.Warnw("failed to notify nodes of subscription activation",
				"subscription_id", cmd.SubscriptionID,
				"error", err,
			)
		}
	}

	return nil
}
