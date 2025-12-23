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
	"github.com/orris-inc/orris/internal/shared/utils"
)

// GetSubscriptionTrafficStatsQuery represents the query parameters for subscription traffic statistics
type GetSubscriptionTrafficStatsQuery struct {
	From         time.Time
	To           time.Time
	ResourceType *string
	Page         int
	PageSize     int
}

// GetSubscriptionTrafficStatsUseCase handles retrieving traffic statistics grouped by subscription
type GetSubscriptionTrafficStatsUseCase struct {
	usageRepo        subscription.SubscriptionUsageRepository
	subscriptionRepo subscription.SubscriptionRepository
	userRepo         user.Repository
	planRepo         subscription.PlanRepository
	logger           logger.Interface
}

// NewGetSubscriptionTrafficStatsUseCase creates a new GetSubscriptionTrafficStatsUseCase
func NewGetSubscriptionTrafficStatsUseCase(
	usageRepo subscription.SubscriptionUsageRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	userRepo user.Repository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *GetSubscriptionTrafficStatsUseCase {
	return &GetSubscriptionTrafficStatsUseCase{
		usageRepo:        usageRepo,
		subscriptionRepo: subscriptionRepo,
		userRepo:         userRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}

// Execute retrieves subscription traffic statistics
func (uc *GetSubscriptionTrafficStatsUseCase) Execute(
	ctx context.Context,
	query GetSubscriptionTrafficStatsQuery,
) (*dto.SubscriptionTrafficStatsResponse, error) {
	uc.logger.Infow("fetching subscription traffic stats",
		"from", query.From,
		"to", query.To,
		"resource_type", query.ResourceType,
		"page", query.Page,
		"page_size", query.PageSize,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid subscription traffic stats query", "error", err)
		return nil, err
	}

	page, pageSize := uc.getPaginationParams(query)

	// Adjust 'to' time to end of day to include all records from that day
	adjustedTo := utils.AdjustToEndOfDay(query.To)

	// Get usage data grouped by subscription
	subscriptionUsages, total, err := uc.usageRepo.GetUsageGroupedBySubscription(
		ctx,
		query.ResourceType,
		query.From,
		adjustedTo,
		page,
		pageSize,
	)
	if err != nil {
		uc.logger.Errorw("failed to fetch subscription usage", "error", err)
		return nil, errors.NewInternalError("failed to fetch subscription usage")
	}

	if len(subscriptionUsages) == 0 {
		return &dto.SubscriptionTrafficStatsResponse{
			Items:    []dto.SubscriptionTrafficStatsItem{},
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
	userIDs := make([]uint, 0, len(subscriptionUsages))
	planIDs := make([]uint, 0, len(subscriptionUsages))

	for _, subID := range subscriptionIDs {
		sub, err := uc.subscriptionRepo.GetByID(ctx, subID)
		if err != nil {
			uc.logger.Warnw("failed to fetch subscription", "subscription_id", subID, "error", err)
			continue
		}
		subscriptions[subID] = sub
		userIDs = append(userIDs, sub.UserID())
		planIDs = append(planIDs, sub.PlanID())
	}

	// Fetch users
	usersMap := make(map[uint]*user.User)
	users, err := uc.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		uc.logger.Errorw("failed to fetch users", "error", err)
		return nil, errors.NewInternalError("failed to fetch user information")
	}
	for _, u := range users {
		usersMap[u.ID()] = u
	}

	// Fetch plans
	plansMap := make(map[uint]*subscription.Plan)
	plans, err := uc.planRepo.GetByIDs(ctx, planIDs)
	if err != nil {
		uc.logger.Errorw("failed to fetch plans", "error", err)
		return nil, errors.NewInternalError("failed to fetch plan information")
	}
	for _, p := range plans {
		plansMap[p.ID()] = p
	}

	// Build response items
	items := make([]dto.SubscriptionTrafficStatsItem, 0, len(subscriptionUsages))
	for _, usage := range subscriptionUsages {
		sub, ok := subscriptions[usage.SubscriptionID]
		if !ok {
			continue
		}

		u, userOk := usersMap[sub.UserID()]
		plan, planOk := plansMap[sub.PlanID()]

		item := dto.SubscriptionTrafficStatsItem{
			SubscriptionSID: sub.SID(),
			Status:          sub.Status().String(),
			Upload:          usage.Upload,
			Download:        usage.Download,
			Total:           usage.Total,
		}

		if userOk {
			item.UserSID = u.SID()
			item.UserEmail = u.Email().String()
		}

		if planOk {
			item.PlanName = plan.Name()
		}

		items = append(items, item)
	}

	response := &dto.SubscriptionTrafficStatsResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	uc.logger.Infow("subscription traffic stats fetched successfully",
		"count", len(items),
		"total", total,
	)

	return response, nil
}

func (uc *GetSubscriptionTrafficStatsUseCase) validateQuery(query GetSubscriptionTrafficStatsQuery) error {
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

func (uc *GetSubscriptionTrafficStatsUseCase) getPaginationParams(query GetSubscriptionTrafficStatsQuery) (int, int) {
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
