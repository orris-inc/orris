package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"orris/internal/domain/notification"
	"orris/internal/infrastructure/persistence/mappers"
	"orris/internal/infrastructure/persistence/models"
	"orris/internal/shared/errors"
)

type NotificationRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.NotificationMapper
}

func NewNotificationRepository(db *gorm.DB) notification.NotificationRepository {
	return &NotificationRepositoryImpl{
		db:     db,
		mapper: mappers.NewNotificationMapper(),
	}
}

func (r *NotificationRepositoryImpl) Create(ctx context.Context, notif *notification.Notification) error {
	model, err := r.mapper.ToModel(notif)
	if err != nil {
		return fmt.Errorf("failed to map notification entity to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	if err := notif.SetID(model.ID); err != nil {
		return fmt.Errorf("failed to set notification ID: %w", err)
	}

	return nil
}

func (r *NotificationRepositoryImpl) GetByID(ctx context.Context, id uint) (*notification.Notification, error) {
	var model models.NotificationModel

	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get notification by ID: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		return nil, fmt.Errorf("failed to map notification model to entity: %w", err)
	}

	return entity, nil
}

func (r *NotificationRepositoryImpl) Update(ctx context.Context, notif *notification.Notification) error {
	model, err := r.mapper.ToModel(notif)
	if err != nil {
		return fmt.Errorf("failed to map notification entity to model: %w", err)
	}

	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return fmt.Errorf("failed to update notification: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("notification not found")
	}

	return nil
}

func (r *NotificationRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.NotificationModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete notification: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("notification not found")
	}

	return nil
}

func (r *NotificationRepositoryImpl) ListByUserID(ctx context.Context, userID uint, limit, offset int) ([]*notification.Notification, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&models.NotificationModel{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	var modelList []*models.NotificationModel
	query = query.Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&modelList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list notifications by user ID: %w", err)
	}

	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to map notification models to entities: %w", err)
	}

	return entities, total, nil
}

func (r *NotificationRepositoryImpl) CountUnread(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.NotificationModel{}).
		Where("user_id = ? AND read_status = ?", userID, "unread").
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count unread notifications: %w", err)
	}

	return count, nil
}

func (r *NotificationRepositoryImpl) MarkAsRead(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).
		Model(&models.NotificationModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"read_status": "read",
			"version":     gorm.Expr("version + ?", 1),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark notification as read: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("notification not found")
	}

	return nil
}

func (r *NotificationRepositoryImpl) BulkCreate(ctx context.Context, notifications []*notification.Notification) error {
	if len(notifications) == 0 {
		return nil
	}

	models, err := r.mapper.ToModels(notifications)
	if err != nil {
		return fmt.Errorf("failed to map notification entities to models: %w", err)
	}

	if err := r.db.WithContext(ctx).CreateInBatches(models, 100).Error; err != nil {
		return fmt.Errorf("failed to bulk create notifications: %w", err)
	}

	for i, model := range models {
		if err := notifications[i].SetID(model.ID); err != nil {
			return fmt.Errorf("failed to set notification ID: %w", err)
		}
	}

	return nil
}

func (r *NotificationRepositoryImpl) FindBySpecification(
	ctx context.Context,
	spec notification.Specification,
	limit, offset int,
) ([]*notification.Notification, int64, error) {
	var modelList []*models.NotificationModel

	query := r.db.WithContext(ctx).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&modelList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find notifications by specification: %w", err)
	}

	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to map notification models to entities: %w", err)
	}

	filtered := make([]*notification.Notification, 0)
	for _, entity := range entities {
		if spec.IsSatisfiedBy(entity) {
			filtered = append(filtered, entity)
		}
	}

	return filtered, int64(len(filtered)), nil
}
