package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SubscriptionUsageRepositoryImpl implements the subscription.SubscriptionUsageRepository interface
type SubscriptionUsageRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.SubscriptionUsageMapper
	logger logger.Interface
}

// NewSubscriptionUsageRepository creates a new subscription usage repository instance
func NewSubscriptionUsageRepository(db *gorm.DB, logger logger.Interface) subscription.SubscriptionUsageRepository {
	return &SubscriptionUsageRepositoryImpl{
		db:     db,
		mapper: mappers.NewSubscriptionUsageMapper(),
		logger: logger,
	}
}

// RecordUsage records a new usage entry
func (r *SubscriptionUsageRepositoryImpl) RecordUsage(ctx context.Context, usage *subscription.SubscriptionUsage) error {
	model, err := r.mapper.ToModel(usage)
	if err != nil {
		r.logger.Errorw("failed to map subscription usage entity to model", "error", err)
		return fmt.Errorf("failed to map subscription usage entity: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to record subscription usage", "resource_type", model.ResourceType, "resource_id", model.ResourceID, "error", err)
		return fmt.Errorf("failed to record subscription usage: %w", err)
	}

	if err := usage.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set subscription usage ID", "error", err)
		return fmt.Errorf("failed to set subscription usage ID: %w", err)
	}

	r.logger.Infow("subscription usage recorded successfully", "id", model.ID, "resource_type", model.ResourceType, "resource_id", model.ResourceID)
	return nil
}

// GetUsageStats retrieves usage statistics based on filter criteria
func (r *SubscriptionUsageRepositoryImpl) GetUsageStats(ctx context.Context, filter subscription.UsageStatsFilter) ([]*subscription.SubscriptionUsage, error) {
	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{})

	// Apply filters
	if filter.ResourceType != nil {
		query = query.Where("resource_type = ?", *filter.ResourceType)
	}
	if filter.ResourceID != nil {
		query = query.Where("resource_id = ?", *filter.ResourceID)
	}
	if filter.SubscriptionID != nil {
		query = query.Where("subscription_id = ?", *filter.SubscriptionID)
	}
	if !filter.From.IsZero() {
		query = query.Where("period >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		query = query.Where("period <= ?", filter.To)
	}
	if filter.Period != nil && *filter.Period != "" {
		// Period format filtering could be enhanced based on requirements
		query = query.Order("period DESC")
	}

	// Apply pagination
	offset := filter.PageFilter.Offset()
	limit := filter.PageFilter.Limit()
	query = query.Offset(offset).Limit(limit)

	// Execute query
	var usageModels []*models.SubscriptionUsageModel
	if err := query.Order("period DESC").Find(&usageModels).Error; err != nil {
		r.logger.Errorw("failed to get usage stats", "error", err)
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(usageModels)
	if err != nil {
		r.logger.Errorw("failed to map subscription usage models to entities", "error", err)
		return nil, fmt.Errorf("failed to map subscription usage: %w", err)
	}

	return entities, nil
}

// GetTotalUsage retrieves the total usage for a resource within a time range
func (r *SubscriptionUsageRepositoryImpl) GetTotalUsage(ctx context.Context, resourceType string, resourceID uint, from, to time.Time) (*subscription.UsageSummary, error) {
	var result struct {
		TotalUpload   uint64
		TotalDownload uint64
		TotalUsage    uint64
	}

	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{}).
		Select("COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage").
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID)

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	if err := query.Scan(&result).Error; err != nil {
		r.logger.Errorw("failed to get total usage", "resource_type", resourceType, "resource_id", resourceID, "error", err)
		return nil, fmt.Errorf("failed to get total usage: %w", err)
	}

	summary := &subscription.UsageSummary{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Upload:       result.TotalUpload,
		Download:     result.TotalDownload,
		Total:        result.TotalUsage,
		From:         from,
		To:           to,
	}

	return summary, nil
}

