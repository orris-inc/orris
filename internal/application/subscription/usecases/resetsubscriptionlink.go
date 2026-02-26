package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ResetSubscriptionLinkCommand struct {
	SubscriptionID uint
}

type ResetSubscriptionLinkUseCase struct {
	subscriptionRepo    subscription.SubscriptionRepository
	planRepo            subscription.PlanRepository
	userRepo            user.Repository
	onlineDeviceCounter OnlineDeviceCounter // optional, nil-safe
	logger              logger.Interface
	baseURL             string
}

func NewResetSubscriptionLinkUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	userRepo user.Repository,
	logger logger.Interface,
	baseURL string,
	onlineDeviceCounter OnlineDeviceCounter,
) *ResetSubscriptionLinkUseCase {
	return &ResetSubscriptionLinkUseCase{
		subscriptionRepo:    subscriptionRepo,
		planRepo:            planRepo,
		userRepo:            userRepo,
		onlineDeviceCounter: onlineDeviceCounter,
		logger:              logger,
		baseURL:             baseURL,
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
		return nil, err
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

	// Fetch user information for embedding in response
	var subscriptionUser *user.User
	if sub.UserID() > 0 {
		subscriptionUser, err = uc.userRepo.GetByID(ctx, sub.UserID())
		if err != nil {
			// Log warning but don't fail - user info is optional
			uc.logger.Warnw("failed to get subscription user", "error", err, "user_id", sub.UserID())
		}
	}

	// Build DTO options for online device count and device limit
	var opts []dto.SubscriptionDTOOption
	if plan != nil && plan.Features() != nil {
		if deviceLimit, dlErr := plan.Features().GetDeviceLimit(); dlErr == nil {
			opts = append(opts, dto.WithDeviceLimit(deviceLimit))
		}
	}
	if uc.onlineDeviceCounter != nil {
		count, dcErr := uc.onlineDeviceCounter.GetOnlineDeviceCount(ctx, sub.ID())
		if dcErr != nil {
			uc.logger.Warnw("failed to get online device count", "error", dcErr, "subscription_id", sub.ID())
		} else {
			opts = append(opts, dto.WithOnlineDeviceCount(count))
		}
	}

	result := dto.ToSubscriptionDTO(sub, plan, subscriptionUser, uc.baseURL, opts...)

	uc.logger.Infow("subscription link reset successfully",
		"subscription_id", cmd.SubscriptionID,
	)

	return result, nil
}
