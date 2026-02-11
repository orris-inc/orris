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
// Hourly data is retained for approximately 48 hours before being aggregated.
const maxHourlyDataHours = 48

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
	uc.logger.Debugw("fetching subscription usage stats",
		"subscription_id", query.SubscriptionID,
		"from", query.From,
		"to", query.To,
		"granularity", query.Granularity,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Warnw("invalid subscription usage stats query", "error", err)
		return nil, err
	}

	// No granularity - default to daily granularity
	if query.Granularity == "" {
		query.Granularity = "day"
	}

	// Get records first, then calculate summary from records to ensure consistency
	return uc.executeWithTrendAggregation(ctx, query)
}

// executeWithTrendAggregation fetches usage stats with time-based aggregation.
// Routes to different data sources based on granularity:
// - hour: Redis HourlyTrafficCache (last 48 hours only)
// - day/month: MySQL subscription_usage_stats table
// Summary is calculated from records to ensure data consistency.
func (uc *GetSubscriptionUsageStatsUseCase) executeWithTrendAggregation(
	ctx context.Context,
	query GetSubscriptionUsageStatsQuery,
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
		// Unknown granularity - return validation error instead of falling back to legacy table
		return nil, errors.NewValidationError("granularity must be one of: hour, day, month")
	}

	if err != nil {
		return nil, err
	}

	// Calculate summary from records to ensure consistency
	// This avoids the issue where summary uses active sets but records use direct key lookup
	summary := uc.calculateSummaryFromRecords(records)

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

	uc.logger.Debugw("subscription usage stats with trend aggregation fetched successfully",
		"subscription_id", query.SubscriptionID,
		"granularity", query.Granularity,
		"count", len(records),
		"summary_total", summary.Total,
	)

	return response, nil
}

// calculateSummaryFromRecords aggregates all records into a summary.
// This ensures summary and records are always consistent since they use the same data source.
func (uc *GetSubscriptionUsageStatsUseCase) calculateSummaryFromRecords(
	records []*SubscriptionUsageStatsRecord,
) *SubscriptionUsageSummary {
	summary := &SubscriptionUsageSummary{}
	for _, record := range records {
		summary.TotalUpload += record.Upload
		summary.TotalDownload += record.Download
		summary.Total += record.Total
	}
	return summary
}

// hourlyRecordKey uniquely identifies an hourly traffic record by resource and time.
type hourlyRecordKey struct {
	resourceType string
	resourceID   uint
	hour         time.Time
}

