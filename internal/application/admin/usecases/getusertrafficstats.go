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
	// maxUserAggregationLimit is the maximum number of records to fetch
	// from MySQL when aggregating user traffic data with Redis.
	maxUserAggregationLimit = 10000
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
	usageStatsRepo     subscription.SubscriptionUsageStatsRepository
	hourlyTrafficCache cache.HourlyTrafficCache
	subscriptionRepo   subscription.SubscriptionRepository
	userRepo           user.Repository
	logger             logger.Interface
}

// NewGetUserTrafficStatsUseCase creates a new GetUserTrafficStatsUseCase
func NewGetUserTrafficStatsUseCase(
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyTrafficCache cache.HourlyTrafficCache,
	subscriptionRepo subscription.SubscriptionRepository,
	userRepo user.Repository,
	logger logger.Interface,
) *GetUserTrafficStatsUseCase {
	return &GetUserTrafficStatsUseCase{
		usageStatsRepo:     usageStatsRepo,
		hourlyTrafficCache: hourlyTrafficCache,
		subscriptionRepo:   subscriptionRepo,
		userRepo:           userRepo,
		logger:             logger,
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

	pagination := utils.ValidatePagination(query.Page, query.PageSize)
	timeWindow := trafficstatsutil.CalculateTimeWindow(query.From, query.To)

	// Prepare to merge subscription usage data from MySQL and Redis
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
			1,                       // Get all data without pagination for merging
			maxUserAggregationLimit, // Safety limit to prevent OOM
		)
		if err != nil {
			uc.logger.Errorw("failed to fetch subscription usage", "error", err)
			return nil, errors.NewInternalError("failed to fetch subscription usage")
		}

		// Warn if data may be truncated
		if mysqlTotal > int64(maxUserAggregationLimit) {
			uc.logger.Warnw("user traffic data may be incomplete due to aggregation limit",
				"total_records", mysqlTotal,
				"limit", maxUserAggregationLimit,
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
		return &dto.UserTrafficStatsResponse{
			Items:    []dto.UserTrafficStatsItem{},
			Total:    0,
			Page:     pagination.Page,
			PageSize: pagination.PageSize,
		}, nil
	}

	// Extract subscription IDs
	subscriptionIDs := make([]uint, 0, len(subscriptionUsageMap))
	for subID := range subscriptionUsageMap {
		subscriptionIDs = append(subscriptionIDs, subID)
	}

	// Fetch subscriptions using batch query
	subscriptions, err := uc.subscriptionRepo.GetByIDs(ctx, subscriptionIDs)
	if err != nil {
		uc.logger.Errorw("failed to fetch subscriptions", "error", err)
		return nil, errors.NewInternalError("failed to fetch subscription information")
	}

	// Aggregate usage by user
	userUsageMap := make(map[uint]*userUsageData)
	for subID, usage := range subscriptionUsageMap {
		sub, ok := subscriptions[subID]
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

	// Convert to slice and sort by total descending
	userUsages := make([]*userUsageData, 0, len(userUsageMap))
	for _, data := range userUsageMap {
		userUsages = append(userUsages, data)
	}
	sort.Slice(userUsages, func(i, j int) bool {
		return userUsages[i].total > userUsages[j].total
	})

	// Apply pagination
	total := int64(len(userUsages))
	start, end := utils.ApplyPagination(len(userUsages), pagination.Page, pagination.PageSize)
	pagedUsages := userUsages[start:end]

	// Fetch user details
	userIDs := make([]uint, len(pagedUsages))
	for i, data := range pagedUsages {
		userIDs[i] = data.userID
	}

	users, err := uc.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		uc.logger.Errorw("failed to fetch users", "error", err)
		return nil, errors.NewInternalError("failed to fetch user information")
	}

	// Create users map for quick lookup
	usersMap := make(map[uint]*user.User)
	for _, u := range users {
		usersMap[u.ID()] = u
	}

	// Build response
	items := make([]dto.UserTrafficStatsItem, 0, len(pagedUsages))
	for _, usageData := range pagedUsages {
		u, ok := usersMap[usageData.userID]
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
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}

	uc.logger.Infow("user traffic stats fetched successfully",
		"count", len(items),
		"total", total,
	)

	return response, nil
}

func (uc *GetUserTrafficStatsUseCase) validateQuery(query GetUserTrafficStatsQuery) error {
	return trafficstatsutil.ValidateTimeRange(query.From, query.To)
}

// userUsageData holds aggregated usage data for a user
type userUsageData struct {
	userID            uint
	upload            uint64
	download          uint64
	total             uint64
	subscriptionCount int
}
