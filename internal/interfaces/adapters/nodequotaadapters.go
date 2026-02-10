package adapters

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	nodeHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/node"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// NodeSubscriptionQuotaCacheAdapter adapts RedisSubscriptionQuotaCache to NodeSubscriptionQuotaCache interface
type NodeSubscriptionQuotaCacheAdapter struct {
	cache  cache.SubscriptionQuotaCache
	logger logger.Interface
}

// NewNodeSubscriptionQuotaCacheAdapter creates a new adapter
func NewNodeSubscriptionQuotaCacheAdapter(
	cache cache.SubscriptionQuotaCache,
	logger logger.Interface,
) *NodeSubscriptionQuotaCacheAdapter {
	return &NodeSubscriptionQuotaCacheAdapter{
		cache:  cache,
		logger: logger,
	}
}

// GetQuota retrieves subscription quota from cache
func (a *NodeSubscriptionQuotaCacheAdapter) GetQuota(ctx context.Context, subscriptionID uint) (*nodeHandlers.CachedQuotaInfo, error) {
	cached, err := a.cache.GetQuota(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if cached == nil {
		return nil, nil
	}

	return &nodeHandlers.CachedQuotaInfo{
		Limit:       cached.Limit,
		PeriodStart: cached.PeriodStart,
		PeriodEnd:   cached.PeriodEnd,
		PlanType:    cached.PlanType,
		Suspended:   cached.Suspended,
		NotFound:    cached.NotFound,
	}, nil
}

// MarkSuspended marks the subscription as suspended in cache
func (a *NodeSubscriptionQuotaCacheAdapter) MarkSuspended(ctx context.Context, subscriptionID uint) error {
	return a.cache.SetSuspended(ctx, subscriptionID, true)
}

// NodeSubscriptionQuotaLoaderAdapter adapts QuotaCacheSyncService to NodeSubscriptionQuotaLoader interface
type NodeSubscriptionQuotaLoaderAdapter struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	quotaCache       cache.SubscriptionQuotaCache
	logger           logger.Interface
}

// NewNodeSubscriptionQuotaLoaderAdapter creates a new adapter
func NewNodeSubscriptionQuotaLoaderAdapter(
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	quotaCache cache.SubscriptionQuotaCache,
	logger logger.Interface,
) *NodeSubscriptionQuotaLoaderAdapter {
	return &NodeSubscriptionQuotaLoaderAdapter{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		quotaCache:       quotaCache,
		logger:           logger,
	}
}

// LoadQuotaByID loads subscription quota from database and caches it.
// When subscription is not found or inactive, a null marker is cached to prevent
// repeated DB lookups (cache penetration protection).
func (a *NodeSubscriptionQuotaLoaderAdapter) LoadQuotaByID(ctx context.Context, subscriptionID uint) (*nodeHandlers.CachedQuotaInfo, error) {
	// Get subscription from database
	sub, err := a.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		a.setNullMarker(ctx, subscriptionID)
		return nil, nil
	}

	// Only cache active subscriptions
	if !sub.IsActive() {
		a.setNullMarker(ctx, subscriptionID)
		return nil, nil
	}

	// Get plan information
	plan, err := a.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		return nil, err
	}
	if plan == nil {
		a.setNullMarker(ctx, subscriptionID)
		return nil, nil
	}

	// Get traffic limit
	trafficLimit, err := plan.GetTrafficLimit()
	if err != nil {
		return nil, err
	}

	// Build cached quota object
	cachedQuota := &cache.CachedQuota{
		Limit:       int64(trafficLimit),
		PeriodStart: sub.CurrentPeriodStart(),
		PeriodEnd:   sub.CurrentPeriodEnd(),
		PlanType:    plan.PlanType().String(),
		Suspended:   false,
	}

	// Cache the quota (ignore error, still return the quota)
	if err := a.quotaCache.SetQuota(ctx, subscriptionID, cachedQuota); err != nil {
		a.logger.Warnw("failed to cache loaded quota",
			"subscription_id", subscriptionID,
			"error", err,
		)
	}

	return &nodeHandlers.CachedQuotaInfo{
		Limit:       cachedQuota.Limit,
		PeriodStart: cachedQuota.PeriodStart,
		PeriodEnd:   cachedQuota.PeriodEnd,
		PlanType:    cachedQuota.PlanType,
		Suspended:   cachedQuota.Suspended,
	}, nil
}

func (a *NodeSubscriptionQuotaLoaderAdapter) setNullMarker(ctx context.Context, subscriptionID uint) {
	if err := a.quotaCache.SetNullMarker(ctx, subscriptionID); err != nil {
		a.logger.Warnw("failed to set quota null marker",
			"subscription_id", subscriptionID,
			"error", err,
		)
	}
}

// NodeSubscriptionUsageReaderAdapter adapts traffic cache and stats to NodeSubscriptionUsageReader interface
type NodeSubscriptionUsageReaderAdapter struct {
	hourlyTrafficCache cache.HourlyTrafficCache
	usageStatsRepo     subscription.SubscriptionUsageStatsRepository
	logger             logger.Interface
}

// NewNodeSubscriptionUsageReaderAdapter creates a new adapter
func NewNodeSubscriptionUsageReaderAdapter(
	hourlyTrafficCache cache.HourlyTrafficCache,
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	logger logger.Interface,
) *NodeSubscriptionUsageReaderAdapter {
	return &NodeSubscriptionUsageReaderAdapter{
		hourlyTrafficCache: hourlyTrafficCache,
		usageStatsRepo:     usageStatsRepo,
		logger:             logger,
	}
}

// GetCurrentPeriodUsage returns the total usage for the current billing period
func (a *NodeSubscriptionUsageReaderAdapter) GetCurrentPeriodUsage(
	ctx context.Context,
	subscriptionID uint,
	periodStart, periodEnd time.Time,
) (int64, error) {
	now := biztime.NowUTC()

	// Use start of yesterday's business day as batch/speed boundary (Lambda architecture)
	// MySQL: complete days before yesterday; Redis: yesterday + today (within 48h TTL)
	recentBoundary := biztime.StartOfDayUTC(now.AddDate(0, 0, -1))

	var total int64

	// Get recent traffic from Redis (yesterday + today, filter by node type)
	resourceType := subscription.ResourceTypeNode.String()
	recentTraffic, err := a.hourlyTrafficCache.GetTotalTrafficBySubscriptionIDs(
		ctx, []uint{subscriptionID}, resourceType, recentBoundary, now,
	)
	if err != nil {
		a.logger.Warnw("failed to get recent traffic from Redis",
			"subscription_id", subscriptionID,
			"error", err,
		)
		// Continue with historical data only
	} else {
		for _, traffic := range recentTraffic {
			total += int64(traffic.Total)
		}
	}

	// Get historical traffic from MySQL (complete days before yesterday, within billing period)
	historicalStart := periodStart
	historicalEnd := recentBoundary.Add(-time.Second)
	if historicalEnd.Before(historicalStart) {
		// If billing period started after recentBoundary, no historical data needed
		return total, nil
	}

	historicalTraffic, err := a.usageStatsRepo.GetTotalBySubscriptionIDs(
		ctx, []uint{subscriptionID}, &resourceType, subscription.GranularityDaily, historicalStart, historicalEnd,
	)
	if err != nil {
		a.logger.Warnw("failed to get historical traffic from stats",
			"subscription_id", subscriptionID,
			"error", err,
		)
		// Return Redis data only
		return total, nil
	}

	if historicalTraffic != nil {
		total += int64(historicalTraffic.Total)
	}

	return total, nil
}
