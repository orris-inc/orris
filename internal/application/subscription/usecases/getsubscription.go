package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type GetSubscriptionQuery struct {
	SubscriptionID uint
}

type GetSubscriptionUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	userRepo         user.Repository
	logger           logger.Interface
	baseURL          string
}

func NewGetSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	userRepo user.Repository,
	logger logger.Interface,
	baseURL string,
) *GetSubscriptionUseCase {
	return &GetSubscriptionUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		userRepo:         userRepo,
		logger:           logger,
		baseURL:          baseURL,
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

	// Fetch user information for embedding in response
	var subscriptionUser *user.User
	if sub.UserID() > 0 {
		subscriptionUser, err = uc.userRepo.GetByID(ctx, sub.UserID())
		if err != nil {
			// Log warning but don't fail - user info is optional
			uc.logger.Warnw("failed to get subscription user", "error", err, "user_id", sub.UserID())
		}
	}

	result := dto.ToSubscriptionDTO(sub, plan, subscriptionUser, uc.baseURL)

	uc.logger.Debugw("subscription retrieved successfully",
		"subscription_id", query.SubscriptionID,
		"user_id", sub.UserID(),
		"status", sub.Status(),
	)

	return result, nil
}

// ExecuteBySID retrieves a subscription by its Stripe-style SID
func (uc *GetSubscriptionUseCase) ExecuteBySID(ctx context.Context, sid string) (*dto.SubscriptionDTO, error) {
	sub, err := uc.subscriptionRepo.GetBySID(ctx, sid)
	if err != nil {
		uc.logger.Errorw("failed to get subscription by SID", "error", err, "subscription_sid", sid)
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan", "error", err, "plan_id", sub.PlanID())
		return nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}

	// Fetch user information for embedding in response
	var subscriptionUser *user.User
	if sub.UserID() > 0 {
		subscriptionUser, err = uc.userRepo.GetByID(ctx, sub.UserID())
		if err != nil {
			// Log warning but don't fail - user info is optional
			uc.logger.Warnw("failed to get subscription user", "error", err, "user_id", sub.UserID())
		}
	}

	result := dto.ToSubscriptionDTO(sub, plan, subscriptionUser, uc.baseURL)

	uc.logger.Debugw("subscription retrieved successfully by SID",
		"subscription_sid", sid,
		"subscription_id", sub.ID(),
		"user_id", sub.UserID(),
		"status", sub.Status(),
	)

	return result, nil
}
