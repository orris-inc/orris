package mappers

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// PlanEntitlementMapper handles the conversion between domain entities and persistence models for plan entitlements
type PlanEntitlementMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.PlanEntitlementModel) (*subscription.Entitlement, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *subscription.Entitlement) (*models.PlanEntitlementModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.PlanEntitlementModel) ([]*subscription.Entitlement, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*subscription.Entitlement) ([]*models.PlanEntitlementModel, error)
}

// planEntitlementMapper is the concrete implementation of PlanEntitlementMapper
type planEntitlementMapper struct{}

// NewPlanEntitlementMapper creates a new plan entitlement mapper
func NewPlanEntitlementMapper() PlanEntitlementMapper {
	return &planEntitlementMapper{}
}

// ToEntity converts a persistence model to a domain entity
func (m *planEntitlementMapper) ToEntity(model *models.PlanEntitlementModel) (*subscription.Entitlement, error) {
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
		return nil, fmt.Errorf("failed to reconstruct plan entitlement entity: %w", err)
	}

	return entity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *planEntitlementMapper) ToModel(entity *subscription.Entitlement) (*models.PlanEntitlementModel, error) {
	if entity == nil {
		return nil, nil
	}

	model := &models.PlanEntitlementModel{
		ID:           entity.ID(),
		PlanID:       entity.PlanID(),
		ResourceType: string(entity.ResourceType()),
		ResourceID:   entity.ResourceID(),
		CreatedAt:    entity.CreatedAt(),
	}

	return model, nil
}

// ToEntities converts multiple persistence models to domain entities
func (m *planEntitlementMapper) ToEntities(models []*models.PlanEntitlementModel) ([]*subscription.Entitlement, error) {
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
func (m *planEntitlementMapper) ToModels(entities []*subscription.Entitlement) ([]*models.PlanEntitlementModel, error) {
	models := make([]*models.PlanEntitlementModel, 0, len(entities))

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
