package usecases

import (
	"context"
	"time"

	dto "github.com/orris-inc/orris/internal/application/admin/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetUserTrafficStatsQuery represents the query parameters for user traffic statistics
type GetUserTrafficStatsQuery struct {
	From         time.Time
	To           time.Time
	ResourceType *string
	Page         int
	PageSize     int
}

// GetUserTrafficStatsUseCase handles retrieving traffic statistics grouped by user
type GetUserTrafficStatsUseCase struct {
	usageRepo        subscription.SubscriptionUsageRepository
	subscriptionRepo subscription.SubscriptionRepository
	userRepo         user.Repository
	logger           logger.Interface
}

// NewGetUserTrafficStatsUseCase creates a new GetUserTrafficStatsUseCase
func NewGetUserTrafficStatsUseCase(
	usageRepo subscription.SubscriptionUsageRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	userRepo user.Repository,
	logger logger.Interface,
) *GetUserTrafficStatsUseCase {
	return &GetUserTrafficStatsUseCase{
		usageRepo:        usageRepo,
		subscriptionRepo: subscriptionRepo,
		userRepo:         userRepo,
		logger:           logger,
	}
}

// Execute retrieves user traffic statistics
func (uc *GetUserTrafficStatsUseCase) Execute(
	ctx context.Context,
	query GetUserTrafficStatsQuery,
) (*dto.UserTrafficStatsResponse, error) {
	uc.logger.Infow("fetching user traffic stats",
		"from", query.From,
		"to", query.To,
		"resource_type", query.ResourceType,
		"page", query.Page,
		"page_size", query.PageSize,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid user traffic stats query", "error", err)
		return nil, err
	}

	page, pageSize := uc.getPaginationParams(query)

	// Get usage data grouped by subscription
	subscriptionUsages, total, err := uc.usageRepo.GetUsageGroupedBySubscription(
		ctx,
		query.ResourceType,
		query.From,
		query.To,
		page,
		pageSize,
	)
	if err != nil {
		uc.logger.Errorw("failed to fetch subscription usage", "error", err)
		return nil, errors.NewInternalError("failed to fetch subscription usage")
	}

	if len(subscriptionUsages) == 0 {
		return &dto.UserTrafficStatsResponse{
			Items:    []dto.UserTrafficStatsItem{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	// Extract subscription IDs
	subscriptionIDs := make([]uint, len(subscriptionUsages))
	for i, usage := range subscriptionUsages {
		subscriptionIDs[i] = usage.SubscriptionID
	}

	// Fetch subscriptions
	subscriptions := make(map[uint]*subscription.Subscription)
	for _, subID := range subscriptionIDs {
		sub, err := uc.subscriptionRepo.GetByID(ctx, subID)
		if err != nil {
			uc.logger.Warnw("failed to fetch subscription", "subscription_id", subID, "error", err)
			continue
		}
		subscriptions[subID] = sub
	}

	// Extract user IDs and aggregate usage by user
	userUsageMap := make(map[uint]*userUsageData)
	for _, usage := range subscriptionUsages {
		sub, ok := subscriptions[usage.SubscriptionID]
		if !ok {
			continue
		}

		userID := sub.UserID()
		if existing, exists := userUsageMap[userID]; exists {
			existing.upload += usage.Upload
			existing.download += usage.Download
			existing.total += usage.Total
			existing.subscriptionCount++
		} else {
			userUsageMap[userID] = &userUsageData{
				userID:            userID,
				upload:            usage.Upload,
				download:          usage.Download,
				total:             usage.Total,
				subscriptionCount: 1,
			}
		}
	}

	// Fetch user details
	userIDs := make([]uint, 0, len(userUsageMap))
	for userID := range userUsageMap {
		userIDs = append(userIDs, userID)
	}

	users, err := uc.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		uc.logger.Errorw("failed to fetch users", "error", err)
		return nil, errors.NewInternalError("failed to fetch user information")
	}

	// Build response
	items := make([]dto.UserTrafficStatsItem, 0, len(users))
	for _, u := range users {
		usageData, ok := userUsageMap[u.ID()]
		if !ok {
			continue
		}

		items = append(items, dto.UserTrafficStatsItem{
			UserSID:            u.SID(),
			UserEmail:          u.Email().String(),
			UserName:           u.Name().String(),
			Upload:             usageData.upload,
			Download:           usageData.download,
			Total:              usageData.total,
			SubscriptionsCount: usageData.subscriptionCount,
		})
	}

	response := &dto.UserTrafficStatsResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	uc.logger.Infow("user traffic stats fetched successfully",
		"count", len(items),
		"total", total,
	)

	return response, nil
}

func (uc *GetUserTrafficStatsUseCase) validateQuery(query GetUserTrafficStatsQuery) error {
	if query.From.IsZero() {
		return errors.NewValidationError("from time is required")
	}

	if query.To.IsZero() {
		return errors.NewValidationError("to time is required")
	}

	if query.To.Before(query.From) {
		return errors.NewValidationError("to time must be after from time")
	}

	if query.Page < 0 {
		return errors.NewValidationError("page must be non-negative")
	}

	if query.PageSize < 0 {
		return errors.NewValidationError("page_size must be non-negative")
	}

	return nil
}

func (uc *GetUserTrafficStatsUseCase) getPaginationParams(query GetUserTrafficStatsQuery) (int, int) {
	page := query.Page
	if page == 0 {
		page = constants.DefaultPage
	}

	pageSize := query.PageSize
	if pageSize == 0 {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxPageSize {
		pageSize = constants.MaxPageSize
	}

	return page, pageSize
}

// userUsageData holds aggregated usage data for a user
type userUsageData struct {
	userID            uint
	upload            uint64
	download          uint64
	total             uint64
	subscriptionCount int
}