// GetTotalUsageBySubscriptionID retrieves the total usage for a subscription within a time range
func (r *SubscriptionUsageRepositoryImpl) GetTotalUsageBySubscriptionID(ctx context.Context, subscriptionID uint, from, to time.Time) (*subscription.UsageSummary, error) {
	var result struct {
		TotalUpload   uint64
		TotalDownload uint64
		TotalUsage    uint64
	}

	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{}).
		Select("COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage").
		Where("subscription_id = ?", subscriptionID)

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	if err := query.Scan(&result).Error; err != nil {
		r.logger.Errorw("failed to get total usage by subscription ID", "subscription_id", subscriptionID, "error", err)
		return nil, fmt.Errorf("failed to get total usage by subscription ID: %w", err)
	}

	summary := &subscription.UsageSummary{
		Upload:   result.TotalUpload,
		Download: result.TotalDownload,
		Total:    result.TotalUsage,
		From:     from,
		To:       to,
	}

	return summary, nil
}

// AggregateDaily aggregates hourly usage into daily statistics
func (r *SubscriptionUsageRepositoryImpl) AggregateDaily(ctx context.Context, date time.Time) error {
	// Start a transaction for atomicity
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Calculate start and end of the day in business timezone, convert to UTC for query
		startOfDay := biztime.StartOfDayUTC(date)
		endOfDay := startOfDay.Add(24 * time.Hour)

		// Aggregate usage by resource_type and resource_id for the day
		var aggregatedRecords []struct {
			ResourceType   string
			ResourceID     uint
			SubscriptionID *uint
			TotalUpload    uint64
			TotalDownload  uint64
			TotalUsage     uint64
		}

		err := tx.Model(&models.SubscriptionUsageModel{}).
			Select("resource_type, resource_id, subscription_id, SUM(upload) as total_upload, SUM(download) as total_download, SUM(total) as total_usage").
			Where("period >= ? AND period < ?", startOfDay, endOfDay).
			Group("resource_type, resource_id, subscription_id").
			Scan(&aggregatedRecords).Error

		if err != nil {
			r.logger.Errorw("failed to aggregate daily usage", "date", date, "error", err)
			return fmt.Errorf("failed to aggregate daily usage: %w", err)
		}

		// Create or update daily records
		for _, record := range aggregatedRecords {
			dailyRecord := &models.SubscriptionUsageModel{
				ResourceType:   record.ResourceType,
				ResourceID:     record.ResourceID,
				SubscriptionID: record.SubscriptionID,
				Upload:         record.TotalUpload,
				Download:       record.TotalDownload,
				Total:          record.TotalUsage,
				Period:         startOfDay,
			}

			// Upsert: create or update if exists
			if err := tx.Where("resource_type = ? AND resource_id = ? AND period = ? AND subscription_id <=> ?",
				record.ResourceType, record.ResourceID, startOfDay, record.SubscriptionID).
				Assign(map[string]interface{}{
					"upload":   record.TotalUpload,
					"download": record.TotalDownload,
					"total":    record.TotalUsage,
				}).
				FirstOrCreate(dailyRecord).Error; err != nil {
				r.logger.Errorw("failed to upsert daily usage record", "resource_type", record.ResourceType, "resource_id", record.ResourceID, "error", err)
				return fmt.Errorf("failed to upsert daily usage record: %w", err)
			}
		}

		r.logger.Infow("daily usage aggregated successfully", "date", date, "records", len(aggregatedRecords))
		return nil
	})
}

