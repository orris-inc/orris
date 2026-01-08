package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// maxAggregationPageSize is the maximum number of records to fetch in a single query
	// during usage aggregation.
	maxAggregationPageSize = 100000
)

// aggregationKey is used to group usage records by subscription, resource type, and resource ID.
type aggregationKey struct {
	subscriptionID uint
	resourceType   string
	resourceID     uint
}

// AggregateUsageUseCase handles aggregating subscription usage data from
// raw hourly data (Redis hourly buckets or subscription_usages fallback) to aggregated stats (subscription_usage_stats).
type AggregateUsageUseCase struct {
	usageRepo      subscription.SubscriptionUsageRepository
	usageStatsRepo subscription.SubscriptionUsageStatsRepository
	hourlyCache    cache.HourlyTrafficCache
	logger         logger.Interface
}

// NewAggregateUsageUseCase creates a new aggregate usage use case instance.
func NewAggregateUsageUseCase(
	usageRepo subscription.SubscriptionUsageRepository,
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyCache cache.HourlyTrafficCache,
	logger logger.Interface,
) *AggregateUsageUseCase {
	return &AggregateUsageUseCase{
		usageRepo:      usageRepo,
		usageStatsRepo: usageStatsRepo,
		hourlyCache:    hourlyCache,
		logger:         logger,
	}
}

// AggregateDailyUsage aggregates hourly data from yesterday into daily stats.
// It reads from Redis hourly buckets and writes to subscription_usage_stats table.
// After successful aggregation, it cleans up the Redis data for processed hours.
func (uc *AggregateUsageUseCase) AggregateDailyUsage(ctx context.Context) error {
	// Calculate yesterday in business timezone
	now := biztime.NowUTC()
	bizNow := biztime.ToBizTimezone(now)
	yesterday := bizNow.AddDate(0, 0, -1)

	// Get start of yesterday in business timezone (00:00)
	startOfDay := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, biztime.Location())
	startUTC := startOfDay.UTC()

	uc.logger.Infow("starting daily usage aggregation from Redis hourly buckets",
		"date", yesterday.Format("2006-01-02"),
		"start_utc", startUTC,
	)

	// Aggregate traffic from all 24 hours of yesterday
	aggregated := make(map[aggregationKey]*subscription.SubscriptionUsageStats)
	totalRecords := 0
	hoursProcessed := 0

	for hour := 0; hour < 24; hour++ {
		// Calculate hour time in business timezone, then convert to UTC for Redis lookup
		hourTime := startOfDay.Add(time.Duration(hour) * time.Hour)

		// Get all traffic data for this hour from Redis
		hourlyData, err := uc.hourlyCache.GetAllHourlyTraffic(ctx, hourTime)
		if err != nil {
			uc.logger.Warnw("failed to get hourly traffic from Redis, skipping hour",
				"hour", hourTime.Format("2006-01-02 15:04"),
				"error", err,
			)
			continue
		}

		if len(hourlyData) == 0 {
			continue
		}

		hoursProcessed++
		totalRecords += len(hourlyData)

		// Aggregate hourly data into daily stats
		uc.aggregateHourlyData(hourlyData, aggregated, startUTC)
	}

	if totalRecords == 0 {
		uc.logger.Infow("no hourly traffic data found in Redis for aggregation",
			"date", yesterday.Format("2006-01-02"),
		)
		return nil
	}

	// Upsert aggregated records
	successCount, errorCount := uc.upsertAggregatedStats(ctx, aggregated)

	uc.logger.Infow("daily usage aggregation completed",
		"date", yesterday.Format("2006-01-02"),
		"hours_processed", hoursProcessed,
		"total_records", totalRecords,
		"aggregated_groups", len(aggregated),
		"success_count", successCount,
		"error_count", errorCount,
	)

	// NOTE: Do NOT clean up Redis data immediately after aggregation.
	// Redis hourly data has 49-hour TTL and will expire naturally.
	// This allows users to query hourly data for the past 48 hours.
	// Previously we called cleanupProcessedHours() here, which caused
	// hourly data to be unavailable before the 48-hour window expired.

	if errorCount > 0 {
		return fmt.Errorf("daily aggregation completed with %d errors", errorCount)
	}

	return nil
}

