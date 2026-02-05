package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AdminTelegramBindingModel is the GORM model for admin_telegram_bindings table
type AdminTelegramBindingModel struct {
	ID                             uint       `gorm:"primaryKey;autoIncrement"`
	SID                            string     `gorm:"column:sid;type:varchar(50);not null;uniqueIndex"`
	UserID                         uint       `gorm:"column:user_id;not null;uniqueIndex"`
	TelegramUserID                 int64      `gorm:"column:telegram_user_id;not null;uniqueIndex"`
	TelegramUsername               string     `gorm:"column:telegram_username;type:varchar(100)"`
	NotifyNodeOffline              bool       `gorm:"column:notify_node_offline;default:true"`
	NotifyAgentOffline             bool       `gorm:"column:notify_agent_offline;default:true"`
	NotifyNewUser                  bool       `gorm:"column:notify_new_user;default:true"`
	NotifyPaymentSuccess           bool       `gorm:"column:notify_payment_success;default:true"`
	NotifyDailySummary             bool       `gorm:"column:notify_daily_summary;default:true"`
	NotifyWeeklySummary            bool       `gorm:"column:notify_weekly_summary;default:true"`
	OfflineThresholdMinutes        int        `gorm:"column:offline_threshold_minutes;default:5"`
	NotifyResourceExpiring         bool       `gorm:"column:notify_resource_expiring;default:true"`
	ResourceExpiringDays           int        `gorm:"column:resource_expiring_days;default:7"`
	DailySummaryHour               int        `gorm:"column:daily_summary_hour;default:9"`
	WeeklySummaryHour              int        `gorm:"column:weekly_summary_hour;default:9"`
	WeeklySummaryWeekday           int        `gorm:"column:weekly_summary_weekday;default:1"`
	OfflineCheckIntervalMinutes    int        `gorm:"column:offline_check_interval_minutes;default:5"`
	LastNodeOfflineNotifyAt        *time.Time `gorm:"column:last_node_offline_notify_at"`
	LastAgentOfflineNotifyAt       *time.Time `gorm:"column:last_agent_offline_notify_at"`
	LastDailySummaryAt             *time.Time `gorm:"column:last_daily_summary_at"`
	LastWeeklySummaryAt            *time.Time `gorm:"column:last_weekly_summary_at"`
	LastResourceExpiringNotifyDate *time.Time `gorm:"column:last_resource_expiring_notify_date"`
	CreatedAt                      time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt                      time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the table name for GORM
func (AdminTelegramBindingModel) TableName() string {
	return "admin_telegram_bindings"
}

// AdminTelegramBindingRepository implements admin.AdminTelegramBindingRepository
type AdminTelegramBindingRepository struct {
	db     *gorm.DB
	logger logger.Interface
}

// NewAdminTelegramBindingRepository creates a new AdminTelegramBindingRepository
func NewAdminTelegramBindingRepository(db *gorm.DB, logger logger.Interface) *AdminTelegramBindingRepository {
	return &AdminTelegramBindingRepository{db: db, logger: logger}
}

// Create creates a new admin telegram binding
func (r *AdminTelegramBindingRepository) Create(ctx context.Context, binding *admin.AdminTelegramBinding) error {
	model := r.toModel(binding)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	binding.SetID(model.ID)
	return nil
}

// GetByID retrieves a binding by ID
func (r *AdminTelegramBindingRepository) GetByID(ctx context.Context, id uint) (*admin.AdminTelegramBinding, error) {
	var model AdminTelegramBindingModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, admin.ErrBindingNotFound
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

// GetBySID retrieves a binding by SID
func (r *AdminTelegramBindingRepository) GetBySID(ctx context.Context, sid string) (*admin.AdminTelegramBinding, error) {
	var model AdminTelegramBindingModel
	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, admin.ErrBindingNotFound
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

// GetByUserID retrieves a binding by user ID
func (r *AdminTelegramBindingRepository) GetByUserID(ctx context.Context, userID uint) (*admin.AdminTelegramBinding, error) {
	var model AdminTelegramBindingModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, admin.ErrBindingNotFound
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

// GetByTelegramUserID retrieves a binding by Telegram user ID
func (r *AdminTelegramBindingRepository) GetByTelegramUserID(ctx context.Context, telegramUserID int64) (*admin.AdminTelegramBinding, error) {
	var model AdminTelegramBindingModel
	if err := r.db.WithContext(ctx).Where("telegram_user_id = ?", telegramUserID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, admin.ErrBindingNotFound
		}
		return nil, err
	}
	return r.toDomain(&model), nil
}

// Update updates an admin telegram binding
func (r *AdminTelegramBindingRepository) Update(ctx context.Context, binding *admin.AdminTelegramBinding) error {
	model := r.toModel(binding)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete deletes an admin telegram binding by ID
func (r *AdminTelegramBindingRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&AdminTelegramBindingModel{}, id).Error
}

// GetAll retrieves all admin telegram bindings
func (r *AdminTelegramBindingRepository) GetAll(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var models []AdminTelegramBindingModel
	if err := r.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(models))
	for _, model := range models {
		bindings = append(bindings, r.toDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForNodeOfflineNotification finds bindings that want node offline notifications
func (r *AdminTelegramBindingRepository) FindBindingsForNodeOfflineNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var models []AdminTelegramBindingModel
	err := r.db.WithContext(ctx).
		Where("notify_node_offline = ?", true).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(models))
	for _, model := range models {
		bindings = append(bindings, r.toDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForNodeOnlineNotification finds bindings that want node online notifications
// Unlike offline notifications, online notifications don't have deduplication (no 24h threshold)
func (r *AdminTelegramBindingRepository) FindBindingsForNodeOnlineNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var models []AdminTelegramBindingModel
	err := r.db.WithContext(ctx).
		Where("notify_node_offline = ?", true). // reuse same preference
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(models))
	for _, model := range models {
		bindings = append(bindings, r.toDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForAgentOfflineNotification finds bindings that want agent offline notifications
func (r *AdminTelegramBindingRepository) FindBindingsForAgentOfflineNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var models []AdminTelegramBindingModel
	err := r.db.WithContext(ctx).
		Where("notify_agent_offline = ?", true).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(models))
	for _, model := range models {
		bindings = append(bindings, r.toDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForAgentOnlineNotification finds bindings that want agent online notifications
// Unlike offline notifications, online notifications don't have deduplication (no 24h threshold)
func (r *AdminTelegramBindingRepository) FindBindingsForAgentOnlineNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var models []AdminTelegramBindingModel
	err := r.db.WithContext(ctx).
		Where("notify_agent_offline = ?", true). // reuse same preference
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(models))
	for _, model := range models {
		bindings = append(bindings, r.toDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForNewUserNotification finds bindings that want new user notifications
func (r *AdminTelegramBindingRepository) FindBindingsForNewUserNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var models []AdminTelegramBindingModel
	err := r.db.WithContext(ctx).
		Where("notify_new_user = ?", true).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(models))
	for _, model := range models {
		bindings = append(bindings, r.toDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForPaymentSuccessNotification finds bindings that want payment success notifications
func (r *AdminTelegramBindingRepository) FindBindingsForPaymentSuccessNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var models []AdminTelegramBindingModel
	err := r.db.WithContext(ctx).
		Where("notify_payment_success = ?", true).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(models))
	for _, model := range models {
		bindings = append(bindings, r.toDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForDailySummary finds bindings that want daily summary at the given business hour
func (r *AdminTelegramBindingRepository) FindBindingsForDailySummary(ctx context.Context, bizHour int) ([]*admin.AdminTelegramBinding, error) {
	var models []AdminTelegramBindingModel

	err := r.db.WithContext(ctx).
		Where("notify_daily_summary = ? AND daily_summary_hour = ?", true, bizHour).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(models))
	for _, model := range models {
		bindings = append(bindings, r.toDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForWeeklySummary finds bindings that want weekly summary at the given business hour and weekday
func (r *AdminTelegramBindingRepository) FindBindingsForWeeklySummary(ctx context.Context, bizHour int, bizWeekday int) ([]*admin.AdminTelegramBinding, error) {
	var models []AdminTelegramBindingModel

	err := r.db.WithContext(ctx).
		Where("notify_weekly_summary = ? AND weekly_summary_hour = ? AND weekly_summary_weekday = ?", true, bizHour, bizWeekday).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(models))
	for _, model := range models {
		bindings = append(bindings, r.toDomain(&model))
	}
	return bindings, nil
}

// toModel converts domain entity to GORM model
func (r *AdminTelegramBindingRepository) toModel(binding *admin.AdminTelegramBinding) *AdminTelegramBindingModel {
	return &AdminTelegramBindingModel{
		ID:                             binding.ID(),
		SID:                            binding.SID(),
		UserID:                         binding.UserID(),
		TelegramUserID:                 binding.TelegramUserID(),
		TelegramUsername:               binding.TelegramUsername(),
		NotifyNodeOffline:              binding.NotifyNodeOffline(),
		NotifyAgentOffline:             binding.NotifyAgentOffline(),
		NotifyNewUser:                  binding.NotifyNewUser(),
		NotifyPaymentSuccess:           binding.NotifyPaymentSuccess(),
		NotifyDailySummary:             binding.NotifyDailySummary(),
		NotifyWeeklySummary:            binding.NotifyWeeklySummary(),
		OfflineThresholdMinutes:        binding.OfflineThresholdMinutes(),
		NotifyResourceExpiring:         binding.NotifyResourceExpiring(),
		ResourceExpiringDays:           binding.ResourceExpiringDays(),
		DailySummaryHour:               binding.DailySummaryHour(),
		WeeklySummaryHour:              binding.WeeklySummaryHour(),
		WeeklySummaryWeekday:           binding.WeeklySummaryWeekday(),
		OfflineCheckIntervalMinutes:    binding.OfflineCheckIntervalMinutes(),
		LastNodeOfflineNotifyAt:        binding.LastNodeOfflineNotifyAt(),
		LastAgentOfflineNotifyAt:       binding.LastAgentOfflineNotifyAt(),
		LastDailySummaryAt:             binding.LastDailySummaryAt(),
		LastWeeklySummaryAt:            binding.LastWeeklySummaryAt(),
		LastResourceExpiringNotifyDate: binding.LastResourceExpiringNotifyDate(),
		CreatedAt:                      binding.CreatedAt(),
		UpdatedAt:                      binding.UpdatedAt(),
	}
}

// toDomain converts GORM model to domain entity
func (r *AdminTelegramBindingRepository) toDomain(model *AdminTelegramBindingModel) *admin.AdminTelegramBinding {
	return admin.ReconstructAdminTelegramBinding(
		model.ID,
		model.SID,
		model.UserID,
		model.TelegramUserID,
		model.TelegramUsername,
		model.NotifyNodeOffline,
		model.NotifyAgentOffline,
		model.NotifyNewUser,
		model.NotifyPaymentSuccess,
		model.NotifyDailySummary,
		model.NotifyWeeklySummary,
		model.OfflineThresholdMinutes,
		model.NotifyResourceExpiring,
		model.ResourceExpiringDays,
		model.DailySummaryHour,
		model.WeeklySummaryHour,
		model.WeeklySummaryWeekday,
		model.OfflineCheckIntervalMinutes,
		model.LastNodeOfflineNotifyAt,
		model.LastAgentOfflineNotifyAt,
		model.LastDailySummaryAt,
		model.LastWeeklySummaryAt,
		model.LastResourceExpiringNotifyDate,
		model.CreatedAt,
		model.UpdatedAt,
	)
}

// FindBindingsForResourceExpiringNotification finds bindings that want resource expiring notifications
// and haven't been notified today
func (r *AdminTelegramBindingRepository) FindBindingsForResourceExpiringNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var models []AdminTelegramBindingModel
	today := biztime.NowUTC().Truncate(24 * time.Hour)

	err := r.db.WithContext(ctx).
		Where("notify_resource_expiring = ?", true).
		Where("last_resource_expiring_notify_date IS NULL OR last_resource_expiring_notify_date < ?", today).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(models))
	for _, model := range models {
		bindings = append(bindings, r.toDomain(&model))
	}
	return bindings, nil
}
