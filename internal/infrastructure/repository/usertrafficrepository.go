package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"orris/internal/domain/node"
	"orris/internal/infrastructure/persistence/mappers"
	"orris/internal/infrastructure/persistence/models"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// UserTrafficRepositoryImpl implements the node.UserTrafficRepository interface
type UserTrafficRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.UserTrafficMapper
	logger logger.Interface
}

// NewUserTrafficRepository creates a new user traffic repository instance
func NewUserTrafficRepository(db *gorm.DB, logger logger.Interface) node.UserTrafficRepository {
	return &UserTrafficRepositoryImpl{
		db:     db,
		mapper: mappers.NewUserTrafficMapper(),
		logger: logger,
	}
}

// Create creates a new user traffic record
func (r *UserTrafficRepositoryImpl) Create(ctx context.Context, traffic *node.UserTraffic) error {
	model, err := r.mapper.ToModel(traffic)
	if err != nil {
		r.logger.Errorw("failed to map user traffic entity to model", "error", err)
		return fmt.Errorf("failed to map user traffic entity: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create user traffic", "user_id", model.UserID, "node_id", model.NodeID, "error", err)
		return fmt.Errorf("failed to create user traffic: %w", err)
	}

	if err := traffic.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set user traffic ID", "error", err)
		return fmt.Errorf("failed to set user traffic ID: %w", err)
	}

	r.logger.Infow("user traffic created successfully", "id", model.ID, "user_id", model.UserID, "node_id", model.NodeID)
	return nil
}

// BatchUpsert batch inserts or updates user traffic records
// This is optimized for XrayR traffic reporting where we need to handle multiple users at once
func (r *UserTrafficRepositoryImpl) BatchUpsert(ctx context.Context, traffics []*node.UserTraffic) error {
	if len(traffics) == 0 {
		return nil
	}

	models, err := r.mapper.ToModels(traffics)
	if err != nil {
		r.logger.Errorw("failed to map user traffic entities to models", "error", err)
		return fmt.Errorf("failed to map user traffic entities: %w", err)
	}

	// Use GORM's Clauses with OnConflict for upsert behavior
	// On conflict (user_id, node_id, period), update upload, download, total
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "node_id"},
			{Name: "period"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"upload":   gorm.Expr("upload + VALUES(upload)"),
			"download": gorm.Expr("download + VALUES(download)"),
			"total":    gorm.Expr("total + VALUES(total)"),
		}),
	}).Create(&models)

	if result.Error != nil {
		r.logger.Errorw("failed to batch upsert user traffic", "count", len(models), "error", result.Error)
		return fmt.Errorf("failed to batch upsert user traffic: %w", result.Error)
	}

	r.logger.Infow("user traffic batch upserted successfully", "count", len(models))
	return nil
}

// GetByUserAndNode retrieves user traffic by user ID, node ID, and period
func (r *UserTrafficRepositoryImpl) GetByUserAndNode(ctx context.Context, userID, nodeID uint, period time.Time) (*node.UserTraffic, error) {
	var model models.UserTrafficModel

	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND node_id = ? AND period = ?", userID, nodeID, period).
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("user traffic not found")
		}
		r.logger.Errorw("failed to get user traffic", "user_id", userID, "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to get user traffic: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map user traffic model to entity", "id", model.ID, "error", err)
		return nil, fmt.Errorf("failed to map user traffic: %w", err)
	}

	return entity, nil
}

