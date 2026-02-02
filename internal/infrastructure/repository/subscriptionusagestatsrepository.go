package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SubscriptionUsageStatsRepositoryImpl implements the subscription.SubscriptionUsageStatsRepository interface
type SubscriptionUsageStatsRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.SubscriptionUsageStatsMapper
	logger logger.Interface
}

// NewSubscriptionUsageStatsRepository creates a new subscription usage stats repository instance
func NewSubscriptionUsageStatsRepository(db *gorm.DB, logger logger.Interface) subscription.SubscriptionUsageStatsRepository {
	return &SubscriptionUsageStatsRepositoryImpl{
		db:     db,
		mapper: mappers.NewSubscriptionUsageStatsMapper(),
		logger: logger,
	}
}

// Upsert inserts or updates an aggregated usage stats record
func (r *SubscriptionUsageStatsRepositoryImpl) Upsert(ctx context.Context, stats *subscription.SubscriptionUsageStats) error {
	model, err := r.mapper.ToModel(stats)
	if err != nil {
		r.logger.Errorw("failed to map subscription usage stats entity to model", "error", err)
		return fmt.Errorf("failed to map subscription usage stats entity: %w", err)
	}

	// Use ON DUPLICATE KEY UPDATE for upsert
	err = r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "subscription_id"},
			{Name: "resource_type"},
			{Name: "resource_id"},
			{Name: "granularity"},
			{Name: "period"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"upload", "download", "total", "updated_at"}),
	}).Create(model).Error

	if err != nil {
		r.logger.Errorw("failed to upsert subscription usage stats",
			"resource_type", model.ResourceType,
			"resource_id", model.ResourceID,
			"granularity", model.Granularity,
			"period", model.Period,
			"error", err,
		)
		return fmt.Errorf("failed to upsert subscription usage stats: %w", err)
	}

	// Set ID back to entity if it was a new record
	if stats.ID() == 0 && model.ID != 0 {
		if err := stats.SetID(model.ID); err != nil {
			r.logger.Warnw("failed to set subscription usage stats ID", "error", err)
		}
	}

	r.logger.Infow("subscription usage stats upserted successfully",
		"id", model.ID,
		"resource_type", model.ResourceType,
		"resource_id", model.ResourceID,
		"granularity", model.Granularity,
		"period", model.Period,
	)
	return nil
}

// GetBySubscriptionID retrieves aggregated usage stats for a subscription within a time range
func (r *SubscriptionUsageStatsRepositoryImpl) GetBySubscriptionID(
	ctx context.Context,
	subscriptionID uint,
	granularity subscription.Granularity,
	from, to time.Time,
) ([]*subscription.SubscriptionUsageStats, error) {
	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageStatsModel{}).
		Where("subscription_id = ? AND granularity = ?", subscriptionID, granularity.String())

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	var statsModels []*models.SubscriptionUsageStatsModel
	if err := query.Order("period ASC").Find(&statsModels).Error; err != nil {
		r.logger.Errorw("failed to get usage stats by subscription ID",
			"subscription_id", subscriptionID,
			"granularity", granularity,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get usage stats by subscription ID: %w", err)
	}

	entities, err := r.mapper.ToEntities(statsModels)
	if err != nil {
		r.logger.Errorw("failed to map subscription usage stats models to entities", "error", err)
		return nil, fmt.Errorf("failed to map subscription usage stats: %w", err)
	}

	return entities, nil
}

