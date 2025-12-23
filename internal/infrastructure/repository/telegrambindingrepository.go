package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/telegram"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// TelegramBindingModel is the GORM model for telegram_bindings table
type TelegramBindingModel struct {
	ID                   uint       `gorm:"primaryKey;autoIncrement"`
	SID                  string     `gorm:"column:sid;type:varchar(50);not null;uniqueIndex"`
	UserID               uint       `gorm:"column:user_id;not null;uniqueIndex"`
	TelegramUserID       int64      `gorm:"column:telegram_user_id;not null;uniqueIndex"`
	TelegramUsername     string     `gorm:"column:telegram_username;type:varchar(100)"`
	NotifyExpiring       bool       `gorm:"column:notify_expiring;default:true"`
	NotifyTraffic        bool       `gorm:"column:notify_traffic;default:true"`
	ExpiringDays         int        `gorm:"column:expiring_days;default:3"`
	TrafficThreshold     int        `gorm:"column:traffic_threshold;default:80"`
	LastExpiringNotifyAt *time.Time `gorm:"column:last_expiring_notify_at"`
	LastTrafficNotifyAt  *time.Time `gorm:"column:last_traffic_notify_at"`
	CreatedAt            time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt            time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the table name for GORM
func (TelegramBindingModel) TableName() string {
	return "telegram_bindings"
}

// TelegramBindingRepository implements telegram.TelegramBindingRepository
type TelegramBindingRepository struct {
	db     *gorm.DB
	logger logger.Interface
}

// NewTelegramBindingRepository creates a new TelegramBindingRepository
func NewTelegramBindingRepository(db *gorm.DB, logger logger.Interface) *TelegramBindingRepository {
	return &TelegramBindingRepository{db: db, logger: logger}
}

// Create creates a new telegram binding
func (r *TelegramBindingRepository) Create(ctx context.Context, binding *telegram.TelegramBinding) error {
	model := r.toModel(binding)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	binding.SetID(model.ID)
	return nil
}

// GetByID retrieves a binding by ID
func (r *TelegramBindingRepository) GetByID(ctx context.Context, id uint) (*telegram.TelegramBinding, error) {
	var model TelegramBindingModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, telegram.ErrBindingNotFound
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

// GetBySID retrieves a binding by SID
func (r *TelegramBindingRepository) GetBySID(ctx context.Context, sid string) (*telegram.TelegramBinding, error) {
	var model TelegramBindingModel
	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, telegram.ErrBindingNotFound
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

// GetByUserID retrieves a binding by user ID
func (r *TelegramBindingRepository) GetByUserID(ctx context.Context, userID uint) (*telegram.TelegramBinding, error) {
	var model TelegramBindingModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, telegram.ErrBindingNotFound
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

// GetByTelegramUserID retrieves a binding by Telegram user ID
func (r *TelegramBindingRepository) GetByTelegramUserID(ctx context.Context, telegramUserID int64) (*telegram.TelegramBinding, error) {
	var model TelegramBindingModel
	if err := r.db.WithContext(ctx).Where("telegram_user_id = ?", telegramUserID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, telegram.ErrBindingNotFound
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

// Update updates a telegram binding
func (r *TelegramBindingRepository) Update(ctx context.Context, binding *telegram.TelegramBinding) error {
	model := r.toModel(binding)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete deletes a telegram binding by ID
func (r *TelegramBindingRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&TelegramBindingModel{}, id).Error
}

// FindBindingsForExpiringNotification finds bindings that need expiring notifications
func (r *TelegramBindingRepository) FindBindingsForExpiringNotification(ctx context.Context) ([]*telegram.TelegramBinding, error) {
	var models []TelegramBindingModel
	threshold := time.Now().Add(-24 * time.Hour)

	err := r.db.WithContext(ctx).
		Where("notify_expiring = ?", true).
		Where("last_expiring_notify_at IS NULL OR last_expiring_notify_at < ?", threshold).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*telegram.TelegramBinding, 0, len(models))
	for _, model := range models {
		bindings = append(bindings, r.toDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForTrafficNotification finds bindings that need traffic notifications
func (r *TelegramBindingRepository) FindBindingsForTrafficNotification(ctx context.Context) ([]*telegram.TelegramBinding, error) {
	var models []TelegramBindingModel
	threshold := time.Now().Add(-24 * time.Hour)

	err := r.db.WithContext(ctx).
		Where("notify_traffic = ?", true).
		Where("last_traffic_notify_at IS NULL OR last_traffic_notify_at < ?", threshold).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*telegram.TelegramBinding, 0, len(models))
	for _, model := range models {
		bindings = append(bindings, r.toDomain(&model))
	}
	return bindings, nil
}

// GetUserIDsWithBinding returns user IDs that have telegram binding
func (r *TelegramBindingRepository) GetUserIDsWithBinding(ctx context.Context) ([]uint, error) {
	var userIDs []uint
	err := r.db.WithContext(ctx).
		Model(&TelegramBindingModel{}).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

// toModel converts domain entity to GORM model
func (r *TelegramBindingRepository) toModel(binding *telegram.TelegramBinding) *TelegramBindingModel {
	return &TelegramBindingModel{
		ID:                   binding.ID(),
		SID:                  binding.SID(),
		UserID:               binding.UserID(),
		TelegramUserID:       binding.TelegramUserID(),
		TelegramUsername:     binding.TelegramUsername(),
		NotifyExpiring:       binding.NotifyExpiring(),
		NotifyTraffic:        binding.NotifyTraffic(),
		ExpiringDays:         binding.ExpiringDays(),
		TrafficThreshold:     binding.TrafficThreshold(),
		LastExpiringNotifyAt: binding.LastExpiringNotifyAt(),
		LastTrafficNotifyAt:  binding.LastTrafficNotifyAt(),
		CreatedAt:            binding.CreatedAt(),
		UpdatedAt:            binding.UpdatedAt(),
	}
}

// toDomain converts GORM model to domain entity
func (r *TelegramBindingRepository) toDomain(model *TelegramBindingModel) *telegram.TelegramBinding {
	return telegram.ReconstructTelegramBinding(
		model.ID,
		model.SID,
		model.UserID,
		model.TelegramUserID,
		model.TelegramUsername,
		model.NotifyExpiring,
		model.NotifyTraffic,
		model.ExpiringDays,
		model.TrafficThreshold,
		model.LastExpiringNotifyAt,
		model.LastTrafficNotifyAt,
		model.CreatedAt,
		model.UpdatedAt,
	)
}
