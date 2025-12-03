package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SubscriptionTrafficRepositoryImpl implements the node.SubscriptionTrafficRepository interface
type SubscriptionTrafficRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.SubscriptionTrafficMapper
	logger logger.Interface
}

// NewSubscriptionTrafficRepository creates a new subscription traffic repository instance
func NewSubscriptionTrafficRepository(db *gorm.DB, logger logger.Interface) node.SubscriptionTrafficRepository {
	return &SubscriptionTrafficRepositoryImpl{
		db:     db,
		mapper: mappers.NewSubscriptionTrafficMapper(),
		logger: logger,
	}
}

// RecordTraffic records a new traffic entry
func (r *SubscriptionTrafficRepositoryImpl) RecordTraffic(ctx context.Context, traffic *node.SubscriptionTraffic) error {
	model, err := r.mapper.ToModel(traffic)
	if err != nil {
		r.logger.Errorw("failed to map subscription traffic entity to model", "error", err)
		return fmt.Errorf("failed to map subscription traffic entity: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to record subscription traffic", "node_id", model.NodeID, "error", err)
		return fmt.Errorf("failed to record subscription traffic: %w", err)
	}

	if err := traffic.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set subscription traffic ID", "error", err)
		return fmt.Errorf("failed to set subscription traffic ID: %w", err)
	}

	r.logger.Infow("subscription traffic recorded successfully", "id", model.ID, "node_id", model.NodeID)
	return nil
}

// GetTrafficStats retrieves traffic statistics based on filter criteria
func (r *SubscriptionTrafficRepositoryImpl) GetTrafficStats(ctx context.Context, filter node.TrafficStatsFilter) ([]*node.SubscriptionTraffic, error) {
	query := r.db.WithContext(ctx).Model(&models.SubscriptionTrafficModel{})

	// Apply filters
	if filter.NodeID != nil {
		query = query.Where("node_id = ?", *filter.NodeID)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
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
	var trafficModels []*models.SubscriptionTrafficModel
	if err := query.Order("period DESC").Find(&trafficModels).Error; err != nil {
		r.logger.Errorw("failed to get traffic stats", "error", err)
		return nil, fmt.Errorf("failed to get traffic stats: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(trafficModels)
	if err != nil {
		r.logger.Errorw("failed to map subscription traffic models to entities", "error", err)
		return nil, fmt.Errorf("failed to map subscription traffic: %w", err)
	}

	return entities, nil
}

// GetTotalTraffic retrieves the total traffic for a node within a time range
func (r *SubscriptionTrafficRepositoryImpl) GetTotalTraffic(ctx context.Context, nodeID uint, from, to time.Time) (*node.TrafficSummary, error) {
	var result struct {
		TotalUpload   uint64
		TotalDownload uint64
		TotalTraffic  uint64
	}

	query := r.db.WithContext(ctx).Model(&models.SubscriptionTrafficModel{}).
		Select("COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_traffic").
		Where("node_id = ?", nodeID)

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	if err := query.Scan(&result).Error; err != nil {
		r.logger.Errorw("failed to get total traffic", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to get total traffic: %w", err)
	}

	summary := &node.TrafficSummary{
		NodeID:   nodeID,
		Upload:   result.TotalUpload,
		Download: result.TotalDownload,
		Total:    result.TotalTraffic,
		From:     from,
		To:       to,
	}

	return summary, nil
}

// AggregateDaily aggregates hourly traffic into daily statistics
func (r *SubscriptionTrafficRepositoryImpl) AggregateDaily(ctx context.Context, date time.Time) error {
	// Start a transaction for atomicity
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Calculate start and end of the day
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		// Aggregate traffic by node_id for the day
		var aggregatedRecords []struct {
			NodeID         uint
			UserID         *uint
			SubscriptionID *uint
			TotalUpload    uint64
			TotalDownload  uint64
			TotalTraffic   uint64
		}

		err := tx.Model(&models.SubscriptionTrafficModel{}).
			Select("node_id, user_id, subscription_id, SUM(upload) as total_upload, SUM(download) as total_download, SUM(total) as total_traffic").
			Where("period >= ? AND period < ?", startOfDay, endOfDay).
			Group("node_id, user_id, subscription_id").
			Scan(&aggregatedRecords).Error

		if err != nil {
			r.logger.Errorw("failed to aggregate daily traffic", "date", date, "error", err)
			return fmt.Errorf("failed to aggregate daily traffic: %w", err)
		}

		// Create or update daily records
		for _, record := range aggregatedRecords {
			dailyRecord := &models.SubscriptionTrafficModel{
				NodeID:         record.NodeID,
				UserID:         record.UserID,
				SubscriptionID: record.SubscriptionID,
				Upload:         record.TotalUpload,
				Download:       record.TotalDownload,
				Total:          record.TotalTraffic,
				Period:         startOfDay,
			}

			// Upsert: create or update if exists
			if err := tx.Where("node_id = ? AND period = ? AND user_id <=> ? AND subscription_id <=> ?",
				record.NodeID, startOfDay, record.UserID, record.SubscriptionID).
				Assign(map[string]interface{}{
					"upload":   record.TotalUpload,
					"download": record.TotalDownload,
					"total":    record.TotalTraffic,
				}).
				FirstOrCreate(dailyRecord).Error; err != nil {
				r.logger.Errorw("failed to upsert daily traffic record", "node_id", record.NodeID, "error", err)
				return fmt.Errorf("failed to upsert daily traffic record: %w", err)
			}
		}

		r.logger.Infow("daily traffic aggregated successfully", "date", date, "records", len(aggregatedRecords))
		return nil
	})
}

// AggregateMonthly aggregates daily traffic into monthly statistics
func (r *SubscriptionTrafficRepositoryImpl) AggregateMonthly(ctx context.Context, year int, month int) error {
	// Start a transaction for atomicity
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Calculate start and end of the month
		startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endOfMonth := startOfMonth.AddDate(0, 1, 0)

		// Aggregate traffic by node_id for the month
		var aggregatedRecords []struct {
			NodeID         uint
			UserID         *uint
			SubscriptionID *uint
			TotalUpload    uint64
			TotalDownload  uint64
			TotalTraffic   uint64
		}

		err := tx.Model(&models.SubscriptionTrafficModel{}).
			Select("node_id, user_id, subscription_id, SUM(upload) as total_upload, SUM(download) as total_download, SUM(total) as total_traffic").
			Where("period >= ? AND period < ?", startOfMonth, endOfMonth).
			Group("node_id, user_id, subscription_id").
			Scan(&aggregatedRecords).Error

		if err != nil {
			r.logger.Errorw("failed to aggregate monthly traffic", "year", year, "month", month, "error", err)
			return fmt.Errorf("failed to aggregate monthly traffic: %w", err)
		}

		// Create or update monthly records
		for _, record := range aggregatedRecords {
			monthlyRecord := &models.SubscriptionTrafficModel{
				NodeID:         record.NodeID,
				UserID:         record.UserID,
				SubscriptionID: record.SubscriptionID,
				Upload:         record.TotalUpload,
				Download:       record.TotalDownload,
				Total:          record.TotalTraffic,
				Period:         startOfMonth,
			}

			// Upsert: create or update if exists
			if err := tx.Where("node_id = ? AND period = ? AND user_id <=> ? AND subscription_id <=> ?",
				record.NodeID, startOfMonth, record.UserID, record.SubscriptionID).
				Assign(map[string]interface{}{
					"upload":   record.TotalUpload,
					"download": record.TotalDownload,
					"total":    record.TotalTraffic,
				}).
				FirstOrCreate(monthlyRecord).Error; err != nil {
				r.logger.Errorw("failed to upsert monthly traffic record", "node_id", record.NodeID, "error", err)
				return fmt.Errorf("failed to upsert monthly traffic record: %w", err)
			}
		}

		r.logger.Infow("monthly traffic aggregated successfully", "year", year, "month", month, "records", len(aggregatedRecords))
		return nil
	})
}

