package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/orris-inc/orris/internal/domain/setting"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SystemSettingRepository implements setting.Repository
type SystemSettingRepository struct {
	db     *gorm.DB
	logger logger.Interface
	mapper mappers.SystemSettingMapper
}

// NewSystemSettingRepository creates a new SystemSettingRepository
func NewSystemSettingRepository(db *gorm.DB, logger logger.Interface) setting.Repository {
	return &SystemSettingRepository{
		db:     db,
		logger: logger,
		mapper: mappers.NewSystemSettingMapper(),
	}
}

// GetByKey retrieves a setting by category and key
func (r *SystemSettingRepository) GetByKey(ctx context.Context, category, key string) (*setting.SystemSetting, error) {
	var model models.SystemSettingModel

	err := r.db.WithContext(ctx).
		Where("category = ? AND setting_key = ?", category, key).
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, setting.ErrSettingNotFound
		}
		r.logger.Error("failed to get setting by key", "category", category, "key", key, "error", err)
		return nil, fmt.Errorf("failed to get setting by key: %w", err)
	}

	return r.mapper.ToDomain(&model), nil
}

// GetByCategory retrieves all settings in a category
func (r *SystemSettingRepository) GetByCategory(ctx context.Context, category string) ([]*setting.SystemSetting, error) {
	var modelList []*models.SystemSettingModel

	err := r.db.WithContext(ctx).
		Where("category = ?", category).
		Order("setting_key ASC").
		Find(&modelList).Error
	if err != nil {
		r.logger.Error("failed to get settings by category", "category", category, "error", err)
		return nil, fmt.Errorf("failed to get settings by category: %w", err)
	}

	return r.mapper.ToDomainList(modelList), nil
}

// GetAll retrieves all system settings
func (r *SystemSettingRepository) GetAll(ctx context.Context) ([]*setting.SystemSetting, error) {
	var modelList []*models.SystemSettingModel

	err := r.db.WithContext(ctx).
		Order("category ASC, setting_key ASC").
		Find(&modelList).Error
	if err != nil {
		r.logger.Error("failed to get all settings", "error", err)
		return nil, fmt.Errorf("failed to get all settings: %w", err)
	}

	return r.mapper.ToDomainList(modelList), nil
}

// Upsert creates or updates a setting
func (r *SystemSettingRepository) Upsert(ctx context.Context, s *setting.SystemSetting) error {
	model := r.mapper.ToModel(s)

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "category"}, {Name: "setting_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "value_type", "description", "updated_by", "version", "updated_at"}),
	}).Create(model).Error
	if err != nil {
		r.logger.Error("failed to upsert setting", "category", s.Category(), "key", s.Key(), "error", err)
		return fmt.Errorf("failed to upsert setting: %w", err)
	}

	// Update the domain entity with the generated ID if it was an insert
	if s.ID() == 0 {
		s.SetID(model.ID)
	}

	return nil
}

// Delete removes a setting by category and key
func (r *SystemSettingRepository) Delete(ctx context.Context, category, key string) error {
	result := r.db.WithContext(ctx).
		Where("category = ? AND setting_key = ?", category, key).
		Delete(&models.SystemSettingModel{})
	if result.Error != nil {
		r.logger.Error("failed to delete setting", "category", category, "key", key, "error", result.Error)
		return fmt.Errorf("failed to delete setting: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return setting.ErrSettingNotFound
	}

	return nil
}
