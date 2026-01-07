package usecases

import (
	"context"
	"time"

	dto "github.com/orris-inc/orris/internal/application/admin/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/user"
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
	usageStatsRepo   subscription.SubscriptionUsageStatsRepository
	subscriptionRepo subscription.SubscriptionRepository
	userRepo         user.Repository
	logger           logger.Interface
}

// NewGetTrafficRankingUseCase creates a new GetTrafficRankingUseCase
func NewGetTrafficRankingUseCase(
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	userRepo user.Repository,
	logger logger.Interface,
) *GetTrafficRankingUseCase {
	return &GetTrafficRankingUseCase{
		usageStatsRepo:   usageStatsRepo,
		subscriptionRepo: subscriptionRepo,
		userRepo:         userRepo,
		logger:           logger,
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

	// Get top subscriptions by usage from subscription_usage_stats table
	topSubscriptions, err := uc.usageStatsRepo.GetTopSubscriptionsByUsage(
		ctx,
		query.ResourceType,
		query.From,
		adjustedTo,
		limit*2, // Fetch more to account for aggregation
	)
	if err != nil {
		uc.logger.Errorw("failed to fetch top subscriptions", "error", err)
		return nil, errors.NewInternalError("failed to fetch traffic ranking")
	}

	if len(topSubscriptions) == 0 {
		return &dto.TrafficRankingResponse{
			Items: []dto.TrafficRankingItem{},
		}, nil
	}

	// Aggregate by user
	userUsageMap := make(map[uint]*userRankingData)
	subscriptionIDs := make([]uint, 0, len(topSubscriptions))

	for _, subUsage := range topSubscriptions {
		if subUsage.SubscriptionID != 0 {
			subscriptionIDs = append(subscriptionIDs, subUsage.SubscriptionID)
		}
	}

	// Fetch subscriptions using batch query to get user IDs
	subscriptions, err := uc.subscriptionRepo.GetByIDs(ctx, subscriptionIDs)
	if err != nil {
		uc.logger.Errorw("failed to fetch subscriptions", "error", err)
		return nil, errors.NewInternalError("failed to fetch subscription information")
	}

	// Aggregate usage by user
	for _, subUsage := range topSubscriptions {
		sub, ok := subscriptions[subUsage.SubscriptionID]
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

	// Convert to slice and sort by total
	userRankings := make([]*userRankingData, 0, len(userUsageMap))
	for _, data := range userUsageMap {
		userRankings = append(userRankings, data)
	}

	// Sort by total descending
	for i := 0; i < len(userRankings)-1; i++ {
		for j := i + 1; j < len(userRankings); j++ {
			if userRankings[j].total > userRankings[i].total {
				userRankings[i], userRankings[j] = userRankings[j], userRankings[i]
			}
		}
	}

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

	// Get top subscriptions by usage from subscription_usage_stats table
	topSubscriptions, err := uc.usageStatsRepo.GetTopSubscriptionsByUsage(
		ctx,
		query.ResourceType,
		query.From,
		adjustedTo,
		limit,
	)
	if err != nil {
		uc.logger.Errorw("failed to fetch top subscriptions", "error", err)
		return nil, errors.NewInternalError("failed to fetch traffic ranking")
	}

	if len(topSubscriptions) == 0 {
		return &dto.TrafficRankingResponse{
			Items: []dto.TrafficRankingItem{},
		}, nil
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
