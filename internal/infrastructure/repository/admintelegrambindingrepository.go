package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AdminTelegramBindingRepository implements admin.AdminTelegramBindingRepository
type AdminTelegramBindingRepository struct {
	db     *gorm.DB
	logger logger.Interface
	mapper mappers.AdminTelegramBindingMapper
}

// NewAdminTelegramBindingRepository creates a new AdminTelegramBindingRepository
func NewAdminTelegramBindingRepository(db *gorm.DB, logger logger.Interface) *AdminTelegramBindingRepository {
	return &AdminTelegramBindingRepository{
		db:     db,
		logger: logger,
		mapper: mappers.NewAdminTelegramBindingMapper(),
	}
}

// Create creates a new admin telegram binding
func (r *AdminTelegramBindingRepository) Create(ctx context.Context, binding *admin.AdminTelegramBinding) error {
	model := r.mapper.ToModel(binding)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	binding.SetID(model.ID)
	return nil
}

// GetByID retrieves a binding by ID
func (r *AdminTelegramBindingRepository) GetByID(ctx context.Context, id uint) (*admin.AdminTelegramBinding, error) {
	var model models.AdminTelegramBindingModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, admin.ErrBindingNotFound
		}
		return nil, err
	}
	return r.mapper.ToDomain(&model), nil
}

// GetBySID retrieves a binding by SID
func (r *AdminTelegramBindingRepository) GetBySID(ctx context.Context, sid string) (*admin.AdminTelegramBinding, error) {
	var model models.AdminTelegramBindingModel
	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, admin.ErrBindingNotFound
		}
		return nil, err
	}
	return r.mapper.ToDomain(&model), nil
}

// GetByUserID retrieves a binding by user ID
func (r *AdminTelegramBindingRepository) GetByUserID(ctx context.Context, userID uint) (*admin.AdminTelegramBinding, error) {
	var model models.AdminTelegramBindingModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, admin.ErrBindingNotFound
		}
		return nil, err
	}
	return r.mapper.ToDomain(&model), nil
}

// GetByTelegramUserID retrieves a binding by Telegram user ID
func (r *AdminTelegramBindingRepository) GetByTelegramUserID(ctx context.Context, telegramUserID int64) (*admin.AdminTelegramBinding, error) {
	var model models.AdminTelegramBindingModel
	if err := r.db.WithContext(ctx).Where("telegram_user_id = ?", telegramUserID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, admin.ErrBindingNotFound
		}
		return nil, err
	}
	return r.mapper.ToDomain(&model), nil
}

// Update updates an admin telegram binding
func (r *AdminTelegramBindingRepository) Update(ctx context.Context, binding *admin.AdminTelegramBinding) error {
	model := r.mapper.ToModel(binding)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete deletes an admin telegram binding by ID
func (r *AdminTelegramBindingRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.AdminTelegramBindingModel{}, id).Error
}

