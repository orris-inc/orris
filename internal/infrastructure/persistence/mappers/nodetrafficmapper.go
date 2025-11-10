package mappers

import (
	"fmt"

	"orris/internal/domain/node"
	"orris/internal/infrastructure/persistence/models"
)

// NodeTrafficMapper handles the conversion between domain entities and persistence models
type NodeTrafficMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.NodeTrafficModel) (*node.NodeTraffic, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *node.NodeTraffic) (*models.NodeTrafficModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.NodeTrafficModel) ([]*node.NodeTraffic, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*node.NodeTraffic) ([]*models.NodeTrafficModel, error)
}

// NodeTrafficMapperImpl is the concrete implementation of NodeTrafficMapper
type NodeTrafficMapperImpl struct{}

// NewNodeTrafficMapper creates a new node traffic mapper
func NewNodeTrafficMapper() NodeTrafficMapper {
	return &NodeTrafficMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity
func (m *NodeTrafficMapperImpl) ToEntity(model *models.NodeTrafficModel) (*node.NodeTraffic, error) {
	if model == nil {
		return nil, nil
	}

	// Reconstruct the domain entity
	nodeTrafficEntity, err := node.ReconstructNodeTraffic(
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
		return nil, fmt.Errorf("failed to reconstruct node traffic entity: %w", err)
	}

	return nodeTrafficEntity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *NodeTrafficMapperImpl) ToModel(entity *node.NodeTraffic) (*models.NodeTrafficModel, error) {
	if entity == nil {
		return nil, nil
	}

	model := &models.NodeTrafficModel{
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
func (m *NodeTrafficMapperImpl) ToEntities(models []*models.NodeTrafficModel) ([]*node.NodeTraffic, error) {
	entities := make([]*node.NodeTraffic, 0, len(models))

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
func (m *NodeTrafficMapperImpl) ToModels(entities []*node.NodeTraffic) ([]*models.NodeTrafficModel, error) {
	models := make([]*models.NodeTrafficModel, 0, len(entities))

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
