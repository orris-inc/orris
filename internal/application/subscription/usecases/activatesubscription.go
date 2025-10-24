package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"
)

type ActivateSubscriptionCommand struct {
	SubscriptionID uint
}

type ActivateSubscriptionUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	logger           logger.Interface
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

	return nil
}
