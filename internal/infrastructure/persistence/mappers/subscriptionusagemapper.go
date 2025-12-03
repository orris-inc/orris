package mappers

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// SubscriptionUsageMapper handles the conversion between domain entities and persistence models
type SubscriptionUsageMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.SubscriptionUsageModel) (*subscription.SubscriptionUsage, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *subscription.SubscriptionUsage) (*models.SubscriptionUsageModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.SubscriptionUsageModel) ([]*subscription.SubscriptionUsage, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*subscription.SubscriptionUsage) ([]*models.SubscriptionUsageModel, error)
}

// subscriptionUsageMapper is the concrete implementation of SubscriptionUsageMapper
type subscriptionUsageMapper struct{}

// NewSubscriptionUsageMapper creates a new subscription usage mapper
func NewSubscriptionUsageMapper() SubscriptionUsageMapper {
	return &subscriptionUsageMapper{}
}

// ToEntity converts a persistence model to a domain entity
func (m *subscriptionUsageMapper) ToEntity(model *models.SubscriptionUsageModel) (*subscription.SubscriptionUsage, error) {
	if model == nil {
		return nil, nil
	}

	// Reconstruct subscription usage using domain factory method
	entity, err := subscription.ReconstructSubscriptionUsage(
		model.ID,
		model.SubscriptionID,
		model.PeriodStart,
		model.UsersCount,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct subscription usage entity: %w", err)
	}

	return entity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *subscriptionUsageMapper) ToModel(entity *subscription.SubscriptionUsage) (*models.SubscriptionUsageModel, error) {
	if entity == nil {
		return nil, nil
	}

	model := &models.SubscriptionUsageModel{
		ID:             entity.ID(),
		SubscriptionID: entity.SubscriptionID(),
		PeriodStart:    entity.Period(),
		PeriodEnd:      entity.Period(),
		UsersCount:     entity.UsersCount(),
		UpdatedAt:      entity.UpdatedAt(),
	}

	return model, nil
}

// ToEntities converts multiple persistence models to domain entities
func (m *subscriptionUsageMapper) ToEntities(models []*models.SubscriptionUsageModel) ([]*subscription.SubscriptionUsage, error) {
	entities := make([]*subscription.SubscriptionUsage, 0, len(models))

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
func (m *subscriptionUsageMapper) ToModels(entities []*subscription.SubscriptionUsage) ([]*models.SubscriptionUsageModel, error) {
	models := make([]*models.SubscriptionUsageModel, 0, len(entities))

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
