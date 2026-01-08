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
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	defaultRankingLimit = 10
	maxRankingLimit     = 100
)

// GetTrafficRankingQuery represents the query parameters for traffic ranking
type GetTrafficRankingQuery struct {
	From         time.Time
	To           time.Time
	ResourceType *string
	Limit        int
	RankingType  string // "user" or "subscription"
}

// GetTrafficRankingUseCase handles retrieving traffic rankings
type GetTrafficRankingUseCase struct {
	usageStatsRepo     subscription.SubscriptionUsageStatsRepository
	hourlyTrafficCache cache.HourlyTrafficCache
	subscriptionRepo   subscription.SubscriptionRepository
	userRepo           user.Repository
	logger             logger.Interface
}

// NewGetTrafficRankingUseCase creates a new GetTrafficRankingUseCase
func NewGetTrafficRankingUseCase(
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyTrafficCache cache.HourlyTrafficCache,
	subscriptionRepo subscription.SubscriptionRepository,
	userRepo user.Repository,
	logger logger.Interface,
) *GetTrafficRankingUseCase {
	return &GetTrafficRankingUseCase{
		usageStatsRepo:     usageStatsRepo,
		hourlyTrafficCache: hourlyTrafficCache,
		subscriptionRepo:   subscriptionRepo,
		userRepo:           userRepo,
		logger:             logger,
	}
}

// ExecuteUserRanking retrieves top users by traffic usage
func (uc *GetTrafficRankingUseCase) ExecuteUserRanking(
	ctx context.Context,
	query GetTrafficRankingQuery,
) (*dto.TrafficRankingResponse, error) {
	uc.logger.Infow("fetching user traffic ranking",
		"from", query.From,
		"to", query.To,
		"resource_type", query.ResourceType,
		"limit", query.Limit,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid traffic ranking query", "error", err)
		return nil, err
	}

	limit := uc.getLimit(query)

	// Adjust 'to' time to end of day to include all records from that day
	adjustedTo := biztime.EndOfDayUTC(query.To)

	// Calculate today's boundary in business timezone
	now := biztime.NowUTC()
	todayStart := biztime.StartOfDayUTC(now)

	// Determine if query includes today (unaggregated data)
	includesToday := !adjustedTo.Before(todayStart)
	includesHistory := query.From.Before(todayStart)

	// Prepare to merge subscription usage data from MySQL and Redis
	subscriptionUsageMap := make(map[uint]*subscription.SubscriptionUsageSummary)

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
		}
	}

	// If query includes historical data, get MySQL data
	if includesHistory {
		mysqlTo := adjustedTo
		if includesToday {
			mysqlTo = todayStart.Add(-time.Nanosecond)
		}

		topSubscriptions, err := uc.usageStatsRepo.GetTopSubscriptionsByUsage(
			ctx,
			query.ResourceType,
			query.From,
			mysqlTo,
			limit*10, // Fetch more to account for aggregation by user
		)
		if err != nil {
			uc.logger.Errorw("failed to fetch top subscriptions", "error", err)
			return nil, errors.NewInternalError("failed to fetch traffic ranking")
		}

		// Merge MySQL data with Redis data
		for _, subUsage := range topSubscriptions {
			if existing, ok := subscriptionUsageMap[subUsage.SubscriptionID]; ok {
				existing.Upload += subUsage.Upload
				existing.Download += subUsage.Download
				existing.Total += subUsage.Total
			} else {
				subscriptionUsageMap[subUsage.SubscriptionID] = &subscription.SubscriptionUsageSummary{
					SubscriptionID: subUsage.SubscriptionID,
					Upload:         subUsage.Upload,
					Download:       subUsage.Download,
					Total:          subUsage.Total,
				}
			}
		}
	}

	if len(subscriptionUsageMap) == 0 {
		return &dto.TrafficRankingResponse{
			Items: []dto.TrafficRankingItem{},
		}, nil
	}

	// Extract subscription IDs
	subscriptionIDs := make([]uint, 0, len(subscriptionUsageMap))
	for subID := range subscriptionUsageMap {
		if subID != 0 {
			subscriptionIDs = append(subscriptionIDs, subID)
		}
	}

	// Fetch subscriptions using batch query to get user IDs
	subscriptions, err := uc.subscriptionRepo.GetByIDs(ctx, subscriptionIDs)
	if err != nil {
		uc.logger.Errorw("failed to fetch subscriptions", "error", err)
		return nil, errors.NewInternalError("failed to fetch subscription information")
	}

	// Aggregate usage by user
	userUsageMap := make(map[uint]*userRankingData)
	for subID, subUsage := range subscriptionUsageMap {
		sub, ok := subscriptions[subID]
		if !ok {
			continue
		}

		userID := sub.UserID()
		if existing, exists := userUsageMap[userID]; exists {
			existing.upload += subUsage.Upload
			existing.download += subUsage.Download
			existing.total += subUsage.Total
		} else {
			userUsageMap[userID] = &userRankingData{
				userID:   userID,
				upload:   subUsage.Upload,
				download: subUsage.Download,
				total:    subUsage.Total,
			}
		}
	}

	// Convert to slice and sort by total descending
	userRankings := make([]*userRankingData, 0, len(userUsageMap))
	for _, data := range userUsageMap {
		userRankings = append(userRankings, data)
	}
	sort.Slice(userRankings, func(i, j int) bool {
		return userRankings[i].total > userRankings[j].total
	})

	// Apply limit
	if len(userRankings) > limit {
		userRankings = userRankings[:limit]
	}

	// Fetch user details
	userIDs := make([]uint, len(userRankings))
	for i, ranking := range userRankings {
		userIDs[i] = ranking.userID
	}

	users, err := uc.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		uc.logger.Errorw("failed to fetch users", "error", err)
		return nil, errors.NewInternalError("failed to fetch user information")
	}

	usersMap := make(map[uint]*user.User)
	for _, u := range users {
		usersMap[u.ID()] = u
	}

	// Build response
	items := make([]dto.TrafficRankingItem, 0, len(userRankings))
	for rank, ranking := range userRankings {
		u, ok := usersMap[ranking.userID]
		if !ok {
			continue
		}

		items = append(items, dto.TrafficRankingItem{
			Rank:     rank + 1,
			ID:       u.SID(),
			Name:     u.Email().String(), // Use email as name for users
			Upload:   ranking.upload,
			Download: ranking.download,
			Total:    ranking.total,
		})
	}

	uc.logger.Infow("user traffic ranking fetched successfully", "count", len(items))

	return &dto.TrafficRankingResponse{Items: items}, nil
}

