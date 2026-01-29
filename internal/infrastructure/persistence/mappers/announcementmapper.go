package mappers

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/notification"
	vo "github.com/orris-inc/orris/internal/domain/notification/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/mapper"
)

type AnnouncementMapper interface {
	ToEntity(model *models.AnnouncementModel) (*notification.Announcement, error)
	ToModel(entity *notification.Announcement) (*models.AnnouncementModel, error)
	ToEntities(models []*models.AnnouncementModel) ([]*notification.Announcement, error)
	ToModels(entities []*notification.Announcement) ([]*models.AnnouncementModel, error)
}

type AnnouncementMapperImpl struct{}

func NewAnnouncementMapper() AnnouncementMapper {
	return &AnnouncementMapperImpl{}
}

func (m *AnnouncementMapperImpl) ToEntity(model *models.AnnouncementModel) (*notification.Announcement, error) {
	if model == nil {
		return nil, nil
	}

	announcementType, err := vo.NewAnnouncementType(model.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to create announcement type: %w", err)
	}

	status, err := vo.NewAnnouncementStatus(model.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to create announcement status: %w", err)
	}

	entity, err := notification.ReconstructAnnouncement(
		model.ID,
		model.SID,
		model.Title,
		model.Content,
		announcementType,
		status,
		model.CreatorID,
		model.Priority,
		model.ScheduledAt,
		model.ExpiresAt,
		model.ViewCount,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct announcement entity: %w", err)
	}

	return entity, nil
}

func (m *AnnouncementMapperImpl) ToModel(entity *notification.Announcement) (*models.AnnouncementModel, error) {
	if entity == nil {
		return nil, nil
	}

	model := &models.AnnouncementModel{
		ID:          entity.ID(),
		SID:         entity.SID(),
		Title:       entity.Title(),
		Content:     entity.Content(),
		Type:        entity.Type().String(),
		Status:      entity.Status().String(),
		CreatorID:   entity.CreatorID(),
		Priority:    entity.Priority(),
		ScheduledAt: entity.ScheduledAt(),
		ExpiresAt:   entity.ExpiresAt(),
		ViewCount:   entity.ViewCount(),
		CreatedAt:   entity.CreatedAt(),
		UpdatedAt:   entity.UpdatedAt(),
	}

	if entity.Status().IsDeleted() {
		model.DeletedAt = gorm.DeletedAt{
			Time:  entity.UpdatedAt(),
			Valid: true,
		}
	}

	return model, nil
}

func (m *AnnouncementMapperImpl) ToEntities(modelList []*models.AnnouncementModel) ([]*notification.Announcement, error) {
	return mapper.MapSlicePtrWithID(modelList, m.ToEntity, func(model *models.AnnouncementModel) uint { return model.ID })
}

func (m *AnnouncementMapperImpl) ToModels(entities []*notification.Announcement) ([]*models.AnnouncementModel, error) {
	return mapper.MapSlicePtrWithID(entities, m.ToModel, func(entity *notification.Announcement) uint { return entity.ID() })
}
