package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/telegram"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// TelegramBindingRepository implements telegram.TelegramBindingRepository
type TelegramBindingRepository struct {
	db     *gorm.DB
	logger logger.Interface
	mapper mappers.TelegramBindingMapper
}

// NewTelegramBindingRepository creates a new TelegramBindingRepository
func NewTelegramBindingRepository(db *gorm.DB, logger logger.Interface) *TelegramBindingRepository {
	return &TelegramBindingRepository{
		db:     db,
		logger: logger,
		mapper: mappers.NewTelegramBindingMapper(),
	}
}

// Create creates a new telegram binding
func (r *TelegramBindingRepository) Create(ctx context.Context, binding *telegram.TelegramBinding) error {
	model := r.mapper.ToModel(binding)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	binding.SetID(model.ID)
	return nil
}

// GetByID retrieves a binding by ID
func (r *TelegramBindingRepository) GetByID(ctx context.Context, id uint) (*telegram.TelegramBinding, error) {
	var model models.TelegramBindingModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, telegram.ErrBindingNotFound
		}
		return nil, err
	}
	return r.mapper.ToDomain(&model), nil
}

// GetBySID retrieves a binding by SID
func (r *TelegramBindingRepository) GetBySID(ctx context.Context, sid string) (*telegram.TelegramBinding, error) {
	var model models.TelegramBindingModel
	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, telegram.ErrBindingNotFound
		}
		return nil, err
	}
	return r.mapper.ToDomain(&model), nil
}

// GetByUserID retrieves a binding by user ID
func (r *TelegramBindingRepository) GetByUserID(ctx context.Context, userID uint) (*telegram.TelegramBinding, error) {
	var model models.TelegramBindingModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, telegram.ErrBindingNotFound
		}
		return nil, err
	}
	return r.mapper.ToDomain(&model), nil
}

// GetByTelegramUserID retrieves a binding by Telegram user ID
func (r *TelegramBindingRepository) GetByTelegramUserID(ctx context.Context, telegramUserID int64) (*telegram.TelegramBinding, error) {
	var model models.TelegramBindingModel
	if err := r.db.WithContext(ctx).Where("telegram_user_id = ?", telegramUserID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, telegram.ErrBindingNotFound
		}
		return nil, err
	}
	return r.mapper.ToDomain(&model), nil
}

// Update updates a telegram binding
func (r *TelegramBindingRepository) Update(ctx context.Context, binding *telegram.TelegramBinding) error {
	model := r.mapper.ToModel(binding)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete deletes a telegram binding by ID
func (r *TelegramBindingRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.TelegramBindingModel{}, id).Error
}

// FindBindingsForExpiringNotification finds bindings that need expiring notifications
func (r *TelegramBindingRepository) FindBindingsForExpiringNotification(ctx context.Context) ([]*telegram.TelegramBinding, error) {
	var dbModels []models.TelegramBindingModel
	threshold := biztime.NowUTC().Add(-24 * time.Hour)

	err := r.db.WithContext(ctx).
		Where("notify_expiring = ?", true).
		Where("last_expiring_notify_at IS NULL OR last_expiring_notify_at < ?", threshold).
		Find(&dbModels).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*telegram.TelegramBinding, 0, len(dbModels))
	for _, model := range dbModels {
		bindings = append(bindings, r.mapper.ToDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForTrafficNotification finds bindings that need traffic notifications
func (r *TelegramBindingRepository) FindBindingsForTrafficNotification(ctx context.Context) ([]*telegram.TelegramBinding, error) {
	var dbModels []models.TelegramBindingModel
	threshold := biztime.NowUTC().Add(-24 * time.Hour)

	err := r.db.WithContext(ctx).
		Where("notify_traffic = ?", true).
		Where("last_traffic_notify_at IS NULL OR last_traffic_notify_at < ?", threshold).
		Find(&dbModels).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*telegram.TelegramBinding, 0, len(dbModels))
	for _, model := range dbModels {
		bindings = append(bindings, r.mapper.ToDomain(&model))
	}
	return bindings, nil
}

// GetUserIDsWithBinding returns user IDs that have telegram binding
func (r *TelegramBindingRepository) GetUserIDsWithBinding(ctx context.Context) ([]uint, error) {
	var userIDs []uint
	err := r.db.WithContext(ctx).
		Model(&models.TelegramBindingModel{}).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}
