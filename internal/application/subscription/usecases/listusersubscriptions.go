package usecases

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

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
	IncludeCounts bool  // When true, also return subscription status counts
}

type ListUserSubscriptionsResult struct {
	Subscriptions []*dto.SubscriptionDTO      `json:"subscriptions"`
	Total         int64                       `json:"total"`
	Page          int                         `json:"page"`
	PageSize      int                         `json:"page_size"`
	StatusCounts  *dto.SubscriptionStatusCounts `json:"status_counts,omitempty"` // Present when IncludeCounts is true
}

type ListUserSubscriptionsUseCase struct {
	subscriptionRepo    subscription.SubscriptionRepository
	planRepo            subscription.PlanRepository
	userRepo            user.Repository
	onlineDeviceCounter OnlineDeviceCounter // optional, nil-safe
	quotaService        QuotaService        // optional, nil-safe
	logger              logger.Interface
	baseURL             string
}

// SetQuotaService sets the quota service for data usage reporting (optional).
func (uc *ListUserSubscriptionsUseCase) SetQuotaService(qs QuotaService) {
	uc.quotaService = qs
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
		// Set data usage from QuotaService
		if uc.quotaService != nil && plan != nil {
			quota, err := uc.quotaService.GetSubscriptionQuotaPreloaded(ctx, sub, plan)
			if err != nil {
				uc.logger.Warnw("failed to get subscription quota", "error", err, "subscription_id", sub.ID())
			} else if quota != nil {
				opts = append(opts, dto.WithDataUsage(quota.UsedBytes, quota.LimitBytes))
			}
		}

		result := dto.ToSubscriptionDTO(sub, plan, subscriptionUser, uc.baseURL, opts...)
		dtos = append(dtos, result)
	}

	// Query status counts if requested
	var statusCounts *dto.SubscriptionStatusCounts
	if query.IncludeCounts {
		statusCounts = &dto.SubscriptionStatusCounts{}
		g, gctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			count, err := uc.subscriptionRepo.CountByStatus(gctx, "active")
			if err != nil {
				return fmt.Errorf("count active: %w", err)
			}
			statusCounts.Active = count
			return nil
		})
		g.Go(func() error {
			count, err := uc.subscriptionRepo.CountByStatus(gctx, "expired")
			if err != nil {
				return fmt.Errorf("count expired: %w", err)
			}
			statusCounts.Expired = count
			return nil
		})
		g.Go(func() error {
			count, err := uc.subscriptionRepo.CountByStatus(gctx, "suspended")
			if err != nil {
				return fmt.Errorf("count suspended: %w", err)
			}
			statusCounts.Suspended = count
			return nil
		})
		g.Go(func() error {
			count, err := uc.subscriptionRepo.CountByStatus(gctx, "pending_payment")
			if err != nil {
				return fmt.Errorf("count pending_payment: %w", err)
			}
			statusCounts.PendingPayment = count
			return nil
		})
		g.Go(func() error {
			subs, err := uc.subscriptionRepo.FindExpiringSubscriptions(gctx, 7)
			if err != nil {
				return fmt.Errorf("find expiring subscriptions: %w", err)
			}
			statusCounts.ExpiringIn7Days = int64(len(subs))
			return nil
		})

		if err := g.Wait(); err != nil {
			uc.logger.Warnw("failed to get subscription status counts", "error", err)
			// Non-fatal: return list without counts
			statusCounts = nil
		}
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
		StatusCounts:  statusCounts,
	}, nil
}
