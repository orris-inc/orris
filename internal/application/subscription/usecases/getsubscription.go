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
	subscriptionRepo     subscription.SubscriptionRepository
	planRepo             subscription.PlanRepository
	userRepo             user.Repository
	onlineDeviceCounter  OnlineDeviceCounter // optional, nil-safe
	logger               logger.Interface
	baseURL              string
}

func NewGetSubscriptionUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	userRepo user.Repository,
	logger logger.Interface,
	baseURL string,
	onlineDeviceCounter OnlineDeviceCounter,
) *GetSubscriptionUseCase {
	return &GetSubscriptionUseCase{
		subscriptionRepo:    subscriptionRepo,
		planRepo:            planRepo,
		userRepo:            userRepo,
		onlineDeviceCounter: onlineDeviceCounter,
		logger:              logger,
		baseURL:             baseURL,
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

	opts := uc.buildDTOOptions(ctx, sub.ID(), plan)
	result := dto.ToSubscriptionDTO(sub, plan, subscriptionUser, uc.baseURL, opts...)

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

	opts := uc.buildDTOOptions(ctx, sub.ID(), plan)
	result := dto.ToSubscriptionDTO(sub, plan, subscriptionUser, uc.baseURL, opts...)

	uc.logger.Debugw("subscription retrieved successfully by SID",
		"subscription_sid", sid,
		"subscription_id", sub.ID(),
		"user_id", sub.UserID(),
		"status", sub.Status(),
	)

	return result, nil
}

// buildDTOOptions builds DTO options for online device count and device limit.
func (uc *GetSubscriptionUseCase) buildDTOOptions(ctx context.Context, subID uint, plan *subscription.Plan) []dto.SubscriptionDTOOption {
	var opts []dto.SubscriptionDTOOption

	// Extract device limit from plan features
	if plan != nil && plan.Features() != nil {
		if deviceLimit, err := plan.Features().GetDeviceLimit(); err == nil {
			opts = append(opts, dto.WithDeviceLimit(deviceLimit))
		}
	}

	// Query online device count
	if uc.onlineDeviceCounter != nil {
		count, err := uc.onlineDeviceCounter.GetOnlineDeviceCount(ctx, subID)
		if err != nil {
			uc.logger.Warnw("failed to get online device count", "error", err, "subscription_id", subID)
		} else {
			opts = append(opts, dto.WithOnlineDeviceCount(count))
		}
	}

	return opts
}
