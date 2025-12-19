package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ResetSubscriptionLinkCommand struct {
	SubscriptionID uint
}

type ResetSubscriptionLinkUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	logger           logger.Interface
	baseURL          string
}

func NewResetSubscriptionLinkUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
	baseURL string,
) *ResetSubscriptionLinkUseCase {
	return &ResetSubscriptionLinkUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
		baseURL:          baseURL,
	}
}

func (uc *ResetSubscriptionLinkUseCase) Execute(ctx context.Context, cmd ResetSubscriptionLinkCommand) (*dto.SubscriptionDTO, error) {
	sub, err := uc.subscriptionRepo.GetByID(ctx, cmd.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Reset link token to generate new subscription link (invalidates old link)
	if err := sub.ResetLinkToken(); err != nil {
		uc.logger.Errorw("failed to reset link token", "error", err, "subscription_id", cmd.SubscriptionID)
		return nil, fmt.Errorf("failed to reset link token: %w", err)
	}

	// Persist the change
	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		uc.logger.Errorw("failed to update subscription", "error", err, "subscription_id", cmd.SubscriptionID)
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Get plan for DTO
	plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Warnw("failed to get subscription plan", "error", err, "plan_id", sub.PlanID())
		// Continue without plan info
	}

	result := dto.ToSubscriptionDTO(sub, plan, uc.baseURL)

	uc.logger.Infow("subscription link reset successfully",
		"subscription_id", cmd.SubscriptionID,
	)

	return result, nil
}
