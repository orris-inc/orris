package usecases

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetSubscriptionUsageStatsQuery represents the query parameters for subscription usage stats
type GetSubscriptionUsageStatsQuery struct {
	SubscriptionID uint
	From           time.Time
	To             time.Time
	Granularity    string // hour, day, month
	Page           int
	PageSize       int
}

// SubscriptionUsageStatsRecord represents a single usage stats record
type SubscriptionUsageStatsRecord struct {
	ResourceType string    `json:"resource_type"`
	ResourceSID  string    `json:"resource_id"` // Stripe-style SID (node_xxx or fwd_xxx)
	Upload       uint64    `json:"upload"`
	Download     uint64    `json:"download"`
	Total        uint64    `json:"total"`
	Period       time.Time `json:"period"`
}

// SubscriptionUsageSummary represents aggregated usage summary
type SubscriptionUsageSummary struct {
	TotalUpload   uint64 `json:"total_upload"`
	TotalDownload uint64 `json:"total_download"`
	Total         uint64 `json:"total"`
}

// GetSubscriptionUsageStatsResponse represents the response for subscription usage stats
type GetSubscriptionUsageStatsResponse struct {
	Records  []*SubscriptionUsageStatsRecord `json:"records"`
	Summary  *SubscriptionUsageSummary       `json:"summary"`
	Total    int                             `json:"total"`
	Page     int                             `json:"page"`
	PageSize int                             `json:"page_size"`
}

// maxHourlyDataHours is the maximum hours of hourly data available in Redis cache.
// Hourly data is only retained for approximately 24 hours before being aggregated.
const maxHourlyDataHours = 24

// GetSubscriptionUsageStatsUseCase handles retrieving usage statistics for a subscription
type GetSubscriptionUsageStatsUseCase struct {
	usageRepo       subscription.SubscriptionUsageRepository
	usageStatsRepo  subscription.SubscriptionUsageStatsRepository
	hourlyCache     cache.HourlyTrafficCache
	nodeRepo        node.NodeRepository
	forwardRuleRepo forward.Repository
	logger          logger.Interface
}

// NewGetSubscriptionUsageStatsUseCase creates a new GetSubscriptionUsageStatsUseCase
func NewGetSubscriptionUsageStatsUseCase(
	usageRepo subscription.SubscriptionUsageRepository,
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyCache cache.HourlyTrafficCache,
	nodeRepo node.NodeRepository,
	forwardRuleRepo forward.Repository,
	logger logger.Interface,
) *GetSubscriptionUsageStatsUseCase {
	return &GetSubscriptionUsageStatsUseCase{
		usageRepo:       usageRepo,
		usageStatsRepo:  usageStatsRepo,
		hourlyCache:     hourlyCache,
		nodeRepo:        nodeRepo,
		forwardRuleRepo: forwardRuleRepo,
		logger:          logger,
	}
}

// Execute retrieves usage statistics for a subscription
func (uc *GetSubscriptionUsageStatsUseCase) Execute(
	ctx context.Context,
	query GetSubscriptionUsageStatsQuery,
) (*GetSubscriptionUsageStatsResponse, error) {
	uc.logger.Infow("fetching subscription usage stats",
		"subscription_id", query.SubscriptionID,
		"from", query.From,
		"to", query.To,
		"granularity", query.Granularity,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid subscription usage stats query", "error", err)
		return nil, err
	}

	// Get total usage summary from Redis (recent 24h) + MySQL stats (historical)
	summary, err := uc.getTotalUsageBySubscriptionID(ctx, query.SubscriptionID, query.From, query.To)
	if err != nil {
		uc.logger.Errorw("failed to fetch subscription usage summary", "error", err)
		return nil, errors.NewInternalError("failed to fetch usage summary")
	}

	// If granularity is specified, use trend aggregation
	if query.Granularity != "" {
		return uc.executeWithTrendAggregation(ctx, query, summary)
	}

	// No granularity - return raw records
	return uc.executeWithRawRecords(ctx, query, summary)
}

