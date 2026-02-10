package mappers

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/mapper"
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

// SubscriptionUsageStatsMapperImpl is the concrete implementation of SubscriptionUsageStatsMapper
type SubscriptionUsageStatsMapperImpl struct{}

// NewSubscriptionUsageStatsMapper creates a new subscription usage stats mapper
func NewSubscriptionUsageStatsMapper() SubscriptionUsageStatsMapper {
	return &SubscriptionUsageStatsMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity
func (m *SubscriptionUsageStatsMapperImpl) ToEntity(model *models.SubscriptionUsageStatsModel) (*subscription.SubscriptionUsageStats, error) {
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
func (m *SubscriptionUsageStatsMapperImpl) ToModel(entity *subscription.SubscriptionUsageStats) (*models.SubscriptionUsageStatsModel, error) {
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
func (m *SubscriptionUsageStatsMapperImpl) ToEntities(modelList []*models.SubscriptionUsageStatsModel) ([]*subscription.SubscriptionUsageStats, error) {
	return mapper.MapSlicePtrWithID(modelList, m.ToEntity, func(model *models.SubscriptionUsageStatsModel) uint { return model.ID })
}

// ToModels converts multiple domain entities to persistence models
func (m *SubscriptionUsageStatsMapperImpl) ToModels(entities []*subscription.SubscriptionUsageStats) ([]*models.SubscriptionUsageStatsModel, error) {
	return mapper.MapSlicePtrWithID(entities, m.ToModel, func(entity *subscription.SubscriptionUsageStats) uint { return entity.ID() })
}
