package mappers

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// SubscriptionPlanMapper handles the conversion between domain entities and persistence models
type SubscriptionPlanMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.SubscriptionPlanModel) (*subscription.SubscriptionPlan, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *subscription.SubscriptionPlan) (*models.SubscriptionPlanModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.SubscriptionPlanModel) ([]*subscription.SubscriptionPlan, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*subscription.SubscriptionPlan) ([]*models.SubscriptionPlanModel, error)
}

// subscriptionPlanMapper is the concrete implementation of SubscriptionPlanMapper
type subscriptionPlanMapper struct{}

// NewSubscriptionPlanMapper creates a new subscription plan mapper
func NewSubscriptionPlanMapper() SubscriptionPlanMapper {
	return &subscriptionPlanMapper{}
}

// ToEntity converts a persistence model to a domain entity
func (m *subscriptionPlanMapper) ToEntity(model *models.SubscriptionPlanModel) (*subscription.SubscriptionPlan, error) {
	if model == nil {
		return nil, nil
	}

	// Parse billing cycle
	billingCycle, err := vo.ParseBillingCycle(model.BillingCycle)
	if err != nil {
		return nil, fmt.Errorf("failed to parse billing cycle: %w", err)
	}

	// Parse features JSON
	var features *vo.PlanFeatures
	if len(model.Features) > 0 {
		features = &vo.PlanFeatures{}
		if err := json.Unmarshal(model.Features, features); err != nil {
			return nil, fmt.Errorf("failed to unmarshal features: %w", err)
		}
	}

	// Parse metadata JSON
	var metadata map[string]interface{}
	if len(model.Metadata) > 0 {
		if err := json.Unmarshal(model.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Reconstruct subscription plan using domain factory method
	entity, err := subscription.ReconstructSubscriptionPlan(
		model.ID,
		model.Name,
		model.Slug,
		model.Description,
		model.Price,
		model.Currency,
		billingCycle,
		model.TrialDays,
		model.Status,
		features,
		model.APIRateLimit,
		model.MaxUsers,
		model.MaxProjects,
		model.IsPublic,
		model.SortOrder,
		metadata,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct subscription plan entity: %w", err)
	}

	return entity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *subscriptionPlanMapper) ToModel(entity *subscription.SubscriptionPlan) (*models.SubscriptionPlanModel, error) {
	if entity == nil {
		return nil, nil
	}

	// Marshal features to JSON
	var featuresJSON datatypes.JSON
	if features := entity.Features(); features != nil {
		data, err := json.Marshal(features)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal features: %w", err)
		}
		featuresJSON = data
	}

	// Marshal metadata to JSON
	var metadataJSON datatypes.JSON
	if metadata := entity.Metadata(); len(metadata) > 0 {
		data, err := json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = data
	}

	model := &models.SubscriptionPlanModel{
		ID:           entity.ID(),
		Name:         entity.Name(),
		Slug:         entity.Slug(),
		Description:  entity.Description(),
		Price:        entity.Price(),
		Currency:     entity.Currency(),
		BillingCycle: entity.BillingCycle().String(),
		TrialDays:    entity.TrialDays(),
		Status:       string(entity.Status()),
		Features:     featuresJSON,
		APIRateLimit: entity.APIRateLimit(),
		MaxUsers:     entity.MaxUsers(),
		MaxProjects:  entity.MaxProjects(),
		IsPublic:     entity.IsPublic(),
		SortOrder:    entity.SortOrder(),
		Metadata:     metadataJSON,
		Version:      entity.Version(),
		CreatedAt:    entity.CreatedAt(),
		UpdatedAt:    entity.UpdatedAt(),
	}

	return model, nil
}

// ToEntities converts multiple persistence models to domain entities
func (m *subscriptionPlanMapper) ToEntities(models []*models.SubscriptionPlanModel) ([]*subscription.SubscriptionPlan, error) {
	entities := make([]*subscription.SubscriptionPlan, 0, len(models))

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
func (m *subscriptionPlanMapper) ToModels(entities []*subscription.SubscriptionPlan) ([]*models.SubscriptionPlanModel, error) {
	models := make([]*models.SubscriptionPlanModel, 0, len(entities))

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
