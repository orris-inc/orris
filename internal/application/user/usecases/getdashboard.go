package usecases

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/application/user/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetDashboardQuery represents the query parameters for user dashboard
type GetDashboardQuery struct {
	UserID uint
}

// GetDashboardUseCase handles retrieving user dashboard data
type GetDashboardUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	usageStatsRepo   subscription.SubscriptionUsageStatsRepository
	hourlyCache      cache.HourlyTrafficCache
	planRepo         subscription.PlanRepository
	logger           logger.Interface
}

// NewGetDashboardUseCase creates a new GetDashboardUseCase
func NewGetDashboardUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyCache cache.HourlyTrafficCache,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *GetDashboardUseCase {
	return &GetDashboardUseCase{
		subscriptionRepo: subscriptionRepo,
		usageStatsRepo:   usageStatsRepo,
		hourlyCache:      hourlyCache,
		planRepo:         planRepo,
		logger:           logger,
	}
}

// Execute retrieves user dashboard data including subscriptions and usage
func (uc *GetDashboardUseCase) Execute(
	ctx context.Context,
	query GetDashboardQuery,
) (*dto.DashboardResponse, error) {
	uc.logger.Infow("fetching user dashboard", "user_id", query.UserID)

	if query.UserID == 0 {
		return nil, errors.NewValidationError("user_id is required")
	}

	// Get user's subscriptions
	subscriptions, err := uc.subscriptionRepo.GetByUserID(ctx, query.UserID)
	if err != nil {
		uc.logger.Errorw("failed to fetch user subscriptions", "user_id", query.UserID, "error", err)
		return nil, errors.NewInternalError("failed to fetch subscriptions")
	}

	// Prepare response
	response := &dto.DashboardResponse{
		Subscriptions: make([]*dto.DashboardSubscriptionDTO, 0, len(subscriptions)),
		TotalUsage: &dto.UsageSummary{
			Upload:   0,
			Download: 0,
			Total:    0,
		},
	}

	// Collect unique plan IDs for batch fetch
	planIDSet := make(map[uint]struct{}, len(subscriptions))
	for _, sub := range subscriptions {
		planIDSet[sub.PlanID()] = struct{}{}
	}
	planIDs := make([]uint, 0, len(planIDSet))
	for id := range planIDSet {
		planIDs = append(planIDs, id)
	}

	// Batch fetch plans
	planMap := make(map[uint]*subscription.Plan)
	if len(planIDs) > 0 {
		plans, err := uc.planRepo.GetByIDs(ctx, planIDs)
		if err != nil {
			uc.logger.Warnw("failed to fetch plans", "error", err)
		} else {
			for _, plan := range plans {
				planMap[plan.ID()] = plan
			}
		}
	}

	// Process each subscription
	for _, sub := range subscriptions {
		// Get usage for current period combining Redis (recent 24h) and MySQL stats (historical)
		periodStart := sub.CurrentPeriodStart()
		periodEnd := biztime.EndOfDayUTC(sub.CurrentPeriodEnd())

		usageSummary := uc.getTotalUsageBySubscriptionID(ctx, sub.ID(), periodStart, periodEnd)

		// Calculate subscription usage summary
		subUsage := &dto.UsageSummary{
			Upload:   usageSummary.Upload,
			Download: usageSummary.Download,
			Total:    usageSummary.Total,
		}

		// Add to total usage
		response.TotalUsage.Upload += subUsage.Upload
		response.TotalUsage.Download += subUsage.Download
		response.TotalUsage.Total += subUsage.Total

		// Build subscription DTO
		subDTO := &dto.DashboardSubscriptionDTO{
			SID:                sub.SID(),
			Status:             sub.Status().String(),
			CurrentPeriodStart: sub.CurrentPeriodStart(),
			CurrentPeriodEnd:   sub.CurrentPeriodEnd(),
			IsActive:           sub.IsActive(),
			Usage:              subUsage,
		}

		// Add plan info if available
		if plan, ok := planMap[sub.PlanID()]; ok {
			var limits map[string]interface{}
			if plan.Features() != nil {
				limits = plan.Features().Limits
			}
			subDTO.Plan = &dto.DashboardPlanDTO{
				SID:      plan.SID(),
				Name:     plan.Name(),
				PlanType: plan.PlanType().String(),
				Limits:   limits,
			}
		}

		response.Subscriptions = append(response.Subscriptions, subDTO)
	}

	uc.logger.Infow("user dashboard fetched successfully",
		"user_id", query.UserID,
		"subscriptions_count", len(response.Subscriptions),
	)

	return response, nil
}

// getTotalUsageBySubscriptionID retrieves total usage by combining Redis (recent 24h) and MySQL stats (historical).
// This method uses a graceful degradation strategy: if any data source fails, it logs a warning
// and continues with available data rather than failing the entire request.
func (uc *GetDashboardUseCase) getTotalUsageBySubscriptionID(
	ctx context.Context,
	subscriptionID uint,
	from, to time.Time,
) *cache.TrafficSummary {
	now := biztime.NowUTC()
	dayAgo := now.Add(-24 * time.Hour)

	result := &cache.TrafficSummary{}
	subscriptionIDs := []uint{subscriptionID}

	// Determine time boundaries for recent data (last 24h from Redis)
	recentFrom := from
	if recentFrom.Before(dayAgo) {
		recentFrom = dayAgo
	}

	// Get recent traffic from Redis (last 24h)
	if recentFrom.Before(to) && recentFrom.Before(now) {
		recentTo := to
		if recentTo.After(now) {
			recentTo = now
		}
		recentTraffic, err := uc.hourlyCache.GetTotalTrafficBySubscriptionIDs(
			ctx, subscriptionIDs, "", recentFrom, recentTo,
		)
		if err != nil {
			uc.logger.Warnw("failed to get recent traffic from Redis",
				"subscription_id", subscriptionID,
				"from", recentFrom,
				"to", recentTo,
				"error", err,
			)
		} else if t, ok := recentTraffic[subscriptionID]; ok {
			result.Upload += t.Upload
			result.Download += t.Download
			result.Total += t.Total
		}
	}

	// Get historical traffic from MySQL stats (before 24h ago)
	if from.Before(dayAgo) {
		historicalTo := dayAgo
		if historicalTo.After(to) {
			historicalTo = to
		}
		historicalTraffic, err := uc.usageStatsRepo.GetTotalBySubscriptionIDs(
			ctx, subscriptionIDs, nil, subscription.GranularityDaily, from, historicalTo,
		)
		if err != nil {
			uc.logger.Warnw("failed to get historical traffic from stats",
				"subscription_id", subscriptionID,
				"from", from,
				"to", historicalTo,
				"error", err,
			)
		} else if historicalTraffic != nil {
			result.Upload += historicalTraffic.Upload
			result.Download += historicalTraffic.Download
			result.Total += historicalTraffic.Total
		}
	}

	return result
}
