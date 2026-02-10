package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	apperrors "github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type CancelSubscriptionCommand struct {
	SubscriptionID uint
	Reason         string
	Immediate      bool
}

type CancelSubscriptionUseCase struct {
	subscriptionRepo     subscription.SubscriptionRepository
	tokenRepo            subscription.SubscriptionTokenRepository
	subscriptionNotifier SubscriptionChangeNotifier // Optional: for notifying node agents
	logger               logger.Interface
}

func NewCancelSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	tokenRepo subscription.SubscriptionTokenRepository,
	logger logger.Interface,
) *CancelSubscriptionUseCase {
	return &CancelSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		tokenRepo:        tokenRepo,
		logger:           logger,
	}
}

// SetSubscriptionNotifier sets the subscription change notifier (optional).
func (uc *CancelSubscriptionUseCase) SetSubscriptionNotifier(notifier SubscriptionChangeNotifier) {
	uc.subscriptionNotifier = notifier
}

func (uc *CancelSubscriptionUseCase) Execute(ctx context.Context, cmd CancelSubscriptionCommand) error {
	sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return fmt.Errorf("subscription not found")
	}

	if err := sub.Cancel(cmd.Reason); err != nil {
		uc.logger.Errorw("failed to cancel subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return err
	}

	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		uc.logger.Errorw("failed to update subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	uc.logger.Infow("subscription cancelled successfully",
		"subscription_id", cmd.SubscriptionID,
		"reason", cmd.Reason,
		"immediate", cmd.Immediate,
		"status", sub.Status(),
	)

	// Revoke all tokens after persisting the cancellation to ensure the subscription
	// is cancelled regardless of token revocation outcome.
	if cmd.Immediate {
		if err := uc.revokeAllTokens(ctx, cmd.SubscriptionID); err != nil {
			uc.logger.Errorw("failed to revoke tokens after immediate cancellation",
				"error", err, "subscription_id", cmd.SubscriptionID)
			return apperrors.NewInternalError("subscription cancelled but failed to revoke all tokens", err.Error())
		}
	}

	// Notify node agents when subscription is no longer active
	if !sub.Status().CanUseService() && uc.subscriptionNotifier != nil {
		notifyCtx := context.Background()
		if err := uc.subscriptionNotifier.NotifySubscriptionDeactivation(notifyCtx, sub); err != nil {
			// Log error but don't fail the cancellation
			uc.logger.Warnw("failed to notify nodes of subscription cancellation",
				"subscription_id", cmd.SubscriptionID,
				"error", err,
			)
		}
	}

	return nil
}

func (uc *CancelSubscriptionUseCase) revokeAllTokens(ctx context.Context, subscriptionID uint) error {
	tokens, err := uc.tokenRepo.GetBySubscriptionID(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get tokens: %w", err)
	}

	failedCount := 0
	for _, token := range tokens {
		if err := token.Revoke(); err != nil {
			uc.logger.Errorw("failed to revoke token", "error", err, "token_id", token.ID())
			failedCount++
			continue
		}

		if err := uc.tokenRepo.Update(ctx, token); err != nil {
			uc.logger.Errorw("failed to update revoked token", "error", err, "token_id", token.ID())
			failedCount++
		}
	}

	if failedCount > 0 {
		return fmt.Errorf("failed to revoke %d of %d tokens", failedCount, len(tokens))
	}

	return nil
}
