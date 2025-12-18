package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
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

// AggregateDaily aggregates hourly usage into daily statistics
func (r *SubscriptionUsageRepositoryImpl) AggregateDaily(ctx context.Context, date time.Time) error {
	// Start a transaction for atomicity
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Calculate start and end of the day
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
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
		// Calculate start and end of the month
		startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
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
	startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
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