// executeWithTrendAggregation fetches usage stats with time-based aggregation.
// Routes to different data sources based on granularity:
// - hour: Redis HourlyTrafficCache (last 24 hours only)
// - day/month: MySQL subscription_usage_stats table
func (uc *GetSubscriptionUsageStatsUseCase) executeWithTrendAggregation(
	ctx context.Context,
	query GetSubscriptionUsageStatsQuery,
	summary *SubscriptionUsageSummary,
) (*GetSubscriptionUsageStatsResponse, error) {
	var records []*SubscriptionUsageStatsRecord
	var err error

	switch query.Granularity {
	case "hour":
		records, err = uc.getHourlyTrendFromRedis(ctx, query)
	case "day":
		records, err = uc.getDailyTrendFromStats(ctx, query)
	case "month":
		records, err = uc.getMonthlyTrendFromStats(ctx, query)
	default:
		// Fallback to legacy MySQL query for backward compatibility
		records, err = uc.getLegacyTrend(ctx, query)
	}

	if err != nil {
		return nil, err
	}

	page := query.Page
	if page == 0 {
		page = constants.DefaultPage
	}
	pageSize := query.PageSize
	if pageSize == 0 {
		pageSize = constants.MaxPageSize
	}

	response := &GetSubscriptionUsageStatsResponse{
		Records:  records,
		Summary:  summary,
		Total:    len(records),
		Page:     page,
		PageSize: pageSize,
	}

	uc.logger.Infow("subscription usage stats with trend aggregation fetched successfully",
		"subscription_id", query.SubscriptionID,
		"granularity", query.Granularity,
		"count", len(records),
	)

	return response, nil
}

// isWithin24Hours checks if the given time is within the last 24 hours.
func isWithin24Hours(t time.Time) bool {
	return time.Since(t) <= maxHourlyDataHours*time.Hour
}

// hourlyRecordKey uniquely identifies an hourly traffic record by resource and time.
type hourlyRecordKey struct {
	resourceType string
	resourceID   uint
	hour         time.Time
}

// getHourlyTrendFromRedis retrieves hourly trend data from Redis HourlyTrafficCache.
// Only the last 24 hours of data is available.
func (uc *GetSubscriptionUsageStatsUseCase) getHourlyTrendFromRedis(
	ctx context.Context,
	query GetSubscriptionUsageStatsQuery,
) ([]*SubscriptionUsageStatsRecord, error) {
	// Validate time range - hourly data only available for last 24 hours
	if !isWithin24Hours(query.From) {
		uc.logger.Warnw("hourly data requested beyond 24-hour window",
			"subscription_id", query.SubscriptionID,
			"from", query.From,
			"hours_ago", time.Since(query.From).Hours(),
		)
		return nil, errors.NewValidationError("hourly data only available for the last 24 hours")
	}

	// Discover active resources for this subscription from recent daily stats
	resourceSet, err := uc.discoverActiveResources(ctx, query.SubscriptionID)
	if err != nil {
		uc.logger.Warnw("failed to discover active resources, falling back to legacy query",
			"subscription_id", query.SubscriptionID,
			"error", err,
		)
		return uc.getLegacyTrend(ctx, query)
	}

	// Fetch hourly data from Redis for each resource
	hourlyRecords := uc.fetchHourlyDataFromRedis(ctx, query, resourceSet)

	// Populate SIDs for all records
	uc.populateSIDsForHourlyRecords(ctx, hourlyRecords, resourceSet)

	// Convert map to slice
	records := make([]*SubscriptionUsageStatsRecord, 0, len(hourlyRecords))
	for _, record := range hourlyRecords {
		records = append(records, record)
	}

	uc.logger.Infow("hourly trend data fetched from Redis",
		"subscription_id", query.SubscriptionID,
		"from", query.From,
		"to", query.To,
		"resources_count", len(resourceSet),
		"records_count", len(records),
	)

	return records, nil
}

// resourceKey uniquely identifies a resource by type and ID.
type resourceKey struct {
	resourceType string
	resourceID   uint
}

// discoverActiveResources finds resources with recent traffic for a subscription.
// It looks back 48 hours to ensure we capture all active resources.
func (uc *GetSubscriptionUsageStatsUseCase) discoverActiveResources(
	ctx context.Context,
	subscriptionID uint,
) (map[resourceKey]struct{}, error) {
	now := biztime.NowUTC()
	lookbackStart := now.Add(-48 * time.Hour)

	dailyStats, err := uc.usageStatsRepo.GetBySubscriptionID(
		ctx, subscriptionID, subscription.GranularityDaily,
		lookbackStart, now,
	)
	if err != nil {
		return nil, err
	}

	resourceSet := make(map[resourceKey]struct{})
	for _, stat := range dailyStats {
		resourceSet[resourceKey{
			resourceType: stat.ResourceType(),
			resourceID:   stat.ResourceID(),
		}] = struct{}{}
	}

	return resourceSet, nil
}

