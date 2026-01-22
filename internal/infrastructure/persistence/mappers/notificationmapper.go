package mappers

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/notification"
	vo "github.com/orris-inc/orris/internal/domain/notification/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/mapper"
)

type NotificationMapper interface {
	ToEntity(model *models.NotificationModel) (*notification.Notification, error)
	ToModel(entity *notification.Notification) (*models.NotificationModel, error)
	ToEntities(models []*models.NotificationModel) ([]*notification.Notification, error)
	ToModels(entities []*notification.Notification) ([]*models.NotificationModel, error)
}

type NotificationMapperImpl struct{}

func NewNotificationMapper() NotificationMapper {
	return &NotificationMapperImpl{}
}

func (m *NotificationMapperImpl) ToEntity(model *models.NotificationModel) (*notification.Notification, error) {
	if model == nil {
		return nil, nil
	}

	notificationType, err := vo.NewNotificationType(model.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification type: %w", err)
	}

	readStatus, err := vo.NewReadStatus(model.ReadStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to create read status: %w", err)
	}

	entity, err := notification.ReconstructNotification(
		model.ID,
		model.UserID,
		notificationType,
		model.Title,
		model.Content,
		model.RelatedID,
		readStatus,
		model.ArchivedAt,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct notification entity: %w", err)
	}

	return entity, nil
}

func (m *NotificationMapperImpl) ToModel(entity *notification.Notification) (*models.NotificationModel, error) {
	if entity == nil {
		return nil, nil
	}

	model := &models.NotificationModel{
		ID:         entity.ID(),
		UserID:     entity.UserID(),
		Type:       entity.Type().String(),
		Title:      entity.Title(),
		Content:    entity.Content(),
		RelatedID:  entity.RelatedID(),
		ReadStatus: entity.ReadStatus().String(),
		ArchivedAt: entity.ArchivedAt(),
		CreatedAt:  entity.CreatedAt(),
		UpdatedAt:  entity.UpdatedAt(),
	}

	if entity.ArchivedAt() != nil {
		model.DeletedAt = gorm.DeletedAt{
			Time:  *entity.ArchivedAt(),
			Valid: true,
		}
	}

	return model, nil
}

func (m *NotificationMapperImpl) ToEntities(modelList []*models.NotificationModel) ([]*notification.Notification, error) {
	return mapper.MapSlicePtrWithID(modelList, m.ToEntity, func(model *models.NotificationModel) uint { return model.ID })
}

func (m *NotificationMapperImpl) ToModels(entities []*notification.Notification) ([]*models.NotificationModel, error) {
	return mapper.MapSlicePtrWithID(entities, m.ToModel, func(entity *notification.Notification) uint { return entity.ID() })
}