// AggregateMonthly aggregates daily usage into monthly statistics
func (r *SubscriptionUsageRepositoryImpl) AggregateMonthly(ctx context.Context, year int, month int) error {
	// Start a transaction for atomicity
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Calculate start and end of the month in business timezone, convert to UTC for query
		startOfMonth := biztime.StartOfMonthUTC(year, time.Month(month))
		endOfMonth := startOfMonth.AddDate(0, 1, 0)

		// Aggregate usage by resource_type and resource_id for the month
		var aggregatedRecords []struct {
			ResourceType   string
			ResourceID     uint
			SubscriptionID *uint
			TotalUpload    uint64
			TotalDownload  uint64
			TotalUsage     uint64
		}

		err := tx.Model(&models.SubscriptionUsageModel{}).
			Select("resource_type, resource_id, subscription_id, SUM(upload) as total_upload, SUM(download) as total_download, SUM(total) as total_usage").
			Where("period >= ? AND period < ?", startOfMonth, endOfMonth).
			Group("resource_type, resource_id, subscription_id").
			Scan(&aggregatedRecords).Error

		if err != nil {
			r.logger.Errorw("failed to aggregate monthly usage", "year", year, "month", month, "error", err)
			return fmt.Errorf("failed to aggregate monthly usage: %w", err)
		}

		// Create or update monthly records
		for _, record := range aggregatedRecords {
			monthlyRecord := &models.SubscriptionUsageModel{
				ResourceType:   record.ResourceType,
				ResourceID:     record.ResourceID,
				SubscriptionID: record.SubscriptionID,
				Upload:         record.TotalUpload,
				Download:       record.TotalDownload,
				Total:          record.TotalUsage,
				Period:         startOfMonth,
			}

			// Upsert: create or update if exists
			if err := tx.Where("resource_type = ? AND resource_id = ? AND period = ? AND subscription_id <=> ?",
				record.ResourceType, record.ResourceID, startOfMonth, record.SubscriptionID).
				Assign(map[string]interface{}{
					"upload":   record.TotalUpload,
					"download": record.TotalDownload,
					"total":    record.TotalUsage,
				}).
				FirstOrCreate(monthlyRecord).Error; err != nil {
				r.logger.Errorw("failed to upsert monthly usage record", "resource_type", record.ResourceType, "resource_id", record.ResourceID, "error", err)
				return fmt.Errorf("failed to upsert monthly usage record: %w", err)
			}
		}

		r.logger.Infow("monthly usage aggregated successfully", "year", year, "month", month, "records", len(aggregatedRecords))
		return nil
	})
}

// GetDailyStats retrieves daily usage statistics for a resource
func (r *SubscriptionUsageRepositoryImpl) GetDailyStats(ctx context.Context, resourceType string, resourceID uint, from, to time.Time) ([]*subscription.SubscriptionUsage, error) {
	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{}).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID)

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	var usageModels []*models.SubscriptionUsageModel
	if err := query.Order("period ASC").Find(&usageModels).Error; err != nil {
		r.logger.Errorw("failed to get daily stats", "resource_type", resourceType, "resource_id", resourceID, "error", err)
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(usageModels)
	if err != nil {
		r.logger.Errorw("failed to map subscription usage models to entities", "error", err)
		return nil, fmt.Errorf("failed to map subscription usage: %w", err)
	}

	return entities, nil
}

// GetMonthlyStats retrieves monthly usage statistics for a resource
func (r *SubscriptionUsageRepositoryImpl) GetMonthlyStats(ctx context.Context, resourceType string, resourceID uint, year int) ([]*subscription.SubscriptionUsage, error) {
	// Use business timezone for year boundaries, convert to UTC for query
	startOfYear := biztime.StartOfYearUTC(year)
	endOfYear := startOfYear.AddDate(1, 0, 0)

	var usageModels []*models.SubscriptionUsageModel
	if err := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{}).
		Where("resource_type = ? AND resource_id = ? AND period >= ? AND period < ?", resourceType, resourceID, startOfYear, endOfYear).
		Order("period ASC").
		Find(&usageModels).Error; err != nil {
		r.logger.Errorw("failed to get monthly stats", "resource_type", resourceType, "resource_id", resourceID, "year", year, "error", err)
		return nil, fmt.Errorf("failed to get monthly stats: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(usageModels)
	if err != nil {
		r.logger.Errorw("failed to map subscription usage models to entities", "error", err)
		return nil, fmt.Errorf("failed to map subscription usage: %w", err)
	}

	return entities, nil
}

// GetTotalUsageBySubscriptionIDs retrieves total usage for a resource type across multiple subscriptions
func (r *SubscriptionUsageRepositoryImpl) GetTotalUsageBySubscriptionIDs(ctx context.Context, resourceType string, subscriptionIDs []uint, from, to time.Time) (*subscription.UsageSummary, error) {
	if len(subscriptionIDs) == 0 {
		return &subscription.UsageSummary{
			ResourceType: resourceType,
			Upload:       0,
			Download:     0,
			Total:        0,
			From:         from,
			To:           to,
		}, nil
	}

	var result struct {
		TotalUpload   uint64
		TotalDownload uint64
		TotalUsage    uint64
	}

	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{}).
		Select("COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage").
		Where("resource_type = ? AND subscription_id IN ?", resourceType, subscriptionIDs)

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	if err := query.Scan(&result).Error; err != nil {
		r.logger.Errorw("failed to get total usage by subscription IDs",
			"resource_type", resourceType,
			"subscription_ids_count", len(subscriptionIDs),
			"error", err,
		)
		return nil, fmt.Errorf("failed to get total usage by subscription IDs: %w", err)
	}

	summary := &subscription.UsageSummary{
		ResourceType: resourceType,
		Upload:       result.TotalUpload,
		Download:     result.TotalDownload,
		Total:        result.TotalUsage,
		From:         from,
		To:           to,
	}

	return summary, nil
}