// fetchHourlyDataFromRedis retrieves hourly traffic data from Redis for all resources.
func (uc *GetSubscriptionUsageStatsUseCase) fetchHourlyDataFromRedis(
	ctx context.Context,
	query GetSubscriptionUsageStatsQuery,
	resourceSet map[resourceKey]struct{},
) map[hourlyRecordKey]*SubscriptionUsageStatsRecord {
	hourlyRecords := make(map[hourlyRecordKey]*SubscriptionUsageStatsRecord)

	for rk := range resourceSet {
		hourlyData, err := uc.hourlyCache.GetHourlyTrafficRange(
			ctx, query.SubscriptionID, rk.resourceType, rk.resourceID,
			query.From, query.To,
		)
		if err != nil {
			uc.logger.Warnw("failed to get hourly traffic from Redis",
				"subscription_id", query.SubscriptionID,
				"resource_type", rk.resourceType,
				"resource_id", rk.resourceID,
				"error", err,
			)
			continue
		}

		for _, point := range hourlyData {
			hk := hourlyRecordKey{
				resourceType: rk.resourceType,
				resourceID:   rk.resourceID,
				hour:         point.Hour,
			}
			hourlyRecords[hk] = &SubscriptionUsageStatsRecord{
				ResourceType: rk.resourceType,
				ResourceSID:  "", // Will be filled by populateSIDsForHourlyRecords
				Upload:       uint64(point.Upload),
				Download:     uint64(point.Download),
				Total:        uint64(point.Upload + point.Download),
				Period:       point.Hour,
			}
		}
	}

	return hourlyRecords
}

// populateSIDsForHourlyRecords fills in the ResourceSID field for all hourly records.
func (uc *GetSubscriptionUsageStatsUseCase) populateSIDsForHourlyRecords(
	ctx context.Context,
	hourlyRecords map[hourlyRecordKey]*SubscriptionUsageStatsRecord,
	resourceSet map[resourceKey]struct{},
) {
	// Collect resource IDs for SID lookup
	nodeIDs := make([]uint, 0)
	forwardRuleIDs := make([]uint, 0)
	for rk := range resourceSet {
		switch subscription.ResourceType(rk.resourceType) {
		case subscription.ResourceTypeNode:
			nodeIDs = append(nodeIDs, rk.resourceID)
		case subscription.ResourceTypeForwardRule:
			forwardRuleIDs = append(forwardRuleIDs, rk.resourceID)
		}
	}

	// Batch fetch SID maps
	nodeSIDMap, forwardRuleSIDMap := uc.fetchSIDMaps(ctx, nodeIDs, forwardRuleIDs)

	// Fill in SIDs using the hourlyRecordKey which contains resourceID
	for hk, record := range hourlyRecords {
		switch subscription.ResourceType(hk.resourceType) {
		case subscription.ResourceTypeNode:
			record.ResourceSID = nodeSIDMap[hk.resourceID]
		case subscription.ResourceTypeForwardRule:
			record.ResourceSID = forwardRuleSIDMap[hk.resourceID]
		}
	}
}

// getDailyTrendFromStats retrieves daily trend data from MySQL subscription_usage_stats table.
func (uc *GetSubscriptionUsageStatsUseCase) getDailyTrendFromStats(
	ctx context.Context,
	query GetSubscriptionUsageStatsQuery,
) ([]*SubscriptionUsageStatsRecord, error) {
	stats, err := uc.usageStatsRepo.GetBySubscriptionID(
		ctx, query.SubscriptionID, subscription.GranularityDaily,
		query.From, query.To,
	)
	if err != nil {
		uc.logger.Errorw("failed to fetch daily stats from subscription_usage_stats",
			"subscription_id", query.SubscriptionID,
			"error", err,
		)
		return nil, errors.NewInternalError("failed to fetch daily usage statistics")
	}

	return uc.convertStatsToRecords(ctx, stats)
}