// GetTotalBySubscriptionIDs retrieves total aggregated usage across multiple subscriptions.
// If resourceType is nil, returns usage for all resource types (used for Hybrid plans).
// If resourceType is specified, returns usage only for that resource type (used for Forward/Node plans).
func (r *SubscriptionUsageStatsRepositoryImpl) GetTotalBySubscriptionIDs(
	ctx context.Context,
	subscriptionIDs []uint,
	resourceType *string,
	granularity subscription.Granularity,
	from, to time.Time,
) (*subscription.UsageSummary, error) {
	if len(subscriptionIDs) == 0 {
		return &subscription.UsageSummary{
			Upload:   0,
			Download: 0,
			Total:    0,
			From:     from,
			To:       to,
		}, nil
	}

	var result struct {
		TotalUpload   uint64
		TotalDownload uint64
		TotalUsage    uint64
	}

	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageStatsModel{}).
		Select("COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage").
		Where("subscription_id IN ? AND granularity = ?", subscriptionIDs, granularity.String())

	// Filter by resource type if specified
	if resourceType != nil {
		query = query.Where("resource_type = ?", *resourceType)
	}

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	if err := query.Scan(&result).Error; err != nil {
		r.logger.Errorw("failed to get total usage by subscription IDs",
			"subscription_ids_count", len(subscriptionIDs),
			"granularity", granularity,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get total usage by subscription IDs: %w", err)
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

// GetTotalBySubscriptionIDsGrouped retrieves total aggregated usage for multiple subscriptions,
// returning results grouped by subscription ID. This is more efficient than calling
// GetTotalBySubscriptionIDs multiple times when you need per-subscription usage.
func (r *SubscriptionUsageStatsRepositoryImpl) GetTotalBySubscriptionIDsGrouped(
	ctx context.Context,
	subscriptionIDs []uint,
	resourceType *string,
	granularity subscription.Granularity,
	from, to time.Time,
) (map[uint]*subscription.UsageSummary, error) {
	result := make(map[uint]*subscription.UsageSummary, len(subscriptionIDs))

	if len(subscriptionIDs) == 0 {
		return result, nil
	}

	var dbResults []struct {
		SubscriptionID uint
		TotalUpload    uint64
		TotalDownload  uint64
		TotalUsage     uint64
	}

	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageStatsModel{}).
		Select("subscription_id, COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage").
		Where("subscription_id IN ? AND granularity = ?", subscriptionIDs, granularity.String()).
		Group("subscription_id")

	// Filter by resource type if specified
	if resourceType != nil {
		query = query.Where("resource_type = ?", *resourceType)
	}

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	if err := query.Scan(&dbResults).Error; err != nil {
		r.logger.Errorw("failed to get total usage grouped by subscription IDs",
			"subscription_ids_count", len(subscriptionIDs),
			"granularity", granularity,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get total usage grouped by subscription IDs: %w", err)
	}

	for _, res := range dbResults {
		result[res.SubscriptionID] = &subscription.UsageSummary{
			Upload:   res.TotalUpload,
			Download: res.TotalDownload,
			Total:    res.TotalUsage,
			From:     from,
			To:       to,
		}
	}

	return result, nil
}

// GetByResourceID retrieves aggregated usage stats for a specific resource within a time range
func (r *SubscriptionUsageStatsRepositoryImpl) GetByResourceID(
	ctx context.Context,
	resourceType string,
	resourceID uint,
	granularity subscription.Granularity,
	from, to time.Time,
) ([]*subscription.SubscriptionUsageStats, error) {
	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageStatsModel{}).
		Where("resource_type = ? AND resource_id = ? AND granularity = ?", resourceType, resourceID, granularity.String())

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	var statsModels []*models.SubscriptionUsageStatsModel
	if err := query.Order("period ASC").Find(&statsModels).Error; err != nil {
		r.logger.Errorw("failed to get usage stats by resource ID",
			"resource_type", resourceType,
			"resource_id", resourceID,
			"granularity", granularity,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get usage stats by resource ID: %w", err)
	}

	entities, err := r.mapper.ToEntities(statsModels)
	if err != nil {
		r.logger.Errorw("failed to map subscription usage stats models to entities", "error", err)
		return nil, fmt.Errorf("failed to map subscription usage stats: %w", err)
	}

	return entities, nil
}

// GetTotalByResourceID retrieves total aggregated usage for a specific resource within a time range
func (r *SubscriptionUsageStatsRepositoryImpl) GetTotalByResourceID(
	ctx context.Context,
	resourceType string,
	resourceID uint,
	granularity subscription.Granularity,
	from, to time.Time,
) (*subscription.UsageSummary, error) {
	var result struct {
		TotalUpload   uint64
		TotalDownload uint64
		TotalUsage    uint64
	}

	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageStatsModel{}).
		Select("COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage").
		Where("resource_type = ? AND resource_id = ? AND granularity = ?", resourceType, resourceID, granularity.String())

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	if err := query.Scan(&result).Error; err != nil {
		r.logger.Errorw("failed to get total usage by resource ID",
			"resource_type", resourceType,
			"resource_id", resourceID,
			"granularity", granularity,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get total usage by resource ID: %w", err)
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

// DeleteOldRecords deletes aggregated usage records older than the specified time
func (r *SubscriptionUsageStatsRepositoryImpl) DeleteOldRecords(
	ctx context.Context,
	granularity subscription.Granularity,
	before time.Time,
) error {
	result := r.db.WithContext(ctx).
		Where("granularity = ? AND period < ?", granularity.String(), before).
		Delete(&models.SubscriptionUsageStatsModel{})

	if result.Error != nil {
		r.logger.Errorw("failed to delete old usage stats records",
			"granularity", granularity,
			"before", before,
			"error", result.Error,
		)
		return fmt.Errorf("failed to delete old usage stats records: %w", result.Error)
	}

	r.logger.Infow("old usage stats records deleted successfully",
		"granularity", granularity,
		"before", before,
		"deleted_count", result.RowsAffected,
	)
	return nil
}

// GetDailyStatsByPeriod retrieves daily aggregated stats within a time range using cursor-based pagination.
// lastID is the ID of the last record from the previous page (use 0 for the first page).
// limit is the maximum number of records to return.
// Cursor-based pagination provides consistent performance regardless of dataset size.
func (r *SubscriptionUsageStatsRepositoryImpl) GetDailyStatsByPeriod(
	ctx context.Context,
	from, to time.Time,
	lastID uint,
	limit int,
) ([]*subscription.SubscriptionUsageStats, error) {
	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageStatsModel{}).
		Where("granularity = ?", subscription.GranularityDaily.String())

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period < ?", to)
	}

	// Cursor-based pagination: fetch records with ID > lastID
	if lastID > 0 {
		query = query.Where("id > ?", lastID)
	}

	query = query.Limit(limit).Order("id ASC")

	var statsModels []*models.SubscriptionUsageStatsModel
	if err := query.Find(&statsModels).Error; err != nil {
		r.logger.Errorw("failed to get daily stats by period",
			"from", from,
			"to", to,
			"last_id", lastID,
			"limit", limit,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}

	return r.mapper.ToEntities(statsModels)
}

// GetPlatformTotalUsage retrieves total platform-wide usage across all subscriptions within a time range.
// Used for admin summary notifications.
func (r *SubscriptionUsageStatsRepositoryImpl) GetPlatformTotalUsage(
	ctx context.Context,
	granularity subscription.Granularity,
	from, to time.Time,
) (*subscription.UsageSummary, error) {
	var result struct {
		TotalUpload   uint64
		TotalDownload uint64
		TotalUsage    uint64
	}

	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageStatsModel{}).
		Select("COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage").
		Where("granularity = ?", granularity.String())

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	if err := query.Scan(&result).Error; err != nil {
		r.logger.Errorw("failed to get platform total usage from stats",
			"granularity", granularity,
			"from", from,
			"to", to,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get platform total usage: %w", err)
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

// GetPlatformTotalUsageByResourceType retrieves total platform-wide usage filtered by resource type.
// If resourceType is nil, returns usage for all resource types.
// Uses daily granularity for aggregation.
func (r *SubscriptionUsageStatsRepositoryImpl) GetPlatformTotalUsageByResourceType(
	ctx context.Context,
	resourceType *string,
	from, to time.Time,
) (*subscription.UsageSummary, error) {
	var result struct {
		TotalUpload   uint64
		TotalDownload uint64
		TotalUsage    uint64
	}

	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageStatsModel{}).
		Select("COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_usage").
		Where("granularity = ?", subscription.GranularityDaily.String())

	if resourceType != nil {
		query = query.Where("resource_type = ?", *resourceType)
	}

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	if err := query.Scan(&result).Error; err != nil {
		r.logger.Errorw("failed to get platform total usage by resource type",
			"resource_type", resourceType,
			"from", from,
			"to", to,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get platform total usage: %w", err)
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

// GetUsageGroupedBySubscription retrieves aggregated usage grouped by subscription with pagination.
// Uses daily granularity for aggregation.
func (r *SubscriptionUsageStatsRepositoryImpl) GetUsageGroupedBySubscription(
	ctx context.Context,
	resourceType *string,
	from, to time.Time,
	page, pageSize int,
) ([]subscription.SubscriptionUsageSummary, int64, error) {
	// Build base query for both count and data retrieval
	baseQuery := r.db.WithContext(ctx).Model(&models.SubscriptionUsageStatsModel{}).
		Where("granularity = ?", subscription.GranularityDaily.String()).
		Where("subscription_id IS NOT NULL")

	if resourceType != nil {
		baseQuery = baseQuery.Where("resource_type = ?", *resourceType)
	}
	if !from.IsZero() {
		baseQuery = baseQuery.Where("period >= ?", from)
	}
	if !to.IsZero() {
		baseQuery = baseQuery.Where("period <= ?", to)
	}

	// Count distinct subscriptions
	var total int64
	countQuery := baseQuery.Session(&gorm.Session{}).Select("COUNT(DISTINCT subscription_id)")
	if err := countQuery.Scan(&total).Error; err != nil {
		r.logger.Errorw("failed to count subscriptions for usage stats",
			"resource_type", resourceType,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to count subscriptions: %w", err)
	}

	// Get aggregated usage grouped by subscription
	var results []struct {
		SubscriptionID uint
		Upload         uint64
		Download       uint64
		Total          uint64
	}

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	dataQuery := baseQuery.Session(&gorm.Session{}).
		Select("subscription_id, SUM(upload) as upload, SUM(download) as download, SUM(total) as total").
		Group("subscription_id").
		Order("total DESC").
		Offset(offset).
		Limit(pageSize)

	if err := dataQuery.Scan(&results).Error; err != nil {
		r.logger.Errorw("failed to get usage grouped by subscription",
			"resource_type", resourceType,
			"from", from,
			"to", to,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to get usage grouped by subscription: %w", err)
	}

	summaries := make([]subscription.SubscriptionUsageSummary, 0, len(results))
	for _, res := range results {
		summaries = append(summaries, subscription.SubscriptionUsageSummary{
			SubscriptionID: res.SubscriptionID,
			Upload:         res.Upload,
			Download:       res.Download,
			Total:          res.Total,
		})
	}

	return summaries, total, nil
}

// GetUsageGroupedByResourceID retrieves aggregated usage grouped by resource ID with pagination.
// Uses daily granularity for aggregation.
func (r *SubscriptionUsageStatsRepositoryImpl) GetUsageGroupedByResourceID(
	ctx context.Context,
	resourceType string,
	from, to time.Time,
	page, pageSize int,
) ([]subscription.ResourceUsageSummary, int64, error) {
	baseQuery := r.db.WithContext(ctx).Model(&models.SubscriptionUsageStatsModel{}).
		Where("granularity = ? AND resource_type = ?", subscription.GranularityDaily.String(), resourceType)

	if !from.IsZero() {
		baseQuery = baseQuery.Where("period >= ?", from)
	}
	if !to.IsZero() {
		baseQuery = baseQuery.Where("period <= ?", to)
	}

	// Count distinct resources
	var total int64
	countQuery := baseQuery.Session(&gorm.Session{}).Select("COUNT(DISTINCT resource_id)")
	if err := countQuery.Scan(&total).Error; err != nil {
		r.logger.Errorw("failed to count resources for usage stats",
			"resource_type", resourceType,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to count resources: %w", err)
	}

	// Get aggregated usage grouped by resource
	var results []struct {
		ResourceID uint
		Upload     uint64
		Download   uint64
		Total      uint64
	}

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	dataQuery := baseQuery.Session(&gorm.Session{}).
		Select("resource_id, SUM(upload) as upload, SUM(download) as download, SUM(total) as total").
		Group("resource_id").
		Order("total DESC").
		Offset(offset).
		Limit(pageSize)

	if err := dataQuery.Scan(&results).Error; err != nil {
		r.logger.Errorw("failed to get usage grouped by resource ID",
			"resource_type", resourceType,
			"from", from,
			"to", to,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to get usage grouped by resource ID: %w", err)
	}

	summaries := make([]subscription.ResourceUsageSummary, 0, len(results))
	for _, res := range results {
		summaries = append(summaries, subscription.ResourceUsageSummary{
			ResourceType: resourceType,
			ResourceID:   res.ResourceID,
			Upload:       res.Upload,
			Download:     res.Download,
			Total:        res.Total,
		})
	}

	return summaries, total, nil
}

// GetTopSubscriptionsByUsage retrieves top N subscriptions by total usage.
// Uses daily granularity for aggregation.
func (r *SubscriptionUsageStatsRepositoryImpl) GetTopSubscriptionsByUsage(
	ctx context.Context,
	resourceType *string,
	from, to time.Time,
	limit int,
) ([]subscription.SubscriptionUsageSummary, error) {
	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageStatsModel{}).
		Select("subscription_id, SUM(upload) as upload, SUM(download) as download, SUM(total) as total").
		Where("granularity = ?", subscription.GranularityDaily.String()).
		Where("subscription_id IS NOT NULL")

	if resourceType != nil {
		query = query.Where("resource_type = ?", *resourceType)
	}
	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	query = query.Group("subscription_id").Order("total DESC").Limit(limit)

	var results []struct {
		SubscriptionID uint
		Upload         uint64
		Download       uint64
		Total          uint64
	}

	if err := query.Scan(&results).Error; err != nil {
		r.logger.Errorw("failed to get top subscriptions by usage",
			"resource_type", resourceType,
			"limit", limit,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get top subscriptions: %w", err)
	}

	summaries := make([]subscription.SubscriptionUsageSummary, 0, len(results))
	for _, res := range results {
		summaries = append(summaries, subscription.SubscriptionUsageSummary{
			SubscriptionID: res.SubscriptionID,
			Upload:         res.Upload,
			Download:       res.Download,
			Total:          res.Total,
		})
	}

	return summaries, nil
}

// GetUsageTrend retrieves usage trend data grouped by time period with specified granularity (day/month).
// Note: For admin analytics, hour granularity is not supported as data is stored at daily level.
func (r *SubscriptionUsageStatsRepositoryImpl) GetUsageTrend(
	ctx context.Context,
	resourceType *string,
	from, to time.Time,
	granularity string,
) ([]subscription.UsageTrendPoint, error) {
	// Determine which granularity to use for query based on requested display granularity
	var dbGranularity subscription.Granularity
	var periodSelect string

	switch granularity {
	case "day":
		dbGranularity = subscription.GranularityDaily
		periodSelect = "DATE(period) as period_date"
	case "month":
		// Use monthly stats if available, otherwise aggregate from daily
		dbGranularity = subscription.GranularityMonthly
		periodSelect = "DATE_FORMAT(period, '%Y-%m-01') as period_date"
	default:
		// Default to daily for unsupported granularities
		dbGranularity = subscription.GranularityDaily
		periodSelect = "DATE(period) as period_date"
	}

	query := r.db.WithContext(ctx).Model(&models.SubscriptionUsageStatsModel{}).
		Select(periodSelect+", SUM(upload) as upload, SUM(download) as download, SUM(total) as total").
		Where("granularity = ?", dbGranularity.String())

	if resourceType != nil {
		query = query.Where("resource_type = ?", *resourceType)
	}
	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	query = query.Group("period_date").Order("period_date ASC")

	var results []struct {
		PeriodDate time.Time
		Upload     uint64
		Download   uint64
		Total      uint64
	}

	if err := query.Scan(&results).Error; err != nil {
		r.logger.Errorw("failed to get usage trend",
			"resource_type", resourceType,
			"granularity", granularity,
			"from", from,
			"to", to,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get usage trend: %w", err)
	}

	points := make([]subscription.UsageTrendPoint, 0, len(results))
	for _, res := range results {
		points = append(points, subscription.UsageTrendPoint{
			Period:   res.PeriodDate,
			Upload:   res.Upload,
			Download: res.Download,
			Total:    res.Total,
		})
	}

	return points, nil
}