// ExecuteSubscriptionRanking retrieves top subscriptions by traffic usage
func (uc *GetTrafficRankingUseCase) ExecuteSubscriptionRanking(
	ctx context.Context,
	query GetTrafficRankingQuery,
) (*dto.TrafficRankingResponse, error) {
	uc.logger.Infow("fetching subscription traffic ranking",
		"from", query.From,
		"to", query.To,
		"resource_type", query.ResourceType,
		"limit", query.Limit,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid traffic ranking query", "error", err)
		return nil, err
	}

	limit := uc.getLimit(query)

	// Adjust 'to' time to end of day to include all records from that day
	adjustedTo := biztime.EndOfDayUTC(query.To)

	// Calculate today's boundary in business timezone
	now := biztime.NowUTC()
	todayStart := biztime.StartOfDayUTC(now)

	// Determine if query includes today (unaggregated data)
	includesToday := !adjustedTo.Before(todayStart)
	includesHistory := query.From.Before(todayStart)

	// Prepare to merge subscription usage data from MySQL and Redis
	subscriptionUsageMap := make(map[uint]*subscription.SubscriptionUsageSummary)

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
		}
	}

	// If query includes historical data, get MySQL data
	if includesHistory {
		mysqlTo := adjustedTo
		if includesToday {
			mysqlTo = todayStart.Add(-time.Nanosecond)
		}

		topSubscriptions, err := uc.usageStatsRepo.GetTopSubscriptionsByUsage(
			ctx,
			query.ResourceType,
			query.From,
			mysqlTo,
			limit*2, // Fetch more to account for merging
		)
		if err != nil {
			uc.logger.Errorw("failed to fetch top subscriptions", "error", err)
			return nil, errors.NewInternalError("failed to fetch traffic ranking")
		}

		// Merge MySQL data with Redis data
		for _, subUsage := range topSubscriptions {
			if existing, ok := subscriptionUsageMap[subUsage.SubscriptionID]; ok {
				existing.Upload += subUsage.Upload
				existing.Download += subUsage.Download
				existing.Total += subUsage.Total
			} else {
				subscriptionUsageMap[subUsage.SubscriptionID] = &subscription.SubscriptionUsageSummary{
					SubscriptionID: subUsage.SubscriptionID,
					Upload:         subUsage.Upload,
					Download:       subUsage.Download,
					Total:          subUsage.Total,
				}
			}
		}
	}

	if len(subscriptionUsageMap) == 0 {
		return &dto.TrafficRankingResponse{
			Items: []dto.TrafficRankingItem{},
		}, nil
	}

	// Convert to slice and sort by total descending
	topSubscriptions := make([]subscription.SubscriptionUsageSummary, 0, len(subscriptionUsageMap))
	for _, usage := range subscriptionUsageMap {
		topSubscriptions = append(topSubscriptions, *usage)
	}
	sort.Slice(topSubscriptions, func(i, j int) bool {
		return topSubscriptions[i].Total > topSubscriptions[j].Total
	})

	// Apply limit
	if len(topSubscriptions) > limit {
		topSubscriptions = topSubscriptions[:limit]
	}

	// Fetch subscription details using batch query
	subscriptionIDs := make([]uint, 0, len(topSubscriptions))
	for _, subUsage := range topSubscriptions {
		if subUsage.SubscriptionID != 0 {
			subscriptionIDs = append(subscriptionIDs, subUsage.SubscriptionID)
		}
	}

	subscriptionsMap, err := uc.subscriptionRepo.GetByIDs(ctx, subscriptionIDs)
	if err != nil {
		uc.logger.Errorw("failed to fetch subscriptions", "error", err)
		return nil, errors.NewInternalError("failed to fetch subscription information")
	}

	// Build response
	items := make([]dto.TrafficRankingItem, 0, len(topSubscriptions))
	for rank, subUsage := range topSubscriptions {
		sub, ok := subscriptionsMap[subUsage.SubscriptionID]
		if !ok {
			continue
		}

		items = append(items, dto.TrafficRankingItem{
			Rank:     rank + 1,
			ID:       sub.SID(),
			Name:     sub.SID(), // Use SID as name for subscriptions
			Upload:   subUsage.Upload,
			Download: subUsage.Download,
			Total:    subUsage.Total,
		})
	}

	uc.logger.Infow("subscription traffic ranking fetched successfully", "count", len(items))

	return &dto.TrafficRankingResponse{Items: items}, nil
}

func (uc *GetTrafficRankingUseCase) validateQuery(query GetTrafficRankingQuery) error {
	if query.From.IsZero() {
		return errors.NewValidationError("from time is required")
	}

	if query.To.IsZero() {
		return errors.NewValidationError("to time is required")
	}

	if query.To.Before(query.From) {
		return errors.NewValidationError("to time must be after from time")
	}

	if query.Limit < 0 {
		return errors.NewValidationError("limit must be non-negative")
	}

	if query.RankingType != "" && query.RankingType != "user" && query.RankingType != "subscription" {
		return errors.NewValidationError("ranking_type must be 'user' or 'subscription'")
	}

	return nil
}

func (uc *GetTrafficRankingUseCase) getLimit(query GetTrafficRankingQuery) int {
	limit := query.Limit
	if limit == 0 {
		limit = defaultRankingLimit
	}
	if limit > maxRankingLimit {
		limit = maxRankingLimit
	}
	return limit
}

// userRankingData holds aggregated usage data for ranking
type userRankingData struct {
	userID   uint
	upload   uint64
	download uint64
	total    uint64
}