// getMonthlyTrendFromStats retrieves monthly trend data from MySQL subscription_usage_stats table.
func (uc *GetSubscriptionUsageStatsUseCase) getMonthlyTrendFromStats(
	ctx context.Context,
	query GetSubscriptionUsageStatsQuery,
) ([]*SubscriptionUsageStatsRecord, error) {
	stats, err := uc.usageStatsRepo.GetBySubscriptionID(
		ctx, query.SubscriptionID, subscription.GranularityMonthly,
		query.From, query.To,
	)
	if err != nil {
		uc.logger.Errorw("failed to fetch monthly stats from subscription_usage_stats",
			"subscription_id", query.SubscriptionID,
			"error", err,
		)
		return nil, errors.NewInternalError("failed to fetch monthly usage statistics")
	}

	return uc.convertStatsToRecords(ctx, stats)
}

// getTotalUsageBySubscriptionID combines recent traffic from Redis with historical from MySQL stats.
// For data within the last 24 hours, it queries Redis HourlyTrafficCache.
// For data older than 24 hours, it queries MySQL subscription_usage_stats table.
func (uc *GetSubscriptionUsageStatsUseCase) getTotalUsageBySubscriptionID(
	ctx context.Context,
	subscriptionID uint,
	from, to time.Time,
) (*SubscriptionUsageSummary, error) {
	now := biztime.NowUTC()
	dayAgo := now.Add(-24 * time.Hour)

	var totalUpload, totalDownload, total uint64

	// Determine time boundaries for recent data (Redis)
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
		// Use hourlyCache to get subscription traffic
		recentTraffic, err := uc.hourlyCache.GetTotalTrafficBySubscriptionIDs(
			ctx, []uint{subscriptionID}, "", recentFrom, recentTo,
		)
		if err != nil {
			uc.logger.Warnw("failed to get recent traffic from Redis",
				"subscription_id", subscriptionID,
				"from", recentFrom,
				"to", recentTo,
				"error", err,
			)
			// Continue with historical data even if Redis fails
		} else if t, ok := recentTraffic[subscriptionID]; ok {
			totalUpload += t.Upload
			totalDownload += t.Download
			total += t.Total
		}
	}

	// Get historical traffic from MySQL stats (before 24h ago)
	if from.Before(dayAgo) {
		historicalTo := dayAgo
		if historicalTo.After(to) {
			historicalTo = to
		}
		// Use daily granularity for historical aggregation
		historicalStats, err := uc.usageStatsRepo.GetTotalBySubscriptionIDs(
			ctx, []uint{subscriptionID}, nil, subscription.GranularityDaily, from, historicalTo,
		)
		if err != nil {
			uc.logger.Warnw("failed to get historical traffic from stats",
				"subscription_id", subscriptionID,
				"from", from,
				"to", historicalTo,
				"error", err,
			)
			// Continue with whatever data we have
		} else if historicalStats != nil {
			totalUpload += historicalStats.Upload
			totalDownload += historicalStats.Download
			total += historicalStats.Total
		}
	}

	return &SubscriptionUsageSummary{
		TotalUpload:   totalUpload,
		TotalDownload: totalDownload,
		Total:         total,
	}, nil
}

// getLegacyTrend retrieves trend data from legacy subscription_usages table.
// DEPRECATED: This method queries the old subscription_usages table and is only used
// as a fallback when active resource discovery fails. Prefer using the stats-based methods.
func (uc *GetSubscriptionUsageStatsUseCase) getLegacyTrend(
	ctx context.Context,
	query GetSubscriptionUsageStatsQuery,
) ([]*SubscriptionUsageStatsRecord, error) {
	uc.logger.Warnw("using deprecated legacy trend query on subscription_usages table",
		"subscription_id", query.SubscriptionID,
		"from", query.From,
		"to", query.To,
		"granularity", query.Granularity,
	)
	trendPoints, err := uc.usageRepo.GetSubscriptionUsageTrend(ctx, query.SubscriptionID, query.From, query.To, query.Granularity)
	if err != nil {
		uc.logger.Errorw("failed to fetch subscription usage trend from legacy table", "error", err)
		return nil, errors.NewInternalError("failed to fetch usage statistics")
	}

	// Collect resource IDs by type for batch lookup
	nodeIDs := make([]uint, 0)
	forwardRuleIDs := make([]uint, 0)
	for _, point := range trendPoints {
		switch subscription.ResourceType(point.ResourceType) {
		case subscription.ResourceTypeNode:
			nodeIDs = append(nodeIDs, point.ResourceID)
		case subscription.ResourceTypeForwardRule:
			forwardRuleIDs = append(forwardRuleIDs, point.ResourceID)
		}
	}

	// Batch fetch SID maps
	nodeSIDMap, forwardRuleSIDMap := uc.fetchSIDMaps(ctx, nodeIDs, forwardRuleIDs)

	// Convert trend points to response format
	records := make([]*SubscriptionUsageStatsRecord, 0, len(trendPoints))
	for _, point := range trendPoints {
		var resourceSID string
		switch subscription.ResourceType(point.ResourceType) {
		case subscription.ResourceTypeNode:
			resourceSID = nodeSIDMap[point.ResourceID]
		case subscription.ResourceTypeForwardRule:
			resourceSID = forwardRuleSIDMap[point.ResourceID]
		}

		records = append(records, &SubscriptionUsageStatsRecord{
			ResourceType: point.ResourceType,
			ResourceSID:  resourceSID,
			Upload:       point.Upload,
			Download:     point.Download,
			Total:        point.Total,
			Period:       point.Period,
		})
	}

	return records, nil
}