// aggregateHourlyData aggregates HourlyTrafficData from Redis into the aggregated map.
func (uc *AggregateUsageUseCase) aggregateHourlyData(
	data []cache.HourlyTrafficData,
	aggregated map[aggregationKey]*subscription.SubscriptionUsageStats,
	period time.Time,
) {
	for _, d := range data {
		key := aggregationKey{
			subscriptionID: d.SubscriptionID,
			resourceType:   d.ResourceType,
			resourceID:     d.ResourceID,
		}

		if _, exists := aggregated[key]; !exists {
			var subscriptionIDPtr *uint
			if d.SubscriptionID != 0 {
				subscriptionIDPtr = &d.SubscriptionID
			}

			stats, err := subscription.NewSubscriptionUsageStats(
				d.ResourceType,
				d.ResourceID,
				subscriptionIDPtr,
				subscription.GranularityDaily,
				period,
			)
			if err != nil {
				uc.logger.Errorw("failed to create usage stats entity",
					"error", err,
					"resource_type", d.ResourceType,
					"resource_id", d.ResourceID,
				)
				continue
			}
			aggregated[key] = stats
		}

		// Accumulate traffic (convert int64 to uint64, negative values become 0)
		var upload, download uint64
		if d.Upload > 0 {
			upload = uint64(d.Upload)
		}
		if d.Download > 0 {
			download = uint64(d.Download)
		}
		if err := aggregated[key].Accumulate(upload, download); err != nil {
			uc.logger.Warnw("failed to accumulate traffic",
				"error", err,
				"resource_type", d.ResourceType,
				"resource_id", d.ResourceID,
			)
		}
	}
}

// CleanupOldUsageData deletes raw usage records older than the specified retention days.
// This helps manage storage by removing historical data that has already been aggregated.
func (uc *AggregateUsageUseCase) CleanupOldUsageData(ctx context.Context, retentionDays int) error {
	// Calculate cutoff time based on retention days
	cutoffTime := biztime.NowUTC().AddDate(0, 0, -retentionDays)

	uc.logger.Infow("starting cleanup of old usage data",
		"retention_days", retentionDays,
		"cutoff_time", cutoffTime,
	)

	// Delete old records using the repository method
	if err := uc.usageRepo.DeleteOldRecords(ctx, cutoffTime); err != nil {
		uc.logger.Errorw("failed to cleanup old usage data",
			"error", err,
			"retention_days", retentionDays,
			"cutoff_time", cutoffTime,
		)
		return fmt.Errorf("failed to cleanup old usage data: %w", err)
	}

	uc.logger.Infow("old usage data cleanup completed successfully",
		"retention_days", retentionDays,
		"cutoff_time", cutoffTime,
	)

	return nil
}

