package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ListUserSubscriptionsQuery struct {
	UserID   *uint // nil means all users (admin only)
	Status   *string
	Page     int
	PageSize int
}

type ListUserSubscriptionsResult struct {
	Subscriptions []*dto.SubscriptionDTO `json:"subscriptions"`
	Total         int64                  `json:"total"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"page_size"`
}

type ListUserSubscriptionsUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	userRepo         user.Repository
	logger           logger.Interface
	baseURL          string
}

func NewListUserSubscriptionsUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	userRepo user.Repository,
	logger logger.Interface,
	baseURL string,
) *ListUserSubscriptionsUseCase {
	return &ListUserSubscriptionsUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		userRepo:         userRepo,
		logger:           logger,
		baseURL:          baseURL,
	}
}

func (uc *ListUserSubscriptionsUseCase) Execute(ctx context.Context, query ListUserSubscriptionsQuery) (*ListUserSubscriptionsResult, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	filter := subscription.SubscriptionFilter{
		UserID:   query.UserID,
		Status:   query.Status,
		Page:     query.Page,
		PageSize: query.PageSize,
		SortBy:   "created_at",
		SortDesc: true,
	}

	subscriptions, total, err := uc.subscriptionRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list subscriptions", "error", err, "user_id", query.UserID)
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	// Collect unique plan IDs
	planIDs := make(map[uint]bool)
	for _, sub := range subscriptions {
		planIDs[sub.PlanID()] = true
	}

	// Fetch plans
	plans := make(map[uint]*subscription.Plan)
	for planID := range planIDs {
		plan, err := uc.planRepo.GetByID(ctx, planID)
		if err != nil {
			uc.logger.Warnw("failed to get plan", "error", err, "plan_id", planID)
			continue
		}
		plans[planID] = plan
	}

	// Collect unique user IDs
	userIDs := make(map[uint]bool)
	for _, sub := range subscriptions {
		if sub.UserID() > 0 {
			userIDs[sub.UserID()] = true
		}
	}

	// Fetch users
	users := make(map[uint]*user.User)
	for userID := range userIDs {
		u, err := uc.userRepo.GetByID(ctx, userID)
		if err != nil {
			uc.logger.Warnw("failed to get user", "error", err, "user_id", userID)
			continue
		}
		users[userID] = u
	}

	// Build DTOs with embedded user and plan info
	dtos := make([]*dto.SubscriptionDTO, 0, len(subscriptions))
	for _, sub := range subscriptions {
		plan := plans[sub.PlanID()]
		subscriptionUser := users[sub.UserID()]
		result := dto.ToSubscriptionDTO(sub, plan, subscriptionUser, uc.baseURL)
		dtos = append(dtos, result)
	}

	uc.logger.Debugw("subscriptions listed successfully",
		"user_id", query.UserID,
		"total", total,
		"page", query.Page,
		"page_size", query.PageSize,
	)

	return &ListUserSubscriptionsResult{
		Subscriptions: dtos,
		Total:         total,
		Page:          query.Page,
		PageSize:      query.PageSize,
	}, nil
}
