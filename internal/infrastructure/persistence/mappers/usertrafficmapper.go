package mappers

import (
	"fmt"

	"orris/internal/domain/node"
	"orris/internal/infrastructure/persistence/models"
)

// UserTrafficMapper handles the conversion between domain entities and persistence models
type UserTrafficMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.UserTrafficModel) (*node.UserTraffic, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *node.UserTraffic) (*models.UserTrafficModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.UserTrafficModel) ([]*node.UserTraffic, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*node.UserTraffic) ([]*models.UserTrafficModel, error)
}

// UserTrafficMapperImpl is the concrete implementation of UserTrafficMapper
type UserTrafficMapperImpl struct{}

// NewUserTrafficMapper creates a new user traffic mapper
func NewUserTrafficMapper() UserTrafficMapper {
	return &UserTrafficMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity
func (m *UserTrafficMapperImpl) ToEntity(model *models.UserTrafficModel) (*node.UserTraffic, error) {
	if model == nil {
		return nil, nil
	}

	// Reconstruct the domain entity
	userTrafficEntity, err := node.ReconstructUserTraffic(
		model.ID,
		model.UserID,
		model.NodeID,
		model.SubscriptionID,
		model.Upload,
		model.Download,
		model.Total,
		model.Period,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct user traffic entity: %w", err)
	}

	return userTrafficEntity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *UserTrafficMapperImpl) ToModel(entity *node.UserTraffic) (*models.UserTrafficModel, error) {
	if entity == nil {
		return nil, nil
	}

	model := &models.UserTrafficModel{
		ID:             entity.ID(),
		UserID:         entity.UserID(),
		NodeID:         entity.NodeID(),
		SubscriptionID: entity.SubscriptionID(),
		Upload:         entity.Upload(),
		Download:       entity.Download(),
		Total:          entity.Total(),
		Period:         entity.Period(),
		CreatedAt:      entity.CreatedAt(),
		UpdatedAt:      entity.UpdatedAt(),
	}

	return model, nil
}

// ToEntities converts multiple persistence models to domain entities
func (m *UserTrafficMapperImpl) ToEntities(models []*models.UserTrafficModel) ([]*node.UserTraffic, error) {
	entities := make([]*node.UserTraffic, 0, len(models))

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

// ToModels converts multiple domain entities to persistence models
func (m *UserTrafficMapperImpl) ToModels(entities []*node.UserTraffic) ([]*models.UserTrafficModel, error) {
	models := make([]*models.UserTrafficModel, 0, len(entities))

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
