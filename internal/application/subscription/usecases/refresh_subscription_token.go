package usecases

import (
	"context"
	"fmt"
	"time"

	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"
)

type RefreshSubscriptionTokenCommand struct {
	OldTokenID uint
}

type RefreshSubscriptionTokenResult struct {
	NewToken  string
	TokenID   uint
	ExpiresAt *time.Time
}

type RefreshSubscriptionTokenUseCase struct {
	tokenRepo        subscription.SubscriptionTokenRepository
	subscriptionRepo subscription.SubscriptionRepository
	tokenGenerator   TokenGenerator
	logger           logger.Interface
}

func NewRefreshSubscriptionTokenUseCase(
	tokenRepo subscription.SubscriptionTokenRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	tokenGenerator TokenGenerator,
	logger logger.Interface,
) *RefreshSubscriptionTokenUseCase {
	return &RefreshSubscriptionTokenUseCase{
		tokenRepo:        tokenRepo,
		subscriptionRepo: subscriptionRepo,
		tokenGenerator:   tokenGenerator,
		logger:           logger,
	}
}

func (uc *RefreshSubscriptionTokenUseCase) Execute(ctx context.Context, cmd RefreshSubscriptionTokenCommand) (*RefreshSubscriptionTokenResult, error) {
	oldToken, err := uc.tokenRepo.GetByID(ctx, cmd.OldTokenID)
	if err != nil {
		uc.logger.Errorw("failed to get old token", "error", err, "token_id", cmd.OldTokenID)
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	if !oldToken.IsValid() {
		return nil, fmt.Errorf("old token is invalid or expired")
	}

	sub, err := uc.subscriptionRepo.GetByID(ctx, oldToken.SubscriptionID())
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", oldToken.SubscriptionID())
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if !sub.IsActive() {
		return nil, fmt.Errorf("subscription is not active")
	}

	plainToken, hashedToken, err := uc.tokenGenerator.Generate("sk")
	if err != nil {
		uc.logger.Errorw("failed to generate new token", "error", err)
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	prefix := plainToken[:8]

	newToken, err := subscription.NewSubscriptionToken(
		oldToken.SubscriptionID(),
		oldToken.Name(),
		hashedToken,
		prefix,
		oldToken.Scope(),
		oldToken.ExpiresAt(),
	)
	if err != nil {
		uc.logger.Errorw("failed to create new token entity", "error", err)
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	if err := uc.tokenRepo.Create(ctx, newToken); err != nil {
		uc.logger.Errorw("failed to persist new token", "error", err)
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	if err := oldToken.Revoke(); err != nil {
		uc.logger.Warnw("failed to revoke old token", "error", err, "old_token_id", cmd.OldTokenID)
	} else {
		if err := uc.tokenRepo.Update(ctx, oldToken); err != nil {
			uc.logger.Warnw("failed to update old token status", "error", err, "old_token_id", cmd.OldTokenID)
		}
	}

	uc.logger.Infow("token refreshed successfully",
		"old_token_id", cmd.OldTokenID,
		"new_token_id", newToken.ID(),
		"subscription_id", oldToken.SubscriptionID(),
	)

	return &RefreshSubscriptionTokenResult{
		NewToken:  plainToken,
		TokenID:   newToken.ID(),
		ExpiresAt: newToken.ExpiresAt(),
	}, nil
}
