package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/entitlement"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type CancelSubscriptionCommand struct {
	SubscriptionID uint
	Reason         string
	Immediate      bool
}

type CancelSubscriptionUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	tokenRepo        subscription.SubscriptionTokenRepository
	entitlementRepo  entitlement.Repository
	logger           logger.Interface
}

func NewCancelSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	tokenRepo subscription.SubscriptionTokenRepository,
	entitlementRepo entitlement.Repository,
	logger logger.Interface,
) *CancelSubscriptionUseCase {
	return &CancelSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		tokenRepo:        tokenRepo,
		entitlementRepo:  entitlementRepo,
		logger:           logger,
	}
}

func (uc *CancelSubscriptionUseCase) Execute(ctx context.Context, cmd CancelSubscriptionCommand) error {
	sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if err := sub.Cancel(cmd.Reason); err != nil {
		uc.logger.Errorw("failed to cancel subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	if cmd.Immediate {
		// Revoke all tokens
		if err := uc.revokeAllTokens(ctx, cmd.SubscriptionID); err != nil {
			uc.logger.Warnw("failed to revoke all tokens", "error", err, "subscription_id", cmd.SubscriptionID)
		}

		// Revoke all entitlements granted by this subscription
		if err := uc.revokeSubscriptionEntitlements(ctx, cmd.SubscriptionID); err != nil {
			uc.logger.Warnw("failed to revoke subscription entitlements", "error", err, "subscription_id", cmd.SubscriptionID)
		}
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

	return nil
}

func (uc *CancelSubscriptionUseCase) revokeAllTokens(ctx context.Context, subscriptionID uint) error {
	tokens, err := uc.tokenRepo.GetBySubscriptionID(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get tokens: %w", err)
	}

	for _, token := range tokens {
		if err := token.Revoke(); err != nil {
			uc.logger.Warnw("failed to revoke token", "error", err, "token_id", token.ID())
			continue
		}

		if err := uc.tokenRepo.Update(ctx, token); err != nil {
			uc.logger.Warnw("failed to update revoked token", "error", err, "token_id", token.ID())
		}
	}

	return nil
}

// revokeSubscriptionEntitlements revokes all entitlements granted by a subscription
func (uc *CancelSubscriptionUseCase) revokeSubscriptionEntitlements(ctx context.Context, subscriptionID uint) error {
	// Revoke all entitlements with this subscription as source
	if err := uc.entitlementRepo.RevokeBySource(ctx, entitlement.SourceTypeSubscription, subscriptionID); err != nil {
		uc.logger.Errorw("failed to revoke entitlements by source",
			"error", err,
			"source_type", entitlement.SourceTypeSubscription,
			"source_id", subscriptionID,
		)
		return fmt.Errorf("failed to revoke subscription entitlements: %w", err)
	}

	uc.logger.Infow("revoked all subscription entitlements",
		"subscription_id", subscriptionID,
	)
	return nil
}
