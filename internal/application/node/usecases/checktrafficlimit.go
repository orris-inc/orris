package usecases

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type CheckTrafficLimitQuery struct {
	NodeID         uint
	UserID         *uint
	SubscriptionID uint
}

type TrafficLimitResult struct {
	Exceeded       bool    `json:"exceeded"`
	TotalTraffic   uint64  `json:"total_traffic"`
	TrafficLimit   uint64  `json:"traffic_limit"`
	RemainingBytes uint64  `json:"remaining_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
}

type CheckTrafficLimitUseCase struct {
	usageStatsRepo   subscription.SubscriptionUsageStatsRepository
	hourlyCache      cache.HourlyTrafficCache
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	logger           logger.Interface
}

func NewCheckTrafficLimitUseCase(
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyCache cache.HourlyTrafficCache,
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *CheckTrafficLimitUseCase {
	return &CheckTrafficLimitUseCase{
		usageStatsRepo:   usageStatsRepo,
		hourlyCache:      hourlyCache,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}

func (uc *CheckTrafficLimitUseCase) Execute(
	ctx context.Context,
	query CheckTrafficLimitQuery,
) (*TrafficLimitResult, error) {
	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid traffic limit query", "error", err)
		return nil, err
	}

	sub, err := uc.subscriptionRepo.GetByID(ctx, query.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err)
		return nil, errors.NewNotFoundError("subscription not found")
	}

	plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan", "error", err)
		return nil, errors.NewNotFoundError("subscription plan not found")
	}

	trafficLimit, err := plan.GetTrafficLimit()
	if err != nil {
		uc.logger.Errorw("failed to get traffic limit", "error", err)
		return nil, errors.NewInternalError("failed to get traffic limit")
	}
	if trafficLimit == 0 {
		uc.logger.Debugw("no traffic limit configured",
			"subscription_id", query.SubscriptionID,
		)
		return &TrafficLimitResult{
			Exceeded:       false,
			TotalTraffic:   0,
			TrafficLimit:   0,
			RemainingBytes: 0,
			UsagePercent:   0,
		}, nil
	}

	// Resolve traffic period based on plan's reset mode (calendar_month or billing_cycle)
	period := subscription.ResolveTrafficPeriod(plan, sub)

	// Get traffic combining Redis (recent 24h) and MySQL stats (historical)
	totalTraffic, err := uc.getTotalUsageByResource(
		ctx,
		subscription.ResourceTypeNode.String(),
		query.NodeID,
		query.SubscriptionID,
		period.Start,
		period.End,
	)
	if err != nil {
		uc.logger.Errorw("failed to get total traffic", "error", err)
		return nil, errors.NewInternalError("failed to get traffic statistics")
	}

	exceeded := totalTraffic >= trafficLimit
	remainingBytes := uint64(0)
	if !exceeded {
		remainingBytes = trafficLimit - totalTraffic
	}

	usagePercent := float64(0)
	if trafficLimit > 0 {
		usagePercent = float64(totalTraffic) / float64(trafficLimit) * 100
	}

	result := &TrafficLimitResult{
		Exceeded:       exceeded,
		TotalTraffic:   totalTraffic,
		TrafficLimit:   trafficLimit,
		RemainingBytes: remainingBytes,
		UsagePercent:   usagePercent,
	}

	if exceeded {
		uc.logger.Warnw("traffic limit exceeded",
			"node_id", query.NodeID,
			"subscription_id", query.SubscriptionID,
			"total_traffic", totalTraffic,
			"limit", trafficLimit,
		)
	}

	return result, nil
}

func (uc *CheckTrafficLimitUseCase) validateQuery(query CheckTrafficLimitQuery) error {
	if query.NodeID == 0 {
		return errors.NewValidationError("node ID is required")
	}

	if query.SubscriptionID == 0 {
		return errors.NewValidationError("subscription ID is required")
	}

	return nil
}

// getTotalUsageByResource calculates total traffic for a resource by combining:
// - Last 24 hours: from Redis HourlyTrafficCache
// - Before 24 hours: from MySQL subscription_usage_stats table
func (uc *CheckTrafficLimitUseCase) getTotalUsageByResource(
	ctx context.Context,
	resourceType string,
	resourceID uint,
	subscriptionID uint,
	from, to time.Time,
) (uint64, error) {
	now := biztime.NowUTC()

	// Use start of yesterday's business day as batch/speed boundary (Lambda architecture)
	// MySQL: complete days before yesterday; Redis: yesterday + today (within 48h TTL)
	recentBoundary := biztime.StartOfDayUTC(now.AddDate(0, 0, -1))

	var total uint64

	// Determine time boundaries for Redis query (yesterday + today)
	recentFrom := from
	if recentFrom.Before(recentBoundary) {
		recentFrom = recentBoundary
	}
	recentTo := to
	if recentTo.After(now) {
		recentTo = now
	}

	// Get recent traffic from Redis (yesterday + today)
	if recentFrom.Before(recentTo) {
		hourlyPoints, err := uc.hourlyCache.GetHourlyTrafficRange(
			ctx, subscriptionID, resourceType, resourceID, recentFrom, recentTo,
		)
		if err != nil {
			// Log warning but don't fail - Redis unavailability shouldn't block limit checks
			uc.logger.Warnw("failed to get recent traffic from Redis, using stats only",
				"subscription_id", subscriptionID,
				"resource_type", resourceType,
				"resource_id", resourceID,
				"error", err,
			)
		} else {
			for _, point := range hourlyPoints {
				// Safe conversion: only add positive values to prevent uint64 overflow
				if point.Upload > 0 {
					total += uint64(point.Upload)
				}
				if point.Download > 0 {
					total += uint64(point.Download)
				}
			}
			uc.logger.Debugw("got recent traffic from Redis",
				"resource_type", resourceType,
				"resource_id", resourceID,
				"recent_total", total,
				"points_count", len(hourlyPoints),
			)
		}
	}

	// Get historical traffic from MySQL stats (complete days before yesterday)
	if from.Before(recentBoundary) {
		historicalTo := recentBoundary.Add(-time.Second)
		if historicalTo.After(to) {
			historicalTo = to
		}

		historicalTraffic, err := uc.usageStatsRepo.GetTotalByResourceID(
			ctx, resourceType, resourceID, subscription.GranularityDaily, from, historicalTo,
		)
		if err != nil {
			uc.logger.Warnw("failed to get historical traffic from stats, using Redis data only",
				"resource_type", resourceType,
				"resource_id", resourceID,
				"error", err,
			)
			// If we already have Redis data, return what we have
			// If we don't have any data from either source, that's ok - it means zero traffic
		} else if historicalTraffic != nil {
			total += historicalTraffic.Total
			uc.logger.Debugw("got historical traffic from MySQL stats",
				"resource_type", resourceType,
				"resource_id", resourceID,
				"historical_total", historicalTraffic.Total,
				"combined_total", total,
			)
		}
	}

	return total, nil
}