// GetByUserIDWithDateRange retrieves all traffic records for a user within a date range
func (r *UserTrafficRepositoryImpl) GetByUserIDWithDateRange(ctx context.Context, userID uint, start, end time.Time) ([]*node.UserTraffic, error) {
	query := r.db.WithContext(ctx).Model(&models.UserTrafficModel{}).
		Where("user_id = ?", userID)

	if !start.IsZero() {
		query = query.Where("period >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("period <= ?", end)
	}

	var trafficModels []*models.UserTrafficModel
	if err := query.Order("period DESC").Find(&trafficModels).Error; err != nil {
		r.logger.Errorw("failed to get user traffic by date range", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get user traffic: %w", err)
	}

	entities, err := r.mapper.ToEntities(trafficModels)
	if err != nil {
		r.logger.Errorw("failed to map user traffic models to entities", "error", err)
		return nil, fmt.Errorf("failed to map user traffic: %w", err)
	}

	return entities, nil
}

// GetTotalByUser calculates total traffic for a user across all nodes
func (r *UserTrafficRepositoryImpl) GetTotalByUser(ctx context.Context, userID uint) (upload uint64, download uint64, total uint64, err error) {
	var result struct {
		TotalUpload   uint64
		TotalDownload uint64
		TotalTraffic  uint64
	}

	err = r.db.WithContext(ctx).Model(&models.UserTrafficModel{}).
		Select("COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_traffic").
		Where("user_id = ?", userID).
		Scan(&result).Error

	if err != nil {
		r.logger.Errorw("failed to get total traffic by user", "user_id", userID, "error", err)
		return 0, 0, 0, fmt.Errorf("failed to get total traffic: %w", err)
	}

	return result.TotalUpload, result.TotalDownload, result.TotalTraffic, nil
}

// GetTotalBySubscription calculates total traffic for a subscription across all nodes
func (r *UserTrafficRepositoryImpl) GetTotalBySubscription(ctx context.Context, subscriptionID uint) (upload uint64, download uint64, total uint64, err error) {
	var result struct {
		TotalUpload   uint64
		TotalDownload uint64
		TotalTraffic  uint64
	}

	err = r.db.WithContext(ctx).Model(&models.UserTrafficModel{}).
		Select("COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download, COALESCE(SUM(total), 0) as total_traffic").
		Where("subscription_id = ?", subscriptionID).
		Scan(&result).Error

	if err != nil {
		r.logger.Errorw("failed to get total traffic by subscription", "subscription_id", subscriptionID, "error", err)
		return 0, 0, 0, fmt.Errorf("failed to get total traffic: %w", err)
	}

	return result.TotalUpload, result.TotalDownload, result.TotalTraffic, nil
}

// IncrementTraffic increments traffic for a user on a specific node (atomic operation)
// This is used for real-time traffic updates from XrayR
func (r *UserTrafficRepositoryImpl) IncrementTraffic(ctx context.Context, userID, nodeID uint, upload, download uint64) error {
	if upload == 0 && download == 0 {
		return nil
	}

	// Use current hour as the period for granular tracking
	period := time.Now().Truncate(time.Hour)

	// Perform atomic increment using raw SQL
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO user_traffic (user_id, node_id, upload, download, total, period, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			upload = upload + VALUES(upload),
			download = download + VALUES(download),
			total = total + VALUES(total),
			updated_at = NOW()
	`, userID, nodeID, upload, download, upload+download, period)

	if result.Error != nil {
		r.logger.Errorw("failed to increment user traffic", "user_id", userID, "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to increment user traffic: %w", result.Error)
	}

	r.logger.Debugw("user traffic incremented", "user_id", userID, "node_id", nodeID, "upload", upload, "download", download)
	return nil
}

// DeleteOldRecords deletes traffic records older than the specified time
func (r *UserTrafficRepositoryImpl) DeleteOldRecords(ctx context.Context, before time.Time) error {
	result := r.db.WithContext(ctx).Where("period < ?", before).Delete(&models.UserTrafficModel{})
	if result.Error != nil {
		r.logger.Errorw("failed to delete old user traffic records", "before", before, "error", result.Error)
		return fmt.Errorf("failed to delete old user traffic records: %w", result.Error)
	}

	r.logger.Infow("old user traffic records deleted successfully", "before", before, "deleted_count", result.RowsAffected)
	return nil
}

// GetTopUsers retrieves top users by traffic usage within a time range
func (r *UserTrafficRepositoryImpl) GetTopUsers(ctx context.Context, limit int, from, to time.Time) ([]*node.UserTraffic, error) {
	query := r.db.WithContext(ctx).Model(&models.UserTrafficModel{})

	if !from.IsZero() {
		query = query.Where("period >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("period <= ?", to)
	}

	// Group by user_id and sum the traffic, then order by total descending
	var trafficModels []*models.UserTrafficModel
	err := query.
		Select("user_id, SUM(upload) as upload, SUM(download) as download, SUM(total) as total").
		Group("user_id").
		Order("total DESC").
		Limit(limit).
		Find(&trafficModels).Error

	if err != nil {
		r.logger.Errorw("failed to get top users by traffic", "error", err)
		return nil, fmt.Errorf("failed to get top users: %w", err)
	}

	entities, err := r.mapper.ToEntities(trafficModels)
	if err != nil {
		r.logger.Errorw("failed to map user traffic models to entities", "error", err)
		return nil, fmt.Errorf("failed to map user traffic: %w", err)
	}

	return entities, nil
}
