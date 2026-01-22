package mappers

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/mapper"
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

	// Reconstruct the domain entity
	usageEntity, err := subscription.ReconstructSubscriptionUsage(
		model.ID,
		model.SID,
		model.ResourceType,
		model.ResourceID,
		model.SubscriptionID,
		model.Upload,
		model.Download,
		model.Total,
		model.Period,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct subscription usage entity: %w", err)
	}

	return usageEntity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *subscriptionUsageMapper) ToModel(entity *subscription.SubscriptionUsage) (*models.SubscriptionUsageModel, error) {
	if entity == nil {
		return nil, nil
	}

	model := &models.SubscriptionUsageModel{
		ID:             entity.ID(),
		SID:            entity.SID(),
		SubscriptionID: entity.SubscriptionID(),
		ResourceType:   entity.ResourceType(),
		ResourceID:     entity.ResourceID(),
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
func (m *subscriptionUsageMapper) ToEntities(modelList []*models.SubscriptionUsageModel) ([]*subscription.SubscriptionUsage, error) {
	return mapper.MapSlicePtrWithID(modelList, m.ToEntity, func(model *models.SubscriptionUsageModel) uint { return model.ID })
}

// ToModels converts multiple domain entities to persistence models
func (m *subscriptionUsageMapper) ToModels(entities []*subscription.SubscriptionUsage) ([]*models.SubscriptionUsageModel, error) {
	return mapper.MapSlicePtrWithID(entities, m.ToModel, func(entity *subscription.SubscriptionUsage) uint { return entity.ID() })
}
