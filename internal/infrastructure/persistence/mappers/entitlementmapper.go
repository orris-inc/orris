package mappers

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// EntitlementMapper handles the conversion between domain entities and persistence models
type EntitlementMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.EntitlementModel) (*subscription.Entitlement, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *subscription.Entitlement) (*models.EntitlementModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.EntitlementModel) ([]*subscription.Entitlement, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*subscription.Entitlement) ([]*models.EntitlementModel, error)
}

// entitlementMapper is the concrete implementation of EntitlementMapper
type entitlementMapper struct{}

// NewEntitlementMapper creates a new entitlement mapper
func NewEntitlementMapper() EntitlementMapper {
	return &entitlementMapper{}
}

// ToEntity converts a persistence model to a domain entity
func (m *entitlementMapper) ToEntity(model *models.EntitlementModel) (*subscription.Entitlement, error) {
	if model == nil {
		return nil, nil
	}

	// Reconstruct entitlement using domain factory method
	entity, err := subscription.ReconstructEntitlement(
		model.ID,
		model.PlanID,
		model.ResourceType,
		model.ResourceID,
		model.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct entitlement entity: %w", err)
	}

	return entity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *entitlementMapper) ToModel(entity *subscription.Entitlement) (*models.EntitlementModel, error) {
	if entity == nil {
		return nil, nil
	}

	model := &models.EntitlementModel{
		ID:           entity.ID(),
		PlanID:       entity.PlanID(),
		ResourceType: string(entity.ResourceType()),
		ResourceID:   entity.ResourceID(),
		CreatedAt:    entity.CreatedAt(),
	}

	return model, nil
}

// ToEntities converts multiple persistence models to domain entities
func (m *entitlementMapper) ToEntities(models []*models.EntitlementModel) ([]*subscription.Entitlement, error) {
	entities := make([]*subscription.Entitlement, 0, len(models))

	for i, model := range models {
		entity, err := m.ToEntity(model)
		if err != nil {
			return nil, fmt.Errorf("failed to map model at index %d (ID %d): %w", i, model.ID, err)
		}
		if entity != nil {
			entities = append(entities, entity)
		}
	}

	return entities, nil
}

// ToModels converts multiple domain entities to persistence models
func (m *entitlementMapper) ToModels(entities []*subscription.Entitlement) ([]*models.EntitlementModel, error) {
	models := make([]*models.EntitlementModel, 0, len(entities))

	for i, entity := range entities {
		model, err := m.ToModel(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to map entity at index %d (ID %d): %w", i, entity.ID(), err)
		}
		if model != nil {
			models = append(models, model)
		}
	}

	return models, nil
}
