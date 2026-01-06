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

// GetTotalBySubscriptionIDs retrieves total aggregated usage across multiple subscriptions
func (r *SubscriptionUsageStatsRepositoryImpl) GetTotalBySubscriptionIDs(
	ctx context.Context,
	subscriptionIDs []uint,
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
