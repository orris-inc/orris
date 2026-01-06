package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
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
// raw hourly data (subscription_usages) to aggregated stats (subscription_usage_stats).
type AggregateUsageUseCase struct {
	usageRepo      subscription.SubscriptionUsageRepository
	usageStatsRepo subscription.SubscriptionUsageStatsRepository
	logger         logger.Interface
}

// NewAggregateUsageUseCase creates a new aggregate usage use case instance.
func NewAggregateUsageUseCase(
	usageRepo subscription.SubscriptionUsageRepository,
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	logger logger.Interface,
) *AggregateUsageUseCase {
	return &AggregateUsageUseCase{
		usageRepo:      usageRepo,
		usageStatsRepo: usageStatsRepo,
		logger:         logger,
	}
}

// AggregateDailyUsage aggregates hourly data from yesterday into daily stats.
// It reads from subscription_usages table and writes to subscription_usage_stats table.
func (uc *AggregateUsageUseCase) AggregateDailyUsage(ctx context.Context) error {
	// Calculate yesterday in business timezone
	now := biztime.NowUTC()
	bizNow := biztime.ToBizTimezone(now)
	yesterday := bizNow.AddDate(0, 0, -1)

	// Get UTC time range for yesterday in business timezone
	startUTC := biztime.StartOfDayUTC(yesterday)
	endUTC := startUTC.Add(24 * time.Hour)

	uc.logger.Infow("starting daily usage aggregation",
		"date", yesterday.Format("2006-01-02"),
		"start_utc", startUTC,
		"end_utc", endUTC,
	)

	// Paginated iteration to fetch all records
	aggregated := make(map[aggregationKey]*subscription.SubscriptionUsageStats)
	page := 1
	pageSize := maxAggregationPageSize
	totalRecords := 0

	for {
		filter := subscription.UsageStatsFilter{
			From: startUTC,
			To:   endUTC,
		}
		filter.PageFilter.Page = page
		filter.PageFilter.PageSize = pageSize

		records, err := uc.usageRepo.GetUsageStats(ctx, filter)
		if err != nil {
			uc.logger.Errorw("failed to fetch hourly usage records",
				"error", err,
				"page", page,
			)
			return fmt.Errorf("failed to fetch hourly usage records (page %d): %w", page, err)
		}

		// No more data
		if len(records) == 0 {
			break
		}

		totalRecords += len(records)

		// Aggregate current page records
		uc.aggregateRecords(records, aggregated, subscription.GranularityDaily, startUTC)

		// If returned records less than pageSize, it's the last page
		if len(records) < pageSize {
			break
		}

		page++
	}

	if totalRecords == 0 {
		uc.logger.Infow("no hourly usage records found for aggregation",
			"date", yesterday.Format("2006-01-02"),
		)
		return nil
	}

	// Upsert aggregated records
	successCount, errorCount := uc.upsertAggregatedStats(ctx, aggregated)

	uc.logger.Infow("daily usage aggregation completed",
		"date", yesterday.Format("2006-01-02"),
		"total_records", totalRecords,
		"pages_processed", page,
		"aggregated_groups", len(aggregated),
		"success_count", successCount,
		"error_count", errorCount,
	)

	if errorCount > 0 {
		return fmt.Errorf("daily aggregation completed with %d errors", errorCount)
	}

	return nil
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

// aggregateRecords aggregates usage records into the aggregated map.
// This is a helper function to reduce code duplication between daily and monthly aggregation.
func (uc *AggregateUsageUseCase) aggregateRecords(
	records []*subscription.SubscriptionUsage,
	aggregated map[aggregationKey]*subscription.SubscriptionUsageStats,
	granularity subscription.Granularity,
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
			// Create new aggregated stats record
			var subscriptionIDPtr *uint
			if subID != 0 {
				subscriptionIDPtr = &subID
			}

			stats, err := subscription.NewSubscriptionUsageStats(
				record.ResourceType(),
				record.ResourceID(),
				subscriptionIDPtr,
				granularity,
				period,
			)
			if err != nil {
				uc.logger.Errorw("failed to create usage stats entity",
					"error", err,
					"resource_type", record.ResourceType(),
					"resource_id", record.ResourceID(),
				)
				continue
			}
			aggregated[key] = stats
		}

		// Accumulate traffic
		if err := aggregated[key].Accumulate(record.Upload(), record.Download()); err != nil {
			uc.logger.Warnw("failed to accumulate traffic",
				"error", err,
				"resource_type", record.ResourceType(),
				"resource_id", record.ResourceID(),
			)
		}
	}
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
