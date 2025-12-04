package mappers

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// SubscriptionTrafficMapper handles the conversion between domain entities and persistence models
type SubscriptionTrafficMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.SubscriptionTrafficModel) (*subscription.SubscriptionTraffic, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *subscription.SubscriptionTraffic) (*models.SubscriptionTrafficModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.SubscriptionTrafficModel) ([]*subscription.SubscriptionTraffic, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*subscription.SubscriptionTraffic) ([]*models.SubscriptionTrafficModel, error)
}

// SubscriptionTrafficMapperImpl is the concrete implementation of SubscriptionTrafficMapper
type SubscriptionTrafficMapperImpl struct{}

// NewSubscriptionTrafficMapper creates a new subscription traffic mapper
func NewSubscriptionTrafficMapper() SubscriptionTrafficMapper {
	return &SubscriptionTrafficMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity
func (m *SubscriptionTrafficMapperImpl) ToEntity(model *models.SubscriptionTrafficModel) (*subscription.SubscriptionTraffic, error) {
	if model == nil {
		return nil, nil
	}

	// Reconstruct the domain entity
	trafficEntity, err := subscription.ReconstructSubscriptionTraffic(
		model.ID,
		model.NodeID,
		model.UserID,
		model.SubscriptionID,
		model.Upload,
		model.Download,
		model.Total,
		model.Period,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct subscription traffic entity: %w", err)
	}

	return trafficEntity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *SubscriptionTrafficMapperImpl) ToModel(entity *subscription.SubscriptionTraffic) (*models.SubscriptionTrafficModel, error) {
	if entity == nil {
		return nil, nil
	}

	model := &models.SubscriptionTrafficModel{
		ID:             entity.ID(),
		NodeID:         entity.NodeID(),
		UserID:         entity.UserID(),
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
func (m *SubscriptionTrafficMapperImpl) ToEntities(models []*models.SubscriptionTrafficModel) ([]*subscription.SubscriptionTraffic, error) {
	entities := make([]*subscription.SubscriptionTraffic, 0, len(models))

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
func (m *SubscriptionTrafficMapperImpl) ToModels(entities []*subscription.SubscriptionTraffic) ([]*models.SubscriptionTrafficModel, error) {
	models := make([]*models.SubscriptionTrafficModel, 0, len(entities))

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