// DeleteOldRecords deletes usage records older than the specified time
func (r *SubscriptionUsageRepositoryImpl) DeleteOldRecords(ctx context.Context, before time.Time) error {
	result := r.db.WithContext(ctx).Where("period < ?", before).Delete(&models.SubscriptionUsageModel{})
	if result.Error != nil {
		r.logger.Errorw("failed to delete old usage records", "before", before, "error", result.Error)
		return fmt.Errorf("failed to delete old usage records: %w", result.Error)
	}

	r.logger.Infow("old usage records deleted successfully", "before", before, "deleted_count", result.RowsAffected)
	return nil
}

// GetPlatformTotalUsage retrieves total usage across the entire platform
func (r *SubscriptionUsageRepositoryImpl) GetPlatformTotalUsage(ctx context.Context, resourceType *string, from, to time.Time) (*subscription.UsageSummary, error) {
	var result struct {
		TotalUpload   uint64
		TotalDownload uint64
		TotalUsage    uint64
	}

	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{}).
		Select("COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage")

	// Apply optional resource type filter
	if resourceType != nil && *resourceType != "" {
		query = query.Where("resource_type = ?", *resourceType)
	}

	// Apply time range filters
	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	if err := query.Scan(&result).Error; err != nil {
		r.logger.Errorw("failed to get platform total usage", "resource_type", resourceType, "error", err)
		return nil, fmt.Errorf("failed to get platform total usage: %w", err)
	}

	summary := &subscription.UsageSummary{
		Upload:   result.TotalUpload,
		Download: result.TotalDownload,
		Total:    result.TotalUsage,
		From:     from,
		To:       to,
	}

	if resourceType != nil {
		summary.ResourceType = *resourceType
	}

	r.logger.Infow("platform total usage retrieved successfully", "upload", result.TotalUpload, "download", result.TotalDownload, "total", result.TotalUsage)
	return summary, nil
}

// GetUsageGroupedBySubscription retrieves usage data grouped by subscription with pagination
func (r *SubscriptionUsageRepositoryImpl) GetUsageGroupedBySubscription(ctx context.Context, resourceType *string, from, to time.Time, page, pageSize int) ([]subscription.SubscriptionUsageSummary, int64, error) {
	// Build base query for aggregation
	baseQuery := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{})

	// Apply optional resource type filter
	if resourceType != nil && *resourceType != "" {
		baseQuery = baseQuery.Where("resource_type = ?", *resourceType)
	}

	// Apply time range filters
	if !from.IsZero() {
		baseQuery = baseQuery.Where("period >= ?", from)
	}
	if !to.IsZero() {
		baseQuery = baseQuery.Where("period <= ?", to)
	}

	// Count total number of distinct subscriptions
	var totalCount int64
	countQuery := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{}).
		Distinct("subscription_id")

	if resourceType != nil && *resourceType != "" {
		countQuery = countQuery.Where("resource_type = ?", *resourceType)
	}
	if !from.IsZero() {
		countQuery = countQuery.Where("period >= ?", from)
	}
	if !to.IsZero() {
		countQuery = countQuery.Where("period <= ?", to)
	}

	if err := countQuery.Count(&totalCount).Error; err != nil {
		r.logger.Errorw("failed to count subscriptions", "error", err)
		return nil, 0, fmt.Errorf("failed to count subscriptions: %w", err)
	}

	// Execute aggregation query with pagination
	var results []struct {
		SubscriptionID uint
		TotalUpload    uint64
		TotalDownload  uint64
		TotalUsage     uint64
	}

	offset := (page - 1) * pageSize
	err := baseQuery.
		Select("subscription_id, COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage").
		Group("subscription_id").
		Order("total_usage DESC").
		Limit(pageSize).
		Offset(offset).
		Scan(&results).Error

	if err != nil {
		r.logger.Errorw("failed to get usage grouped by subscription", "resource_type", resourceType, "error", err)
		return nil, 0, fmt.Errorf("failed to get usage grouped by subscription: %w", err)
	}

	// Convert to domain type
	summaries := make([]subscription.SubscriptionUsageSummary, len(results))
	for i, result := range results {
		summaries[i] = subscription.SubscriptionUsageSummary{
			SubscriptionID: result.SubscriptionID,
			Upload:         result.TotalUpload,
			Download:       result.TotalDownload,
			Total:          result.TotalUsage,
		}
	}

	r.logger.Infow("usage grouped by subscription retrieved successfully", "count", len(summaries), "total", totalCount)
	return summaries, totalCount, nil
}