// convertStatsToRecords converts SubscriptionUsageStats entities to SubscriptionUsageStatsRecord response format.
func (uc *GetSubscriptionUsageStatsUseCase) convertStatsToRecords(
	ctx context.Context,
	stats []*subscription.SubscriptionUsageStats,
) ([]*SubscriptionUsageStatsRecord, error) {
	// Collect resource IDs by type for batch lookup
	nodeIDs := make([]uint, 0)
	forwardRuleIDs := make([]uint, 0)
	for _, stat := range stats {
		switch subscription.ResourceType(stat.ResourceType()) {
		case subscription.ResourceTypeNode:
			nodeIDs = append(nodeIDs, stat.ResourceID())
		case subscription.ResourceTypeForwardRule:
			forwardRuleIDs = append(forwardRuleIDs, stat.ResourceID())
		}
	}

	// Batch fetch SID maps
	nodeSIDMap, forwardRuleSIDMap := uc.fetchSIDMaps(ctx, nodeIDs, forwardRuleIDs)

	// Convert stats to response format
	records := make([]*SubscriptionUsageStatsRecord, 0, len(stats))
	for _, stat := range stats {
		var resourceSID string
		switch subscription.ResourceType(stat.ResourceType()) {
		case subscription.ResourceTypeNode:
			resourceSID = nodeSIDMap[stat.ResourceID()]
		case subscription.ResourceTypeForwardRule:
			resourceSID = forwardRuleSIDMap[stat.ResourceID()]
		}

		records = append(records, &SubscriptionUsageStatsRecord{
			ResourceType: stat.ResourceType(),
			ResourceSID:  resourceSID,
			Upload:       stat.Upload(),
			Download:     stat.Download(),
			Total:        stat.Total(),
			Period:       stat.Period(),
		})
	}

	return records, nil
}

// executeWithRawRecords fetches raw usage records without aggregation.
// DEPRECATED: This method queries the old subscription_usages table for raw records.
// Raw record queries are being phased out in favor of aggregated stats from Redis + MySQL stats table.
func (uc *GetSubscriptionUsageStatsUseCase) executeWithRawRecords(
	ctx context.Context,
	query GetSubscriptionUsageStatsQuery,
	summary *SubscriptionUsageSummary,
) (*GetSubscriptionUsageStatsResponse, error) {
	uc.logger.Warnw("using deprecated raw records query on subscription_usages table",
		"subscription_id", query.SubscriptionID,
		"from", query.From,
		"to", query.To,
	)
	filter := uc.buildFilter(query)

	usageRecords, err := uc.usageRepo.GetUsageStats(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to fetch subscription usage stats", "error", err)
		return nil, errors.NewInternalError("failed to fetch usage statistics")
	}

	// Collect resource IDs by type for batch lookup
	nodeIDs := make([]uint, 0)
	forwardRuleIDs := make([]uint, 0)
	for _, record := range usageRecords {
		switch subscription.ResourceType(record.ResourceType()) {
		case subscription.ResourceTypeNode:
			nodeIDs = append(nodeIDs, record.ResourceID())
		case subscription.ResourceTypeForwardRule:
			forwardRuleIDs = append(forwardRuleIDs, record.ResourceID())
		}
	}

	// Batch fetch SID maps
	nodeSIDMap, forwardRuleSIDMap := uc.fetchSIDMaps(ctx, nodeIDs, forwardRuleIDs)

	// Convert records to response format
	records := make([]*SubscriptionUsageStatsRecord, 0, len(usageRecords))

	for _, record := range usageRecords {
		// Get resource SID based on type
		var resourceSID string
		switch subscription.ResourceType(record.ResourceType()) {
		case subscription.ResourceTypeNode:
			resourceSID = nodeSIDMap[record.ResourceID()]
		case subscription.ResourceTypeForwardRule:
			resourceSID = forwardRuleSIDMap[record.ResourceID()]
		}

		records = append(records, &SubscriptionUsageStatsRecord{
			ResourceType: record.ResourceType(),
			ResourceSID:  resourceSID,
			Upload:       record.Upload(),
			Download:     record.Download(),
			Total:        record.Total(),
			Period:       record.Period(),
		})
	}

	response := &GetSubscriptionUsageStatsResponse{
		Records:  records,
		Summary:  summary,
		Total:    len(records),
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}

	uc.logger.Infow("subscription usage stats fetched successfully",
		"subscription_id", query.SubscriptionID,
		"count", len(records),
	)

	return response, nil
}

