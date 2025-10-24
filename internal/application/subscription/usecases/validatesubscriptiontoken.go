package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"
)

type ValidateSubscriptionTokenCommand struct {
	PlainToken    string
	RequiredScope string
	IPAddress     string
}

type ValidateSubscriptionTokenResult struct {
	Token        *subscription.SubscriptionToken
	Subscription *subscription.Subscription
	Plan         *subscription.SubscriptionPlan
}

type ValidateSubscriptionTokenUseCase struct {
	tokenRepo        subscription.SubscriptionTokenRepository
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.SubscriptionPlanRepository
	tokenGenerator   TokenGenerator
	logger           logger.Interface
}

func NewValidateSubscriptionTokenUseCase(
	tokenRepo subscription.SubscriptionTokenRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.SubscriptionPlanRepository,
	tokenGenerator TokenGenerator,
	logger logger.Interface,
) *ValidateSubscriptionTokenUseCase {
	return &ValidateSubscriptionTokenUseCase{
		tokenRepo:        tokenRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		tokenGenerator:   tokenGenerator,
		logger:           logger,
	}
}

func (uc *ValidateSubscriptionTokenUseCase) Execute(ctx context.Context, cmd ValidateSubscriptionTokenCommand) (*ValidateSubscriptionTokenResult, error) {
	_, tokenHash, err := uc.tokenGenerator.Generate("")
	if err != nil {
		return nil, fmt.Errorf("failed to hash token: %w", err)
	}

	token, err := uc.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		uc.logger.Warnw("token not found", "error", err)
		return nil, fmt.Errorf("invalid token")
	}

	if !token.IsValid() {
		uc.logger.Warnw("token is invalid or expired", "token_id", token.ID())
		return nil, fmt.Errorf("token is invalid or expired")
	}

	if cmd.RequiredScope != "" && !token.HasScope(cmd.RequiredScope) {
		uc.logger.Warnw("token lacks required scope",
			"token_id", token.ID(),
			"required_scope", cmd.RequiredScope,
			"token_scope", token.Scope().String(),
		)
		return nil, fmt.Errorf("token lacks required scope: %s", cmd.RequiredScope)
	}

	sub, err := uc.subscriptionRepo.GetByID(ctx, token.SubscriptionID())
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", token.SubscriptionID())
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if !sub.IsActive() {
		uc.logger.Warnw("subscription is not active",
			"subscription_id", sub.ID(),
			"status", sub.Status(),
		)
		return nil, fmt.Errorf("subscription is not active")
	}

	plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan", "error", err, "plan_id", sub.PlanID())
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	if cmd.IPAddress != "" {
		token.RecordUsage(cmd.IPAddress)
		if err := uc.tokenRepo.Update(ctx, token); err != nil {
			uc.logger.Warnw("failed to update token usage", "error", err, "token_id", token.ID())
		}
	}

	uc.logger.Infow("token validated successfully",
		"token_id", token.ID(),
		"subscription_id", sub.ID(),
		"usage_count", token.UsageCount(),
	)

	return &ValidateSubscriptionTokenResult{
		Token:        token,
		Subscription: sub,
		Plan:         plan,
	}, nil
}