// GetDailyStats retrieves daily traffic statistics for a node
func (r *SubscriptionTrafficRepositoryImpl) GetDailyStats(ctx context.Context, nodeID uint, from, to time.Time) ([]*node.SubscriptionTraffic, error) {
	query := r.db.WithContext(ctx).Model(&models.SubscriptionTrafficModel{}).
		Where("node_id = ?", nodeID)

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	var trafficModels []*models.SubscriptionTrafficModel
	if err := query.Order("period ASC").Find(&trafficModels).Error; err != nil {
		r.logger.Errorw("failed to get daily stats", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(trafficModels)
	if err != nil {
		r.logger.Errorw("failed to map subscription traffic models to entities", "error", err)
		return nil, fmt.Errorf("failed to map subscription traffic: %w", err)
	}

	return entities, nil
}

// GetMonthlyStats retrieves monthly traffic statistics for a node
func (r *SubscriptionTrafficRepositoryImpl) GetMonthlyStats(ctx context.Context, nodeID uint, year int) ([]*node.SubscriptionTraffic, error) {
	startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endOfYear := startOfYear.AddDate(1, 0, 0)

	var trafficModels []*models.SubscriptionTrafficModel
	if err := r.db.WithContext(ctx).Model(&models.SubscriptionTrafficModel{}).
		Where("node_id = ? AND period >= ? AND period < ?", nodeID, startOfYear, endOfYear).
		Order("period ASC").
		Find(&trafficModels).Error; err != nil {
		r.logger.Errorw("failed to get monthly stats", "node_id", nodeID, "year", year, "error", err)
		return nil, fmt.Errorf("failed to get monthly stats: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(trafficModels)
	if err != nil {
		r.logger.Errorw("failed to map subscription traffic models to entities", "error", err)
		return nil, fmt.Errorf("failed to map subscription traffic: %w", err)
	}

	return entities, nil
}

// DeleteOldRecords deletes traffic records older than the specified time
func (r *SubscriptionTrafficRepositoryImpl) DeleteOldRecords(ctx context.Context, before time.Time) error {
	result := r.db.WithContext(ctx).Where("period < ?", before).Delete(&models.SubscriptionTrafficModel{})
	if result.Error != nil {
		r.logger.Errorw("failed to delete old traffic records", "before", before, "error", result.Error)
		return fmt.Errorf("failed to delete old traffic records: %w", result.Error)
	}

	r.logger.Infow("old traffic records deleted successfully", "before", before, "deleted_count", result.RowsAffected)
	return nil
}