// GetUsageGroupedByResourceID retrieves usage data grouped by resource ID with pagination
func (r *SubscriptionUsageRepositoryImpl) GetUsageGroupedByResourceID(ctx context.Context, resourceType string, from, to time.Time, page, pageSize int) ([]subscription.ResourceUsageSummary, int64, error) {
	// Build base query for aggregation
	baseQuery := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{}).
		Where("resource_type = ?", resourceType)

	// Apply time range filters
	if !from.IsZero() {
		baseQuery = baseQuery.Where("period >= ?", from)
	}
	if !to.IsZero() {
		baseQuery = baseQuery.Where("period <= ?", to)
	}

	// Count total number of distinct resource IDs
	var totalCount int64
	countQuery := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{}).
		Where("resource_type = ?", resourceType).
		Distinct("resource_id")

	if !from.IsZero() {
		countQuery = countQuery.Where("period >= ?", from)
	}
	if !to.IsZero() {
		countQuery = countQuery.Where("period <= ?", to)
	}

	if err := countQuery.Count(&totalCount).Error; err != nil {
		r.logger.Errorw("failed to count resources", "resource_type", resourceType, "error", err)
		return nil, 0, fmt.Errorf("failed to count resources: %w", err)
	}

	// Execute aggregation query with pagination
	var results []struct {
		ResourceType  string
		ResourceID    uint
		TotalUpload   uint64
		TotalDownload uint64
		TotalUsage    uint64
	}

	offset := (page - 1) * pageSize
	err := baseQuery.
		Select("resource_type, resource_id, COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage").
		Group("resource_type, resource_id").
		Order("total_usage DESC").
		Limit(pageSize).
		Offset(offset).
		Scan(&results).Error

	if err != nil {
		r.logger.Errorw("failed to get usage grouped by resource ID", "resource_type", resourceType, "error", err)
		return nil, 0, fmt.Errorf("failed to get usage grouped by resource ID: %w", err)
	}

	// Convert to domain type
	summaries := make([]subscription.ResourceUsageSummary, len(results))
	for i, result := range results {
		summaries[i] = subscription.ResourceUsageSummary{
			ResourceType: result.ResourceType,
			ResourceID:   result.ResourceID,
			Upload:       result.TotalUpload,
			Download:     result.TotalDownload,
			Total:        result.TotalUsage,
		}
	}

	r.logger.Infow("usage grouped by resource ID retrieved successfully", "resource_type", resourceType, "count", len(summaries), "total", totalCount)
	return summaries, totalCount, nil
}

