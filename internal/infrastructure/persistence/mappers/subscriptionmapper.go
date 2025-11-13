package mappers

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"orris/internal/domain/subscription"
	vo "orris/internal/domain/subscription/value_objects"
	"orris/internal/infrastructure/persistence/models"
)

type SubscriptionMapper interface {
	ToEntity(model *models.SubscriptionModel) (*subscription.Subscription, error)
	ToModel(entity *subscription.Subscription) (*models.SubscriptionModel, error)
	ToEntities(models []*models.SubscriptionModel) ([]*subscription.Subscription, error)
	ToModels(entities []*subscription.Subscription) ([]*models.SubscriptionModel, error)
}

type SubscriptionMapperImpl struct{}

func NewSubscriptionMapper() SubscriptionMapper {
	return &SubscriptionMapperImpl{}
}

func (m *SubscriptionMapperImpl) ToEntity(model *models.SubscriptionModel) (*subscription.Subscription, error) {
	if model == nil {
		return nil, nil
	}

	status := vo.SubscriptionStatus(model.Status)
	if !vo.ValidStatuses[status] {
		return nil, fmt.Errorf("invalid subscription status: %s", model.Status)
	}

	var metadata map[string]interface{}
	if model.Metadata != nil {
		if err := json.Unmarshal(model.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	entity, err := subscription.ReconstructSubscription(
		model.ID,
		model.UserID,
		model.PlanID,
		model.UUID,
		status,
		model.StartDate,
		model.EndDate,
		model.AutoRenew,
		model.CurrentPeriodStart,
		model.CurrentPeriodEnd,
		model.CancelledAt,
		model.CancelReason,
		metadata,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct subscription entity: %w", err)
	}

	return entity, nil
}

func (m *SubscriptionMapperImpl) ToModel(entity *subscription.Subscription) (*models.SubscriptionModel, error) {
	if entity == nil {
		return nil, nil
	}

	var metadataJSON datatypes.JSON
	if metadata := entity.Metadata(); len(metadata) > 0 {
		data, err := json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = data
	}

	model := &models.SubscriptionModel{
		ID:                 entity.ID(),
		UUID:               entity.UUID(),
		UserID:             entity.UserID(),
		PlanID:             entity.PlanID(),
		Status:             entity.Status().String(),
		StartDate:          entity.StartDate(),
		EndDate:            entity.EndDate(),
		AutoRenew:          entity.AutoRenew(),
		CurrentPeriodStart: entity.CurrentPeriodStart(),
		CurrentPeriodEnd:   entity.CurrentPeriodEnd(),
		CancelledAt:        entity.CancelledAt(),
		CancelReason:       entity.CancelReason(),
		Metadata:           metadataJSON,
		Version:            entity.Version(),
		CreatedAt:          entity.CreatedAt(),
		UpdatedAt:          entity.UpdatedAt(),
	}

	if entity.Status() == vo.StatusCancelled {
		now := entity.UpdatedAt()
		model.DeletedAt = gorm.DeletedAt{
			Time:  now,
			Valid: true,
		}
	}

	return model, nil
}

func (m *SubscriptionMapperImpl) ToEntities(models []*models.SubscriptionModel) ([]*subscription.Subscription, error) {
	entities := make([]*subscription.Subscription, 0, len(models))

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

func (m *SubscriptionMapperImpl) ToModels(entities []*subscription.Subscription) ([]*models.SubscriptionModel, error) {
	models := make([]*models.SubscriptionModel, 0, len(entities))

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
