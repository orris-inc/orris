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

	// Batch fetch usage data for all subscriptions
	usageMap := uc.batchGetUsageBySubscriptions(ctx, subscriptions)

	// Process each subscription
	for _, sub := range subscriptions {
		// Get usage from batch result
		usageSummary := usageMap[sub.ID()]
		if usageSummary == nil {
			usageSummary = &cache.TrafficSummary{}
		}

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
			Status:             sub.EffectiveStatus().String(),
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

// batchGetUsageBySubscriptions retrieves usage for all subscriptions in batch.
// This method uses a graceful degradation strategy: if any data source fails, it logs a warning
// and continues with available data rather than failing the entire request.
func (uc *GetDashboardUseCase) batchGetUsageBySubscriptions(
	ctx context.Context,
	subscriptions []*subscription.Subscription,
) map[uint]*cache.TrafficSummary {
	result := make(map[uint]*cache.TrafficSummary, len(subscriptions))

	if len(subscriptions) == 0 {
		return result
	}

	now := biztime.NowUTC()
	dayAgo := now.Add(-24 * time.Hour)

	// Collect all subscription IDs and find the widest time range
	subscriptionIDs := make([]uint, 0, len(subscriptions))
	var earliestFrom, latestTo time.Time

	for _, sub := range subscriptions {
		subscriptionIDs = append(subscriptionIDs, sub.ID())
		periodStart := sub.CurrentPeriodStart()
		periodEnd := biztime.EndOfDayUTC(sub.CurrentPeriodEnd())

		if earliestFrom.IsZero() || periodStart.Before(earliestFrom) {
			earliestFrom = periodStart
		}
		if latestTo.IsZero() || periodEnd.After(latestTo) {
			latestTo = periodEnd
		}
	}

	// Calculate recent time range (last 24h from Redis)
	recentFrom := earliestFrom
	if recentFrom.Before(dayAgo) {
		recentFrom = dayAgo
	}
	recentTo := latestTo
	if recentTo.After(now) {
		recentTo = now
	}

	// Batch get recent traffic from Redis (last 24h) for all subscriptions
	var recentTraffic map[uint]*cache.TrafficSummary
	if recentFrom.Before(recentTo) && recentFrom.Before(now) {
		var err error
		recentTraffic, err = uc.hourlyCache.GetTotalTrafficBySubscriptionIDs(
			ctx, subscriptionIDs, "", recentFrom, recentTo,
		)
		if err != nil {
			uc.logger.Warnw("failed to get recent traffic from Redis",
				"subscription_ids_count", len(subscriptionIDs),
				"from", recentFrom,
				"to", recentTo,
				"error", err,
			)
		}
	}

	// Batch get historical traffic from MySQL stats (before 24h ago) for all subscriptions
	var historicalTraffic map[uint]*subscription.UsageSummary
	if earliestFrom.Before(dayAgo) {
		historicalTo := dayAgo
		if historicalTo.After(latestTo) {
			historicalTo = latestTo
		}
		var err error
		historicalTraffic, err = uc.usageStatsRepo.GetTotalBySubscriptionIDsGrouped(
			ctx, subscriptionIDs, nil, subscription.GranularityDaily, earliestFrom, historicalTo,
		)
		if err != nil {
			uc.logger.Warnw("failed to get historical traffic from stats",
				"subscription_ids_count", len(subscriptionIDs),
				"from", earliestFrom,
				"to", historicalTo,
				"error", err,
			)
		}
	}

	// Merge results for each subscription
	for _, sub := range subscriptions {
		subID := sub.ID()
		usage := &cache.TrafficSummary{}

		// Add recent traffic if available
		if recentTraffic != nil {
			if t, ok := recentTraffic[subID]; ok {
				usage.Upload += t.Upload
				usage.Download += t.Download
				usage.Total += t.Total
			}
		}

		// Add historical traffic if available and subscription period started before 24h ago
		if historicalTraffic != nil && sub.CurrentPeriodStart().Before(dayAgo) {
			if t, ok := historicalTraffic[subID]; ok {
				usage.Upload += t.Upload
				usage.Download += t.Download
				usage.Total += t.Total
			}
		}

		result[subID] = usage
	}

	return result
}
