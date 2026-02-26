package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils/setutil"
)

type ListUserSubscriptionsQuery struct {
	UserID        *uint // nil means all users (admin only)
	PlanID        *uint
	Status        *string
	BillingCycle  *string
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
	ExpiresBefore *time.Time
	Page          int
	PageSize      int
	SortBy        string
	SortDesc      *bool // nil means default (true = DESC)
}

type ListUserSubscriptionsResult struct {
	Subscriptions []*dto.SubscriptionDTO `json:"subscriptions"`
	Total         int64                  `json:"total"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"page_size"`
}

type ListUserSubscriptionsUseCase struct {
	subscriptionRepo    subscription.SubscriptionRepository
	planRepo            subscription.PlanRepository
	userRepo            user.Repository
	onlineDeviceCounter OnlineDeviceCounter // optional, nil-safe
	logger              logger.Interface
	baseURL             string
}

func NewListUserSubscriptionsUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	userRepo user.Repository,
	logger logger.Interface,
	baseURL string,
	onlineDeviceCounter OnlineDeviceCounter,
) *ListUserSubscriptionsUseCase {
	return &ListUserSubscriptionsUseCase{
		subscriptionRepo:    subscriptionRepo,
		planRepo:            planRepo,
		userRepo:            userRepo,
		onlineDeviceCounter: onlineDeviceCounter,
		logger:              logger,
		baseURL:             baseURL,
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

	// Default sort settings
	sortBy := "created_at"
	if query.SortBy != "" {
		sortBy = query.SortBy
	}
	sortDesc := true
	if query.SortDesc != nil {
		sortDesc = *query.SortDesc
	}

	filter := subscription.SubscriptionFilter{
		UserID:        query.UserID,
		PlanID:        query.PlanID,
		Status:        query.Status,
		BillingCycle:  query.BillingCycle,
		CreatedFrom:   query.CreatedFrom,
		CreatedTo:     query.CreatedTo,
		ExpiresBefore: query.ExpiresBefore,
		Page:          query.Page,
		PageSize:      query.PageSize,
		SortBy:        sortBy,
		SortDesc:      sortDesc,
	}

	subscriptions, total, err := uc.subscriptionRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list subscriptions", "error", err, "user_id", query.UserID)
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	// Collect unique plan IDs and user IDs
	planIDSet := setutil.NewUintSet()
	userIDSet := setutil.NewUintSet()
	for _, sub := range subscriptions {
		planIDSet.Add(sub.PlanID())
		if sub.UserID() > 0 {
			userIDSet.Add(sub.UserID())
		}
	}

	// Batch fetch plans
	plans := make(map[uint]*subscription.Plan)
	if planIDs := planIDSet.ToSlice(); len(planIDs) > 0 {
		planList, err := uc.planRepo.GetByIDs(ctx, planIDs)
		if err != nil {
			uc.logger.Warnw("failed to batch get plans", "error", err)
		} else {
			for _, plan := range planList {
				plans[plan.ID()] = plan
			}
		}
	}

	// Batch fetch users
	users := make(map[uint]*user.User)
	if userIDs := userIDSet.ToSlice(); len(userIDs) > 0 {
		userList, err := uc.userRepo.GetByIDs(ctx, userIDs)
		if err != nil {
			uc.logger.Warnw("failed to batch get users", "error", err)
		} else {
			for _, u := range userList {
				users[u.ID()] = u
			}
		}
	}

	// Batch query online device counts
	subIDs := make([]uint, 0, len(subscriptions))
	for _, sub := range subscriptions {
		subIDs = append(subIDs, sub.ID())
	}
	onlineCounts := make(map[uint]int)
	if uc.onlineDeviceCounter != nil && len(subIDs) > 0 {
		var err error
		onlineCounts, err = uc.onlineDeviceCounter.GetOnlineDeviceCounts(ctx, subIDs)
		if err != nil {
			uc.logger.Warnw("failed to batch get online device counts", "error", err)
			onlineCounts = make(map[uint]int)
		}
	}

	// Build DTOs with embedded user and plan info
	dtos := make([]*dto.SubscriptionDTO, 0, len(subscriptions))
	for _, sub := range subscriptions {
		plan := plans[sub.PlanID()]
		subscriptionUser := users[sub.UserID()]

		var opts []dto.SubscriptionDTOOption
		// Set device limit from plan features
		if plan != nil && plan.Features() != nil {
			if deviceLimit, err := plan.Features().GetDeviceLimit(); err == nil {
				opts = append(opts, dto.WithDeviceLimit(deviceLimit))
			}
		}
		// Set online device count
		if count, ok := onlineCounts[sub.ID()]; ok {
			opts = append(opts, dto.WithOnlineDeviceCount(count))
		}

		result := dto.ToSubscriptionDTO(sub, plan, subscriptionUser, uc.baseURL, opts...)
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
