package mappers

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/notification"
	vo "github.com/orris-inc/orris/internal/domain/notification/value_objects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

type NotificationTemplateMapper interface {
	ToEntity(model *models.NotificationTemplateModel) (*notification.NotificationTemplate, error)
	ToModel(entity *notification.NotificationTemplate) (*models.NotificationTemplateModel, error)
	ToEntities(models []*models.NotificationTemplateModel) ([]*notification.NotificationTemplate, error)
	ToModels(entities []*notification.NotificationTemplate) ([]*models.NotificationTemplateModel, error)
}

type NotificationTemplateMapperImpl struct{}

func NewNotificationTemplateMapper() NotificationTemplateMapper {
	return &NotificationTemplateMapperImpl{}
}

func (m *NotificationTemplateMapperImpl) ToEntity(model *models.NotificationTemplateModel) (*notification.NotificationTemplate, error) {
	if model == nil {
		return nil, nil
	}

	templateType, err := vo.NewTemplateType(model.TemplateType)
	if err != nil {
		return nil, fmt.Errorf("failed to create template type: %w", err)
	}

	var variables []string
	if model.Variables != "" {
		if err := json.Unmarshal([]byte(model.Variables), &variables); err != nil {
			return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
		}
	}
	if variables == nil {
		variables = []string{}
	}

	entity, err := notification.ReconstructNotificationTemplate(
		model.ID,
		templateType,
		model.Name,
		model.Title,
		model.Content,
		variables,
		model.Enabled,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct notification template entity: %w", err)
	}

	return entity, nil
}

func (m *NotificationTemplateMapperImpl) ToModel(entity *notification.NotificationTemplate) (*models.NotificationTemplateModel, error) {
	if entity == nil {
		return nil, nil
	}

	var variablesJSON string
	if variables := entity.Variables(); len(variables) > 0 {
		data, err := json.Marshal(variables)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal variables: %w", err)
		}
		variablesJSON = string(data)
	}

	model := &models.NotificationTemplateModel{
		ID:           entity.ID(),
		TemplateType: entity.TemplateType().String(),
		Name:         entity.Name(),
		Title:        entity.Title(),
		Content:      entity.Content(),
		Variables:    variablesJSON,
		Enabled:      entity.Enabled(),
		CreatedAt:    entity.CreatedAt(),
		UpdatedAt:    entity.UpdatedAt(),
	}

	if !entity.Enabled() {
		model.DeletedAt = gorm.DeletedAt{
			Time:  entity.UpdatedAt(),
			Valid: false,
		}
	}

	return model, nil
}

func (m *NotificationTemplateMapperImpl) ToEntities(models []*models.NotificationTemplateModel) ([]*notification.NotificationTemplate, error) {
	if models == nil {
		return nil, nil
	}

	entities := make([]*notification.NotificationTemplate, 0, len(models))
	for _, model := range models {
		entity, err := m.ToEntity(model)
		if err != nil {
			return nil, fmt.Errorf("failed to map model ID %d: %w", model.ID, err)
		}
		if entity != nil {
			entities = append(entities, entity)
		}
	}

	return entities, nil
}

func (m *NotificationTemplateMapperImpl) ToModels(entities []*notification.NotificationTemplate) ([]*models.NotificationTemplateModel, error) {
	if entities == nil {
		return nil, nil
	}

	models := make([]*models.NotificationTemplateModel, 0, len(entities))
	for _, entity := range entities {
		model, err := m.ToModel(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to map entity ID %d: %w", entity.ID(), err)
		}
		if model != nil {
			models = append(models, model)
		}
	}

	return models, nil
}
