package usecases

import (
	"context"
	"sort"
	"time"

	dto "github.com/orris-inc/orris/internal/application/admin/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// maxSubscriptionAggregationLimit is the maximum number of records to fetch
	// from MySQL when aggregating subscription data with Redis.
	maxSubscriptionAggregationLimit = 10000
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
	usageStatsRepo     subscription.SubscriptionUsageStatsRepository
	hourlyTrafficCache cache.HourlyTrafficCache
	subscriptionRepo   subscription.SubscriptionRepository
	userRepo           user.Repository
	planRepo           subscription.PlanRepository
	logger             logger.Interface
}

// NewGetSubscriptionTrafficStatsUseCase creates a new GetSubscriptionTrafficStatsUseCase
func NewGetSubscriptionTrafficStatsUseCase(
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyTrafficCache cache.HourlyTrafficCache,
	subscriptionRepo subscription.SubscriptionRepository,
	userRepo user.Repository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *GetSubscriptionTrafficStatsUseCase {
	return &GetSubscriptionTrafficStatsUseCase{
		usageStatsRepo:     usageStatsRepo,
		hourlyTrafficCache: hourlyTrafficCache,
		subscriptionRepo:   subscriptionRepo,
		userRepo:           userRepo,
		planRepo:           planRepo,
		logger:             logger,
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
	adjustedTo := biztime.EndOfDayUTC(query.To)

	// Calculate today's boundary in business timezone
	now := biztime.NowUTC()
	todayStart := biztime.StartOfDayUTC(now)

	// Determine if query includes today (unaggregated data)
	includesToday := !adjustedTo.Before(todayStart)
	includesHistory := query.From.Before(todayStart)

	// Prepare to merge data from MySQL and Redis
	subscriptionUsageMap := make(map[uint]*subscription.SubscriptionUsageSummary)
	var total int64

	// If query includes today, get Redis data first
	if includesToday {
		redisFrom := todayStart
		if query.From.After(todayStart) {
			redisFrom = query.From
		}

		resourceType := ""
		if query.ResourceType != nil {
			resourceType = *query.ResourceType
		}

		redisTraffic, err := uc.hourlyTrafficCache.GetTrafficGroupedBySubscription(ctx, resourceType, redisFrom, adjustedTo)
		if err != nil {
			uc.logger.Warnw("failed to get today's traffic from Redis",
				"error", err,
			)
		} else {
			for subID, traffic := range redisTraffic {
				subscriptionUsageMap[subID] = &subscription.SubscriptionUsageSummary{
					SubscriptionID: subID,
					Upload:         traffic.Upload,
					Download:       traffic.Download,
					Total:          traffic.Total,
				}
			}
			uc.logger.Debugw("got today's subscription traffic from Redis",
				"subscriptions_count", len(redisTraffic),
			)
		}
	}

	// If query includes historical data, get MySQL data
	if includesHistory {
		mysqlTo := adjustedTo
		if includesToday {
			// Exclude today from MySQL query
			mysqlTo = todayStart.Add(-time.Nanosecond)
		}

		subscriptionUsages, mysqlTotal, err := uc.usageStatsRepo.GetUsageGroupedBySubscription(
			ctx,
			query.ResourceType,
			query.From,
			mysqlTo,
			1,                                // Get all data without pagination for merging
			maxSubscriptionAggregationLimit,  // Safety limit to prevent OOM
		)
		if err != nil {
			uc.logger.Errorw("failed to fetch subscription usage", "error", err)
			return nil, errors.NewInternalError("failed to fetch subscription usage")
		}

		// Warn if data may be truncated
		if mysqlTotal > int64(maxSubscriptionAggregationLimit) {
			uc.logger.Warnw("subscription traffic data may be incomplete due to aggregation limit",
				"total_records", mysqlTotal,
				"limit", maxSubscriptionAggregationLimit,
				"from", query.From,
				"to", mysqlTo,
			)
		}

		// Merge MySQL data with Redis data
		for _, usage := range subscriptionUsages {
			if existing, ok := subscriptionUsageMap[usage.SubscriptionID]; ok {
				existing.Upload += usage.Upload
				existing.Download += usage.Download
				existing.Total += usage.Total
			} else {
				subscriptionUsageMap[usage.SubscriptionID] = &subscription.SubscriptionUsageSummary{
					SubscriptionID: usage.SubscriptionID,
					Upload:         usage.Upload,
					Download:       usage.Download,
					Total:          usage.Total,
				}
			}
		}

		// Use MySQL total as a baseline (may not be accurate when today has new subscriptions)
		if !includesToday {
			total = mysqlTotal
		}
	}

	// If no data found
	if len(subscriptionUsageMap) == 0 {
		return &dto.SubscriptionTrafficStatsResponse{
			Items:    []dto.SubscriptionTrafficStatsItem{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	// Convert map to slice and sort by total descending
	subscriptionUsages := make([]subscription.SubscriptionUsageSummary, 0, len(subscriptionUsageMap))
	for _, usage := range subscriptionUsageMap {
		subscriptionUsages = append(subscriptionUsages, *usage)
	}
	sort.Slice(subscriptionUsages, func(i, j int) bool {
		return subscriptionUsages[i].Total > subscriptionUsages[j].Total
	})

	// Apply pagination
	total = int64(len(subscriptionUsages))
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(subscriptionUsages) {
		start = len(subscriptionUsages)
	}
	if end > len(subscriptionUsages) {
		end = len(subscriptionUsages)
	}
	pagedUsages := subscriptionUsages[start:end]

	// Extract subscription IDs
	subscriptionIDs := make([]uint, len(pagedUsages))
	for i, usage := range pagedUsages {
		subscriptionIDs[i] = usage.SubscriptionID
	}

	// Fetch subscriptions using batch query
	subscriptions, err := uc.subscriptionRepo.GetByIDs(ctx, subscriptionIDs)
	if err != nil {
		uc.logger.Errorw("failed to fetch subscriptions", "error", err)
		return nil, errors.NewInternalError("failed to fetch subscription information")
	}

	// Extract user IDs and plan IDs
	userIDs := make([]uint, 0, len(subscriptions))
	planIDs := make([]uint, 0, len(subscriptions))
	for _, sub := range subscriptions {
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
	items := make([]dto.SubscriptionTrafficStatsItem, 0, len(pagedUsages))
	for _, usage := range pagedUsages {
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
	if page < 1 {
		page = constants.DefaultPage
	}

	pageSize := query.PageSize
	if pageSize < 1 {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxPageSize {
		pageSize = constants.MaxPageSize
	}

	return page, pageSize
}
