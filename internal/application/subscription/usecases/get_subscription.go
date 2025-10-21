package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/subscription/dto"
	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"
)

type GetSubscriptionQuery struct {
	SubscriptionID uint
}

type GetSubscriptionUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.SubscriptionPlanRepository
	logger           logger.Interface
}

func NewGetSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.SubscriptionPlanRepository,
	logger logger.Interface,
) *GetSubscriptionUseCase {
	return &GetSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}

func (uc *GetSubscriptionUseCase) Execute(ctx context.Context, query GetSubscriptionQuery) (*dto.SubscriptionDTO, error) {
	sub, err := uc.subscriptionRepo.GetByID(ctx, query.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err, "subscription_id", query.SubscriptionID)
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan", "error", err, "plan_id", sub.PlanID())
		return nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}

	result := dto.ToSubscriptionDTO(sub, plan)

	uc.logger.Debugw("subscription retrieved successfully",
		"subscription_id", query.SubscriptionID,
		"user_id", sub.UserID(),
		"status", sub.Status(),
	)

	return result, nil
}
