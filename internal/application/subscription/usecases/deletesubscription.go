package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type DeleteSubscriptionUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	tokenRepo        subscription.SubscriptionTokenRepository
	txMgr            *db.TransactionManager
	logger           logger.Interface
}

func NewDeleteSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	tokenRepo subscription.SubscriptionTokenRepository,
	txMgr *db.TransactionManager,
	logger logger.Interface,
) *DeleteSubscriptionUseCase {
	return &DeleteSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		tokenRepo:        tokenRepo,
		txMgr:            txMgr,
		logger:           logger,
	}
}

func (uc *DeleteSubscriptionUseCase) Execute(ctx context.Context, subscriptionID uint) error {
	// Get the subscription first to verify it exists
	sub, err := uc.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", subscriptionID)
		return fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return errors.NewNotFoundError("subscription not found")
	}

	// Only allow deleting cancelled or expired subscriptions
	status := sub.Status()
	if status != valueobjects.StatusCancelled && status != valueobjects.StatusExpired {
		return errors.NewValidationError("can only delete cancelled or expired subscriptions")
	}

	// Delete tokens and subscription in a transaction
	err = uc.txMgr.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Delete all tokens associated with this subscription
		if err := uc.tokenRepo.RevokeAllBySubscriptionID(txCtx, subscriptionID); err != nil {
			uc.logger.Errorw("failed to revoke subscription tokens", "error", err, "subscription_id", subscriptionID)
			return fmt.Errorf("failed to revoke subscription tokens: %w", err)
		}

		// Delete the subscription
		if err := uc.subscriptionRepo.Delete(txCtx, subscriptionID); err != nil {
			uc.logger.Errorw("failed to delete subscription", "error", err, "subscription_id", subscriptionID)
			return fmt.Errorf("failed to delete subscription: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	uc.logger.Infow("subscription deleted successfully", "subscription_id", subscriptionID)
	return nil
}