// GetAll retrieves all admin telegram bindings
func (r *AdminTelegramBindingRepository) GetAll(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var dbModels []models.AdminTelegramBindingModel
	if err := r.db.WithContext(ctx).Find(&dbModels).Error; err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(dbModels))
	for _, model := range dbModels {
		bindings = append(bindings, r.mapper.ToDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForNodeOfflineNotification finds bindings that want node offline notifications
func (r *AdminTelegramBindingRepository) FindBindingsForNodeOfflineNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var dbModels []models.AdminTelegramBindingModel
	err := r.db.WithContext(ctx).
		Where("notify_node_offline = ?", true).
		Find(&dbModels).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(dbModels))
	for _, model := range dbModels {
		bindings = append(bindings, r.mapper.ToDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForNodeOnlineNotification finds bindings that want node online notifications
// Unlike offline notifications, online notifications don't have deduplication (no 24h threshold)
func (r *AdminTelegramBindingRepository) FindBindingsForNodeOnlineNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var dbModels []models.AdminTelegramBindingModel
	err := r.db.WithContext(ctx).
		Where("notify_node_offline = ?", true). // reuse same preference
		Find(&dbModels).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(dbModels))
	for _, model := range dbModels {
		bindings = append(bindings, r.mapper.ToDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForAgentOfflineNotification finds bindings that want agent offline notifications
func (r *AdminTelegramBindingRepository) FindBindingsForAgentOfflineNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var dbModels []models.AdminTelegramBindingModel
	err := r.db.WithContext(ctx).
		Where("notify_agent_offline = ?", true).
		Find(&dbModels).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(dbModels))
	for _, model := range dbModels {
		bindings = append(bindings, r.mapper.ToDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForAgentOnlineNotification finds bindings that want agent online notifications
// Unlike offline notifications, online notifications don't have deduplication (no 24h threshold)
func (r *AdminTelegramBindingRepository) FindBindingsForAgentOnlineNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var dbModels []models.AdminTelegramBindingModel
	err := r.db.WithContext(ctx).
		Where("notify_agent_offline = ?", true). // reuse same preference
		Find(&dbModels).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(dbModels))
	for _, model := range dbModels {
		bindings = append(bindings, r.mapper.ToDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForNewUserNotification finds bindings that want new user notifications
func (r *AdminTelegramBindingRepository) FindBindingsForNewUserNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var dbModels []models.AdminTelegramBindingModel
	err := r.db.WithContext(ctx).
		Where("notify_new_user = ?", true).
		Find(&dbModels).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(dbModels))
	for _, model := range dbModels {
		bindings = append(bindings, r.mapper.ToDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForPaymentSuccessNotification finds bindings that want payment success notifications
func (r *AdminTelegramBindingRepository) FindBindingsForPaymentSuccessNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var dbModels []models.AdminTelegramBindingModel
	err := r.db.WithContext(ctx).
		Where("notify_payment_success = ?", true).
		Find(&dbModels).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(dbModels))
	for _, model := range dbModels {
		bindings = append(bindings, r.mapper.ToDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForDailySummary finds bindings that want daily summary at the given business hour
func (r *AdminTelegramBindingRepository) FindBindingsForDailySummary(ctx context.Context, bizHour int) ([]*admin.AdminTelegramBinding, error) {
	var dbModels []models.AdminTelegramBindingModel

	err := r.db.WithContext(ctx).
		Where("notify_daily_summary = ? AND daily_summary_hour = ?", true, bizHour).
		Find(&dbModels).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(dbModels))
	for _, model := range dbModels {
		bindings = append(bindings, r.mapper.ToDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForWeeklySummary finds bindings that want weekly summary at the given business hour and weekday
func (r *AdminTelegramBindingRepository) FindBindingsForWeeklySummary(ctx context.Context, bizHour int, bizWeekday int) ([]*admin.AdminTelegramBinding, error) {
	var dbModels []models.AdminTelegramBindingModel

	err := r.db.WithContext(ctx).
		Where("notify_weekly_summary = ? AND weekly_summary_hour = ? AND weekly_summary_weekday = ?", true, bizHour, bizWeekday).
		Find(&dbModels).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(dbModels))
	for _, model := range dbModels {
		bindings = append(bindings, r.mapper.ToDomain(&model))
	}
	return bindings, nil
}

// FindBindingsForResourceExpiringNotification finds bindings that want resource expiring notifications
// and haven't been notified today
func (r *AdminTelegramBindingRepository) FindBindingsForResourceExpiringNotification(ctx context.Context) ([]*admin.AdminTelegramBinding, error) {
	var dbModels []models.AdminTelegramBindingModel
	today := biztime.StartOfDayUTC(biztime.NowUTC())

	err := r.db.WithContext(ctx).
		Where("notify_resource_expiring = ?", true).
		Where("last_resource_expiring_notify_date IS NULL OR last_resource_expiring_notify_date < ?", today).
		Find(&dbModels).Error
	if err != nil {
		return nil, err
	}

	bindings := make([]*admin.AdminTelegramBinding, 0, len(dbModels))
	for _, model := range dbModels {
		bindings = append(bindings, r.mapper.ToDomain(&model))
	}
	return bindings, nil
}
