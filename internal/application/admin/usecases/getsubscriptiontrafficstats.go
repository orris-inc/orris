package usecases

import (
	"context"
	"sort"
	"time"

	dto "github.com/orris-inc/orris/internal/application/admin/dto"
	"github.com/orris-inc/orris/internal/application/admin/usecases/trafficstatsutil"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
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

	pagination := utils.ValidatePagination(query.Page, query.PageSize)
	timeWindow := trafficstatsutil.CalculateTimeWindow(query.From, query.To)

	// Prepare to merge data from MySQL and Redis
	subscriptionUsageMap := make(map[uint]*subscription.SubscriptionUsageSummary)

	// If query overlaps with Redis data window, get Redis data first
	if timeWindow.IncludesRedisWindow {
		redisFrom, redisTo := timeWindow.GetRedisQueryRange(query.From)

		resourceType := ""
		if query.ResourceType != nil {
			resourceType = *query.ResourceType
		}

		redisTraffic, err := uc.hourlyTrafficCache.GetTrafficGroupedBySubscription(ctx, resourceType, redisFrom, redisTo)
		if err != nil {
			uc.logger.Warnw("failed to get traffic from Redis",
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
			uc.logger.Debugw("got subscription traffic from Redis",
				"subscriptions_count", len(redisTraffic),
			)
		}
	}

	// If query includes historical data (before Redis window), get MySQL data
	if timeWindow.IncludesHistory {
		_, mysqlTo := timeWindow.GetMySQLQueryRange(query.From)

		subscriptionUsages, mysqlTotal, err := uc.usageStatsRepo.GetUsageGroupedBySubscription(
			ctx,
			query.ResourceType,
			query.From,
			mysqlTo,
			1,                               // Get all data without pagination for merging
			maxSubscriptionAggregationLimit, // Safety limit to prevent OOM
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
	}

	// If no data found
	if len(subscriptionUsageMap) == 0 {
		return &dto.SubscriptionTrafficStatsResponse{
			Items:    []dto.SubscriptionTrafficStatsItem{},
			Total:    0,
			Page:     pagination.Page,
			PageSize: pagination.PageSize,
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
	total := int64(len(subscriptionUsages))
	start, end := utils.ApplyPagination(len(subscriptionUsages), pagination.Page, pagination.PageSize)
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
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}

	uc.logger.Infow("subscription traffic stats fetched successfully",
		"count", len(items),
		"total", total,
	)

	return response, nil
}

func (uc *GetSubscriptionTrafficStatsUseCase) validateQuery(query GetSubscriptionTrafficStatsQuery) error {
	return trafficstatsutil.ValidateTimeRange(query.From, query.To)
}
