package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type GenerateSubscriptionTokenCommand struct {
	SubscriptionID uint
	Name           string
	Scope          string
	ExpiresAt      *time.Time
}

type GenerateSubscriptionTokenResult struct {
	Token     string     `json:"token"`
	TokenID   uint       `json:"token_id"`
	Prefix    string     `json:"prefix"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type GenerateSubscriptionTokenUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	tokenRepo        subscription.SubscriptionTokenRepository
	tokenGenerator   TokenGenerator
	logger           logger.Interface
}

func NewGenerateSubscriptionTokenUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	tokenRepo subscription.SubscriptionTokenRepository,
	tokenGenerator TokenGenerator,
	logger logger.Interface,
) *GenerateSubscriptionTokenUseCase {
	return &GenerateSubscriptionTokenUseCase{
		subscriptionRepo: subscriptionRepo,
		tokenRepo:        tokenRepo,
		tokenGenerator:   tokenGenerator,
		logger:           logger,
	}
}

func (uc *GenerateSubscriptionTokenUseCase) Execute(ctx context.Context, cmd GenerateSubscriptionTokenCommand) (*GenerateSubscriptionTokenResult, error) {
	sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if !sub.IsActive() {
		return nil, fmt.Errorf("subscription is not active")
	}

	tokenScope, err := vo.NewTokenScope(cmd.Scope)
	if err != nil {
		uc.logger.Errorw("invalid token scope", "error", err, "scope", cmd.Scope)
		return nil, fmt.Errorf("invalid token scope: %w", err)
	}

	if cmd.ExpiresAt != nil && cmd.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("expiration time cannot be in the past")
	}

	plainToken, hashedToken, err := uc.tokenGenerator.Generate("sk")
	if err != nil {
		uc.logger.Errorw("failed to generate token", "error", err)
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	prefix := plainToken[:8]

	token, err := subscription.NewSubscriptionToken(
		cmd.SubscriptionID,
		cmd.Name,
		hashedToken,
		prefix,
		*tokenScope,
		cmd.ExpiresAt,
	)
	if err != nil {
		uc.logger.Errorw("failed to create token entity", "error", err)
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	if err := uc.tokenRepo.Create(ctx, token); err != nil {
		uc.logger.Errorw("failed to persist token", "error", err)
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	uc.logger.Infow("subscription token generated successfully",
		"token_id", token.ID(),
		"subscription_id", cmd.SubscriptionID,
		"scope", cmd.Scope,
	)

	return &GenerateSubscriptionTokenResult{
		Token:     plainToken,
		TokenID:   token.ID(),
		Prefix:    prefix,
		ExpiresAt: token.ExpiresAt(),
		CreatedAt: token.CreatedAt(),
	}, nil
}