// getHourlyTrendFromRedis retrieves hourly trend data from Redis HourlyTrafficCache.
// Only the last 48 hours of data is available.
func (uc *GetSubscriptionUsageStatsUseCase) getHourlyTrendFromRedis(
	ctx context.Context,
	query GetSubscriptionUsageStatsQuery,
) ([]*SubscriptionUsageStatsRecord, error) {
	// Adjust time range if from is beyond retention window
	now := biztime.NowUTC()
	retentionBoundary := now.Add(-maxHourlyDataHours * time.Hour)
	adjustedFrom := query.From
	if adjustedFrom.Before(retentionBoundary) {
		uc.logger.Debugw("hourly data requested beyond retention window, adjusting from time",
			"subscription_id", query.SubscriptionID,
			"original_from", query.From,
			"adjusted_from", retentionBoundary,
			"hours_ago", time.Since(query.From).Hours(),
			"max_hours", maxHourlyDataHours,
		)
		adjustedFrom = retentionBoundary
	}

	// Update query with adjusted from time
	query.From = adjustedFrom

	// Discover active resources for this subscription from recent daily stats
	resourceSet, err := uc.discoverActiveResources(ctx, query.SubscriptionID)
	if err != nil {
		uc.logger.Warnw("failed to discover active resources, returning empty result",
			"subscription_id", query.SubscriptionID,
			"error", err,
		)
		// Return empty result instead of falling back to legacy table
		return []*SubscriptionUsageStatsRecord{}, nil
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

	uc.logger.Debugw("hourly trend data fetched from Redis",
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
// It discovers resources directly from Redis hourly cache (last 48 hours).
func (uc *GetSubscriptionUsageStatsUseCase) discoverActiveResources(
	ctx context.Context,
	subscriptionID uint,
) (map[resourceKey]struct{}, error) {
	now := biztime.NowUTC()
	lookbackStart := now.Add(-maxHourlyDataHours * time.Hour)

	resourceSet := make(map[resourceKey]struct{})

	// Discover from Redis hourly cache (real-time data within 48 hours)
	hourlyData, err := uc.hourlyCache.GetAllHourlyTrafficBatch(ctx, lookbackStart, now)
	if err != nil {
		uc.logger.Warnw("failed to get hourly traffic batch for resource discovery",
			"subscription_id", subscriptionID,
			"error", err,
		)
		return resourceSet, nil
	}

	for _, data := range hourlyData {
		if data.SubscriptionID == subscriptionID {
			resourceSet[resourceKey{
				resourceType: data.ResourceType,
				resourceID:   data.ResourceID,
			}] = struct{}{}
		}
	}

	uc.logger.Debugw("discovered active resources for subscription from Redis",
		"subscription_id", subscriptionID,
		"resource_count", len(resourceSet),
	)

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
			// Safe conversion: treat negative int64 values as 0 to prevent uint64 underflow
			var upload, download uint64
			if point.Upload > 0 {
				upload = uint64(point.Upload)
			}
			if point.Download > 0 {
				download = uint64(point.Download)
			}
			hourlyRecords[hk] = &SubscriptionUsageStatsRecord{
				ResourceType: rk.resourceType,
				ResourceSID:  "", // Will be filled by populateSIDsForHourlyRecords
				Upload:       upload,
				Download:     download,
				Total:        upload + download,
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

// dailyRecordKey uniquely identifies a daily traffic record by resource and day.
type dailyRecordKey struct {
	resourceType string
	resourceID   uint
	day          time.Time
}

// getDailyTrendFromStats retrieves daily trend data by combining:
// - Redis HourlyTrafficCache for recent data (last 48 hours)
// - MySQL subscription_usage_stats table for historical data
func (uc *GetSubscriptionUsageStatsUseCase) getDailyTrendFromStats(
	ctx context.Context,
	query GetSubscriptionUsageStatsQuery,
) ([]*SubscriptionUsageStatsRecord, error) {
	now := biztime.NowUTC()
	retentionBoundary := now.Add(-maxHourlyDataHours * time.Hour)

	// Determine if query overlaps with Redis data window (last 48 hours)
	includesRedisWindow := !query.To.Before(retentionBoundary)
	includesHistory := query.From.Before(retentionBoundary)

	var records []*SubscriptionUsageStatsRecord

	// Step 1: Get recent data from Redis (if query overlaps with Redis window)
	if includesRedisWindow {
		redisFrom := query.From
		if redisFrom.Before(retentionBoundary) {
			redisFrom = retentionBoundary
		}

		redisRecords, err := uc.getDailyTrendFromRedis(ctx, query.SubscriptionID, redisFrom, query.To)
		if err != nil {
			uc.logger.Warnw("failed to get daily trend from Redis, continuing with MySQL only",
				"subscription_id", query.SubscriptionID,
				"error", err,
			)
		} else {
			records = append(records, redisRecords...)
		}
	}

	// Step 2: Get historical data from MySQL (if query includes data before Redis window)
	if includesHistory {
		mysqlTo := query.To
		if includesRedisWindow {
			// Exclude Redis window from MySQL query to avoid double counting
			mysqlTo = retentionBoundary.Add(-time.Nanosecond)
		}

		stats, err := uc.usageStatsRepo.GetBySubscriptionID(
			ctx, query.SubscriptionID, subscription.GranularityDaily,
			query.From, mysqlTo,
		)
		if err != nil {
			uc.logger.Errorw("failed to fetch daily stats from subscription_usage_stats",
				"subscription_id", query.SubscriptionID,
				"error", err,
			)
			return nil, errors.NewInternalError("failed to fetch daily usage statistics")
		}

		// Convert MySQL stats to records
		mysqlRecords, err := uc.convertStatsToRecords(ctx, stats)
		if err != nil {
			return nil, err
		}
		records = append(records, mysqlRecords...)
	}

	uc.logger.Debugw("daily trend data fetched from Redis + MySQL",
		"subscription_id", query.SubscriptionID,
		"from", query.From,
		"to", query.To,
		"includes_redis", includesRedisWindow,
		"includes_history", includesHistory,
		"records_count", len(records),
	)

	return records, nil
}

// getDailyTrendFromRedis aggregates hourly data from Redis into daily records.
func (uc *GetSubscriptionUsageStatsUseCase) getDailyTrendFromRedis(
	ctx context.Context,
	subscriptionID uint,
	from, to time.Time,
) ([]*SubscriptionUsageStatsRecord, error) {
	// Discover active resources for this subscription
	resourceSet, err := uc.discoverActiveResources(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	if len(resourceSet) == 0 {
		return []*SubscriptionUsageStatsRecord{}, nil
	}

	// Aggregate hourly data into daily records
	// Use dailyRecordKey to track resourceID for later SID lookup
	dailyRecords := make(map[dailyRecordKey]*SubscriptionUsageStatsRecord)

	for rk := range resourceSet {
		hourlyData, err := uc.hourlyCache.GetHourlyTrafficRange(
			ctx, subscriptionID, rk.resourceType, rk.resourceID,
			from, to,
		)
		if err != nil {
			uc.logger.Warnw("failed to get hourly traffic from Redis",
				"subscription_id", subscriptionID,
				"resource_type", rk.resourceType,
				"resource_id", rk.resourceID,
				"error", err,
			)
			continue
		}

		for _, point := range hourlyData {
			// Aggregate by day (start of day in business timezone, stored as UTC)
			dayStart := biztime.StartOfDayUTC(point.Hour)
			key := dailyRecordKey{
				resourceType: rk.resourceType,
				resourceID:   rk.resourceID,
				day:          dayStart,
			}

			// Safe conversion: treat negative int64 values as 0
			var upload, download uint64
			if point.Upload > 0 {
				upload = uint64(point.Upload)
			}
			if point.Download > 0 {
				download = uint64(point.Download)
			}

			if existing, ok := dailyRecords[key]; ok {
				existing.Upload += upload
				existing.Download += download
				existing.Total += upload + download
			} else {
				dailyRecords[key] = &SubscriptionUsageStatsRecord{
					ResourceType: rk.resourceType,
					ResourceSID:  "", // Will be filled below
					Upload:       upload,
					Download:     download,
					Total:        upload + download,
					Period:       dayStart,
				}
			}
		}
	}

	// Collect resource IDs for SID lookup
	nodeIDs := make([]uint, 0)
	forwardRuleIDs := make([]uint, 0)
	for key := range dailyRecords {
		switch subscription.ResourceType(key.resourceType) {
		case subscription.ResourceTypeNode:
			nodeIDs = append(nodeIDs, key.resourceID)
		case subscription.ResourceTypeForwardRule:
			forwardRuleIDs = append(forwardRuleIDs, key.resourceID)
		}
	}

	// Batch fetch SID maps
	nodeSIDMap, forwardRuleSIDMap := uc.fetchSIDMaps(ctx, nodeIDs, forwardRuleIDs)

	// Fill in SIDs and convert to slice
	records := make([]*SubscriptionUsageStatsRecord, 0, len(dailyRecords))
	for key, record := range dailyRecords {
		switch subscription.ResourceType(key.resourceType) {
		case subscription.ResourceTypeNode:
			record.ResourceSID = nodeSIDMap[key.resourceID]
		case subscription.ResourceTypeForwardRule:
			record.ResourceSID = forwardRuleSIDMap[key.resourceID]
		}
		records = append(records, record)
	}

	return records, nil
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