// GetTopSubscriptionsByUsage retrieves top N subscriptions by total usage
func (r *SubscriptionUsageRepositoryImpl) GetTopSubscriptionsByUsage(ctx context.Context, resourceType *string, from, to time.Time, limit int) ([]subscription.SubscriptionUsageSummary, error) {
	// Build base query
	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{})

	// Filter out records without subscription_id (NULL subscription_id)
	query = query.Where("subscription_id IS NOT NULL")

	// Apply optional resource type filter
	if resourceType != nil && *resourceType != "" {
		query = query.Where("resource_type = ?", *resourceType)
	}

	// Apply time range filters
	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	// Execute aggregation query
	var results []struct {
		SubscriptionID uint
		TotalUpload    uint64
		TotalDownload  uint64
		TotalUsage     uint64
	}

	err := query.
		Select("subscription_id, COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage").
		Group("subscription_id").
		Order("total_usage DESC").
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		r.logger.Errorw("failed to get top subscriptions by usage", "resource_type", resourceType, "limit", limit, "error", err)
		return nil, fmt.Errorf("failed to get top subscriptions by usage: %w", err)
	}

	// Convert to domain type
	summaries := make([]subscription.SubscriptionUsageSummary, len(results))
	for i, result := range results {
		summaries[i] = subscription.SubscriptionUsageSummary{
			SubscriptionID: result.SubscriptionID,
			Upload:         result.TotalUpload,
			Download:       result.TotalDownload,
			Total:          result.TotalUsage,
		}
	}

	r.logger.Infow("top subscriptions by usage retrieved successfully", "count", len(summaries), "limit", limit)
	return summaries, nil
}

// GetUsageTrend retrieves usage trend data with specified granularity (hour/day/month)
func (r *SubscriptionUsageRepositoryImpl) GetUsageTrend(ctx context.Context, resourceType *string, from, to time.Time, granularity string) ([]subscription.UsageTrendPoint, error) {
	// Build base query
	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{})

	// Apply optional resource type filter
	if resourceType != nil && *resourceType != "" {
		query = query.Where("resource_type = ?", *resourceType)
	}

	// Apply time range filters
	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	// Determine date truncation based on granularity
	// Use CONVERT_TZ to convert UTC to business timezone before formatting
	// This ensures day/month boundaries align with business timezone
	tzOffset := biztime.MySQLTimezoneOffset()
	var dateFormat string
	switch granularity {
	case "hour":
		dateFormat = fmt.Sprintf("DATE_FORMAT(CONVERT_TZ(period, '+00:00', '%s'), '%%Y-%%m-%%d %%H:00:00')", tzOffset)
	case "day":
		dateFormat = fmt.Sprintf("DATE_FORMAT(CONVERT_TZ(period, '+00:00', '%s'), '%%Y-%%m-%%d')", tzOffset)
	case "month":
		dateFormat = fmt.Sprintf("DATE_FORMAT(CONVERT_TZ(period, '+00:00', '%s'), '%%Y-%%m-01')", tzOffset)
	default:
		r.logger.Errorw("invalid granularity", "granularity", granularity)
		return nil, fmt.Errorf("invalid granularity: %s, must be one of: hour, day, month", granularity)
	}

	// Execute aggregation query
	// Note: DATE_FORMAT returns a string, so we use string type for Period
	var results []struct {
		Period        string
		TotalUpload   uint64
		TotalDownload uint64
		TotalUsage    uint64
	}

	err := query.
		Select(fmt.Sprintf("%s as period, COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage", dateFormat)).
		Group(dateFormat).
		Order("period ASC").
		Scan(&results).Error

	if err != nil {
		r.logger.Errorw("failed to get usage trend", "resource_type", resourceType, "granularity", granularity, "error", err)
		return nil, fmt.Errorf("failed to get usage trend: %w", err)
	}

	// Determine time format for parsing based on granularity
	var timeLayout string
	switch granularity {
	case "hour":
		timeLayout = "2006-01-02 15:00:00"
	case "day":
		timeLayout = "2006-01-02"
	case "month":
		timeLayout = "2006-01-02"
	}

	// Convert to domain type
	// Parse the period string in business timezone, then convert to UTC
	trendPoints := make([]subscription.UsageTrendPoint, len(results))
	for i, result := range results {
		parsedTime, parseErr := time.ParseInLocation(timeLayout, result.Period, biztime.Location())
		if parseErr != nil {
			r.logger.Warnw("failed to parse period", "period", result.Period, "layout", timeLayout, "error", parseErr)
			// Use zero time if parsing fails
			parsedTime = time.Time{}
		} else {
			// Convert to UTC for consistent storage/transport
			parsedTime = parsedTime.UTC()
		}
		trendPoints[i] = subscription.UsageTrendPoint{
			Period:   parsedTime,
			Upload:   result.TotalUpload,
			Download: result.TotalDownload,
			Total:    result.TotalUsage,
		}
	}

	r.logger.Infow("usage trend retrieved successfully", "granularity", granularity, "count", len(trendPoints))
	return trendPoints, nil
}