// AggregateMonthlyUsage aggregates daily data from last month into monthly stats.
// It reads from subscription_usage_stats table (daily granularity) and writes
// monthly aggregated records to the same table.
func (uc *AggregateUsageUseCase) AggregateMonthlyUsage(ctx context.Context) error {
	// Calculate last month in business timezone
	now := biztime.NowUTC()
	bizNow := biztime.ToBizTimezone(now)

	// Get first day of current month, then go back one month
	firstDayOfCurrentMonth := time.Date(bizNow.Year(), bizNow.Month(), 1, 0, 0, 0, 0, biztime.Location())
	lastMonth := firstDayOfCurrentMonth.AddDate(0, -1, 0)
	year := lastMonth.Year()
	month := lastMonth.Month()

	// Get UTC time range for last month in business timezone
	startUTC := biztime.StartOfMonthUTC(year, month)
	endUTC := startUTC.AddDate(0, 1, 0)

	uc.logger.Infow("starting monthly usage aggregation from daily stats",
		"year", year,
		"month", month,
		"start_utc", startUTC,
		"end_utc", endUTC,
	)

	// Cursor-based pagination to fetch daily aggregated records
	aggregated := make(map[aggregationKey]*subscription.SubscriptionUsageStats)
	var lastID uint
	limit := maxAggregationPageSize
	totalRecords := 0
	pageCount := 0

	for {
		// Read from daily aggregated stats table using cursor pagination
		records, err := uc.usageStatsRepo.GetDailyStatsByPeriod(ctx, startUTC, endUTC, lastID, limit)
		if err != nil {
			uc.logger.Errorw("failed to fetch daily stats for monthly aggregation",
				"error", err,
				"last_id", lastID,
			)
			return fmt.Errorf("failed to fetch daily stats (last_id %d): %w", lastID, err)
		}

		// No more data
		if len(records) == 0 {
			break
		}

		totalRecords += len(records)
		pageCount++

		// Aggregate current page records (daily -> monthly)
		uc.aggregateStatsRecords(records, aggregated, startUTC)

		// Update cursor to the last record's ID for next iteration
		lastID = records[len(records)-1].ID()

		// If returned records less than limit, it's the last page
		if len(records) < limit {
			break
		}
	}

	if totalRecords == 0 {
		uc.logger.Warnw("no daily stats found for monthly aggregation, ensure daily aggregation runs first",
			"year", year,
			"month", month,
		)
		return nil
	}

	// Upsert aggregated records
	successCount, errorCount := uc.upsertAggregatedStats(ctx, aggregated)

	uc.logger.Infow("monthly usage aggregation completed",
		"year", year,
		"month", month,
		"total_daily_records", totalRecords,
		"pages_processed", pageCount,
		"aggregated_groups", len(aggregated),
		"success_count", successCount,
		"error_count", errorCount,
	)

	if errorCount > 0 {
		return fmt.Errorf("monthly aggregation completed with %d errors", errorCount)
	}

	return nil
}

// upsertAggregatedStats upserts all aggregated stats records to the repository.
// Returns the count of successful and failed upserts.
func (uc *AggregateUsageUseCase) upsertAggregatedStats(
	ctx context.Context,
	aggregated map[aggregationKey]*subscription.SubscriptionUsageStats,
) (successCount, errorCount int) {
	for _, stats := range aggregated {
		if err := uc.usageStatsRepo.Upsert(ctx, stats); err != nil {
			uc.logger.Errorw("failed to upsert usage stats",
				"error", err,
				"resource_type", stats.ResourceType(),
				"resource_id", stats.ResourceID(),
			)
			errorCount++
			continue
		}
		successCount++
	}
	return successCount, errorCount
}

// aggregateStatsRecords aggregates usage stats records into the aggregated map.
// Used for monthly aggregation from daily stats.
func (uc *AggregateUsageUseCase) aggregateStatsRecords(
	records []*subscription.SubscriptionUsageStats,
	aggregated map[aggregationKey]*subscription.SubscriptionUsageStats,
	period time.Time,
) {
	for _, record := range records {
		var subID uint
		if record.SubscriptionID() != nil {
			subID = *record.SubscriptionID()
		}

		key := aggregationKey{
			subscriptionID: subID,
			resourceType:   record.ResourceType(),
			resourceID:     record.ResourceID(),
		}

		if _, exists := aggregated[key]; !exists {
			var subscriptionIDPtr *uint
			if subID != 0 {
				subscriptionIDPtr = &subID
			}

			stats, err := subscription.NewSubscriptionUsageStats(
				record.ResourceType(),
				record.ResourceID(),
				subscriptionIDPtr,
				subscription.GranularityMonthly,
				period,
			)
			if err != nil {
				uc.logger.Errorw("failed to create monthly usage stats entity",
					"error", err,
					"resource_type", record.ResourceType(),
					"resource_id", record.ResourceID(),
				)
				continue
			}
			aggregated[key] = stats
		}

		// Accumulate traffic from daily stats
		if err := aggregated[key].Accumulate(record.Upload(), record.Download()); err != nil {
			uc.logger.Warnw("failed to accumulate monthly traffic",
				"error", err,
				"resource_type", record.ResourceType(),
				"resource_id", record.ResourceID(),
			)
		}
	}
}