// fetchSIDMaps fetches SID mappings for nodes and forward rules using batch queries
func (uc *GetSubscriptionUsageStatsUseCase) fetchSIDMaps(
	ctx context.Context,
	nodeIDs []uint,
	forwardRuleIDs []uint,
) (map[uint]string, map[uint]string) {
	// Deduplicate node IDs
	nodeSIDMap := make(map[uint]string)
	if len(nodeIDs) > 0 {
		uniqueNodeIDs := deduplicateUintSlice(nodeIDs)
		nodes, err := uc.nodeRepo.GetByIDs(ctx, uniqueNodeIDs)
		if err != nil {
			uc.logger.Warnw("failed to fetch nodes for SID lookup", "error", err)
		} else {
			for _, n := range nodes {
				nodeSIDMap[n.ID()] = n.SID()
			}
		}
	}

	// Deduplicate forward rule IDs and use batch query
	forwardRuleSIDMap := make(map[uint]string)
	if len(forwardRuleIDs) > 0 {
		uniqueRuleIDs := deduplicateUintSlice(forwardRuleIDs)
		rules, err := uc.forwardRuleRepo.GetByIDs(ctx, uniqueRuleIDs)
		if err != nil {
			uc.logger.Warnw("failed to fetch forward rules for SID lookup", "error", err)
		} else {
			for id, rule := range rules {
				forwardRuleSIDMap[id] = rule.SID()
			}
		}
	}

	return nodeSIDMap, forwardRuleSIDMap
}

// deduplicateUintSlice removes duplicate values from a uint slice
func deduplicateUintSlice(slice []uint) []uint {
	seen := make(map[uint]struct{}, len(slice))
	result := make([]uint, 0, len(slice))
	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

func (uc *GetSubscriptionUsageStatsUseCase) validateQuery(query GetSubscriptionUsageStatsQuery) error {
	if query.SubscriptionID == 0 {
		return errors.NewValidationError("subscription_id is required")
	}

	if query.From.IsZero() {
		return errors.NewValidationError("from time is required")
	}

	if query.To.IsZero() {
		return errors.NewValidationError("to time is required")
	}

	if query.To.Before(query.From) {
		return errors.NewValidationError("to time must be after from time")
	}

	if query.Granularity != "" &&
		query.Granularity != "hour" &&
		query.Granularity != "day" &&
		query.Granularity != "month" {
		return errors.NewValidationError("granularity must be one of: hour, day, month")
	}

	if query.Page < 0 {
		return errors.NewValidationError("page must be non-negative")
	}

	if query.PageSize < 0 {
		return errors.NewValidationError("page_size must be non-negative")
	}

	return nil
}

func (uc *GetSubscriptionUsageStatsUseCase) buildFilter(query GetSubscriptionUsageStatsQuery) subscription.UsageStatsFilter {
	page := query.Page
	if page == 0 {
		page = constants.DefaultPage
	}

	pageSize := query.PageSize
	if pageSize == 0 {
		pageSize = constants.MaxPageSize
	}

	filter := subscription.UsageStatsFilter{
		SubscriptionID: &query.SubscriptionID,
		From:           query.From,
		To:             query.To,
	}
	filter.Page = page
	filter.PageSize = pageSize

	if query.Granularity != "" {
		filter.Period = &query.Granularity
	}

	return filter
}
