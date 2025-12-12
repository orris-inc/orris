package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/notification"
	vo "github.com/orris-inc/orris/internal/domain/notification/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/errors"
)

type NotificationTemplateRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.NotificationTemplateMapper
}

func NewNotificationTemplateRepository(db *gorm.DB) notification.NotificationTemplateRepository {
	return &NotificationTemplateRepositoryImpl{
		db:     db,
		mapper: mappers.NewNotificationTemplateMapper(),
	}
}

func (r *NotificationTemplateRepositoryImpl) Create(ctx context.Context, template *notification.NotificationTemplate) error {
	model, err := r.mapper.ToModel(template)
	if err != nil {
		return fmt.Errorf("failed to map notification template entity to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create notification template: %w", err)
	}

	if err := template.SetID(model.ID); err != nil {
		return fmt.Errorf("failed to set notification template ID: %w", err)
	}

	return nil
}

func (r *NotificationTemplateRepositoryImpl) GetByID(ctx context.Context, id uint) (*notification.NotificationTemplate, error) {
	var model models.NotificationTemplateModel

	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get notification template by ID: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		return nil, fmt.Errorf("failed to map notification template model to entity: %w", err)
	}

	return entity, nil
}

func (r *NotificationTemplateRepositoryImpl) Update(ctx context.Context, template *notification.NotificationTemplate) error {
	model, err := r.mapper.ToModel(template)
	if err != nil {
		return fmt.Errorf("failed to map notification template entity to model: %w", err)
	}

	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return fmt.Errorf("failed to update notification template: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("notification template not found")
	}

	return nil
}

func (r *NotificationTemplateRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.NotificationTemplateModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete notification template: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("notification template not found")
	}

	return nil
}

func (r *NotificationTemplateRepositoryImpl) GetByTemplateType(ctx context.Context, templateType vo.TemplateType) (*notification.NotificationTemplate, error) {
	var model models.NotificationTemplateModel

	if err := r.db.WithContext(ctx).Where("template_type = ?", templateType.String()).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get notification template by type: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		return nil, fmt.Errorf("failed to map notification template model to entity: %w", err)
	}

	return entity, nil
}

func (r *NotificationTemplateRepositoryImpl) ListEnabled(ctx context.Context) ([]*notification.NotificationTemplate, error) {
	var modelList []*models.NotificationTemplateModel

	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("created_at DESC").Find(&modelList).Error; err != nil {
		return nil, fmt.Errorf("failed to list enabled notification templates: %w", err)
	}

	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		return nil, fmt.Errorf("failed to map notification template models to entities: %w", err)
	}

	return entities, nil
}

func (r *NotificationTemplateRepositoryImpl) List(ctx context.Context, limit, offset int) ([]*notification.NotificationTemplate, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.NotificationTemplateModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count notification templates: %w", err)
	}

	var modelList []*models.NotificationTemplateModel
	query := r.db.WithContext(ctx).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&modelList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list notification templates: %w", err)
	}

	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to map notification template models to entities: %w", err)
	}

	return entities, total, nil
}
