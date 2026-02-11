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
	uc.logger.Debugw("fetching user dashboard", "user_id", query.UserID)

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
	usageMap := uc.batchGetUsageBySubscriptions(ctx, subscriptions, planMap)

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

	uc.logger.Debugw("user dashboard fetched successfully",
		"user_id", query.UserID,
		"subscriptions_count", len(response.Subscriptions),
	)

	return response, nil
}

// batchGetUsageBySubscriptions retrieves usage for all subscriptions in batch.
// Subscriptions are grouped by their traffic period (determined by plan's traffic_reset_mode),
// and each group is queried in batch for efficiency.
// This method uses a graceful degradation strategy: if any data source fails, it logs a warning
// and continues with available data rather than failing the entire request.
func (uc *GetDashboardUseCase) batchGetUsageBySubscriptions(
	ctx context.Context,
	subscriptions []*subscription.Subscription,
	planMap map[uint]*subscription.Plan,
) map[uint]*cache.TrafficSummary {
	result := make(map[uint]*cache.TrafficSummary, len(subscriptions))

	if len(subscriptions) == 0 {
		return result
	}

	now := biztime.NowUTC()

	// Use start of yesterday's business day as batch/speed boundary (Lambda architecture)
	// MySQL: complete days before yesterday; Redis: yesterday + today (within 48h TTL)
	recentBoundary := biztime.StartOfDayUTC(now.AddDate(0, 0, -1))

	// Group subscriptions by traffic period for batch querying.
	// calendar_month subscriptions all share the same period (1 group).
	// billing_cycle subscriptions may have different periods (typically 1-3 groups per user).
	type periodKey struct {
		startUnix int64
		endUnix   int64
	}
	type periodGroup struct {
		period          subscription.TrafficPeriod
		subscriptionIDs []uint
	}

	groups := make(map[periodKey]*periodGroup)
	for _, sub := range subscriptions {
		plan := planMap[sub.PlanID()]
		if plan == nil {
			uc.logger.Warnw("plan not found for subscription, using calendar month fallback",
				"subscription_id", sub.ID(), "plan_id", sub.PlanID())
		}
		period := subscription.ResolveTrafficPeriod(plan, sub)
		key := periodKey{startUnix: period.Start.Unix(), endUnix: period.End.Unix()}
		if g, ok := groups[key]; ok {
			g.subscriptionIDs = append(g.subscriptionIDs, sub.ID())
		} else {
			groups[key] = &periodGroup{
				period:          period,
				subscriptionIDs: []uint{sub.ID()},
			}
		}
	}

	// For each period group, fetch usage in batch
	for _, group := range groups {
		periodStart := group.period.Start
		periodEnd := group.period.End
		if periodEnd.After(now) {
			periodEnd = now
		}

		// Calculate recent time range (yesterday + today from Redis)
		recentFrom := periodStart
		if recentFrom.Before(recentBoundary) {
			recentFrom = recentBoundary
		}

		// Batch get recent traffic from Redis (yesterday + today) for this group
		var recentTraffic map[uint]*cache.TrafficSummary
		if recentFrom.Before(periodEnd) && recentFrom.Before(now) {
			var err error
			recentTraffic, err = uc.hourlyCache.GetTotalTrafficBySubscriptionIDs(
				ctx, group.subscriptionIDs, "", recentFrom, periodEnd,
			)
			if err != nil {
				uc.logger.Warnw("failed to get recent traffic from Redis",
					"subscription_ids_count", len(group.subscriptionIDs),
					"from", recentFrom,
					"to", periodEnd,
					"error", err,
				)
			}
		}

		// Batch get historical traffic from MySQL stats (complete days before yesterday) for this group
		var historicalTraffic map[uint]*subscription.UsageSummary
		if periodStart.Before(recentBoundary) {
			historicalTo := recentBoundary.Add(-time.Second)
			if historicalTo.After(periodEnd) {
				historicalTo = periodEnd
			}
			var err error
			historicalTraffic, err = uc.usageStatsRepo.GetTotalBySubscriptionIDsGrouped(
				ctx, group.subscriptionIDs, nil, subscription.GranularityDaily, periodStart, historicalTo,
			)
			if err != nil {
				uc.logger.Warnw("failed to get historical traffic from stats",
					"subscription_ids_count", len(group.subscriptionIDs),
					"from", periodStart,
					"to", historicalTo,
					"error", err,
				)
			}
		}

		// Merge results for each subscription in this group
		for _, subID := range group.subscriptionIDs {
			usage := &cache.TrafficSummary{}

			// Add recent traffic if available
			if recentTraffic != nil {
				if t, ok := recentTraffic[subID]; ok {
					usage.Upload += t.Upload
					usage.Download += t.Download
					usage.Total += t.Total
				}
			}

			// Add historical traffic if available
			if historicalTraffic != nil && periodStart.Before(recentBoundary) {
				if t, ok := historicalTraffic[subID]; ok {
					usage.Upload += t.Upload
					usage.Download += t.Download
					usage.Total += t.Total
				}
			}

			result[subID] = usage
		}
	}

	return result
}
