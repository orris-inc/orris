package mappers

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// SubscriptionUsageStatsMapper handles the conversion between domain entities and persistence models
type SubscriptionUsageStatsMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.SubscriptionUsageStatsModel) (*subscription.SubscriptionUsageStats, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *subscription.SubscriptionUsageStats) (*models.SubscriptionUsageStatsModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.SubscriptionUsageStatsModel) ([]*subscription.SubscriptionUsageStats, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*subscription.SubscriptionUsageStats) ([]*models.SubscriptionUsageStatsModel, error)
}

// subscriptionUsageStatsMapper is the concrete implementation of SubscriptionUsageStatsMapper
type subscriptionUsageStatsMapper struct{}

// NewSubscriptionUsageStatsMapper creates a new subscription usage stats mapper
func NewSubscriptionUsageStatsMapper() SubscriptionUsageStatsMapper {
	return &subscriptionUsageStatsMapper{}
}

// ToEntity converts a persistence model to a domain entity
func (m *subscriptionUsageStatsMapper) ToEntity(model *models.SubscriptionUsageStatsModel) (*subscription.SubscriptionUsageStats, error) {
	if model == nil {
		return nil, nil
	}

	// Reconstruct the domain entity
	entity, err := subscription.ReconstructSubscriptionUsageStats(
		model.ID,
		model.SID,
		model.ResourceType,
		model.ResourceID,
		model.SubscriptionID,
		model.Upload,
		model.Download,
		model.Total,
		subscription.Granularity(model.Granularity),
		model.Period,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct subscription usage stats entity: %w", err)
	}

	return entity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *subscriptionUsageStatsMapper) ToModel(entity *subscription.SubscriptionUsageStats) (*models.SubscriptionUsageStatsModel, error) {
	if entity == nil {
		return nil, nil
	}

	model := &models.SubscriptionUsageStatsModel{
		ID:             entity.ID(),
		SID:            entity.SID(),
		SubscriptionID: entity.SubscriptionID(),
		ResourceType:   entity.ResourceType(),
		ResourceID:     entity.ResourceID(),
		Upload:         entity.Upload(),
		Download:       entity.Download(),
		Total:          entity.Total(),
		Granularity:    entity.Granularity().String(),
		Period:         entity.Period(),
		CreatedAt:      entity.CreatedAt(),
		UpdatedAt:      entity.UpdatedAt(),
	}

	return model, nil
}

// ToEntities converts multiple persistence models to domain entities
func (m *subscriptionUsageStatsMapper) ToEntities(models []*models.SubscriptionUsageStatsModel) ([]*subscription.SubscriptionUsageStats, error) {
	entities := make([]*subscription.SubscriptionUsageStats, 0, len(models))

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
func (m *subscriptionUsageStatsMapper) ToModels(entities []*subscription.SubscriptionUsageStats) ([]*models.SubscriptionUsageStatsModel, error) {
	result := make([]*models.SubscriptionUsageStatsModel, 0, len(entities))

	for _, entity := range entities {
		model, err := m.ToModel(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to map entity ID %d: %w", entity.ID(), err)
		}
		if model != nil {
			result = append(result, model)
		}
	}

	return result, nil
}