// GetSubscriptionUsageTrend retrieves usage trend data for a specific subscription with specified granularity
func (r *SubscriptionUsageRepositoryImpl) GetSubscriptionUsageTrend(ctx context.Context, subscriptionID uint, from, to time.Time, granularity string) ([]subscription.SubscriptionUsageTrendPoint, error) {
	// Build base query
	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageModel{}).
		Where("subscription_id = ?", subscriptionID)

	// Apply time range filters
	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	// Determine date truncation based on granularity
	// Use CONVERT_TZ to convert UTC to business timezone before formatting
	tzOffset := biztime.MySQLTimezoneOffset()
	var dateFormat string
	switch granularity {
	case "hour":
		dateFormat = fmt.Sprintf("DATE_FORMAT(CONVERT_TZ(period, '+00:00', '%s'), '%%Y-%%m-%%d %%H:00:00')", tzOffset)
	case "day":
		dateFormat = fmt.Sprintf("DATE_FORMAT(CONVERT_TZ(period, '+00:00', '%s'), '%%Y-%%m-%%d')", tzOffset)
	case "month":
		dateFormat = fmt.Sprintf("DATE_FORMAT(CONVERT_TZ(period, '+00:00', '%s'), '%%Y-%%m-01')", tzOffset)
	default:
		r.logger.Errorw("invalid granularity for subscription usage trend", "granularity", granularity)
		return nil, fmt.Errorf("invalid granularity: %s, must be one of: hour, day, month", granularity)
	}

	// Execute aggregation query grouped by resource_type, resource_id, and period
	var results []struct {
		ResourceType  string
		ResourceID    uint
		Period        string
		TotalUpload   uint64
		TotalDownload uint64
		TotalUsage    uint64
	}

	// Limit results to prevent excessive data transfer
	const maxTrendRecords = 1000

	err := query.
		Select(fmt.Sprintf("resource_type, resource_id, %s as period, COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage", dateFormat)).
		Group(fmt.Sprintf("resource_type, resource_id, %s", dateFormat)).
		Order("period ASC, resource_type ASC, resource_id ASC").
		Limit(maxTrendRecords).
		Scan(&results).Error

	if err != nil {
		r.logger.Errorw("failed to get subscription usage trend", "subscription_id", subscriptionID, "granularity", granularity, "error", err)
		return nil, fmt.Errorf("failed to get subscription usage trend: %w", err)
	}

	// Determine time format for parsing based on granularity
	var timeLayout string
	switch granularity {
	case "hour":
		timeLayout = "2006-01-02 15:00:00"
	case "day":
		timeLayout = "2006-01-02"
	case "month":
		timeLayout = "2006-01-02"
	}

	// Convert to domain type
	trendPoints := make([]subscription.SubscriptionUsageTrendPoint, len(results))
	for i, result := range results {
		parsedTime, parseErr := time.ParseInLocation(timeLayout, result.Period, biztime.Location())
		if parseErr != nil {
			r.logger.Warnw("failed to parse period", "period", result.Period, "layout", timeLayout, "error", parseErr)
			parsedTime = time.Time{}
		} else {
			// Convert to UTC for consistent storage/transport
			parsedTime = parsedTime.UTC()
		}
		trendPoints[i] = subscription.SubscriptionUsageTrendPoint{
			ResourceType: result.ResourceType,
			ResourceID:   result.ResourceID,
			Period:       parsedTime,
			Upload:       result.TotalUpload,
			Download:     result.TotalDownload,
			Total:        result.TotalUsage,
		}
	}

	r.logger.Infow("subscription usage trend retrieved successfully", "subscription_id", subscriptionID, "granularity", granularity, "count", len(trendPoints))
	return trendPoints, nil
}
