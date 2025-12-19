package mappers

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"

	"github.com/orris-inc/orris/internal/domain/subscription"
	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// PlanMapper handles the conversion between domain entities and persistence models
type PlanMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.PlanModel) (*subscription.Plan, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *subscription.Plan) (*models.PlanModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.PlanModel) ([]*subscription.Plan, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*subscription.Plan) ([]*models.PlanModel, error)
}

// planMapper is the concrete implementation of PlanMapper
type planMapper struct{}

// NewPlanMapper creates a new plan mapper
func NewPlanMapper() PlanMapper {
	return &planMapper{}
}

// ToEntity converts a persistence model to a domain entity
func (m *planMapper) ToEntity(model *models.PlanModel) (*subscription.Plan, error) {
	if model == nil {
		return nil, nil
	}

	// Parse plan type
	planType := model.PlanType
	if planType == "" {
		planType = "node" // default value
	}

	// Parse limits JSON
	var features *vo.PlanFeatures
	if model.Limits != nil && len(model.Limits) > 0 {
		var limits map[string]interface{}
		if err := json.Unmarshal(model.Limits, &limits); err != nil {
			return nil, fmt.Errorf("failed to unmarshal limits: %w", err)
		}
		features = vo.NewPlanFeatures(limits)
	}

	// Parse metadata JSON
	var metadata map[string]interface{}
	if model.Metadata != nil && len(model.Metadata) > 0 {
		if err := json.Unmarshal(model.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Reconstruct plan using domain factory method
	entity, err := subscription.ReconstructPlan(
		model.ID,
		model.SID,
		model.Name,
		model.Slug,
		model.Description,
		model.TrialDays,
		model.Status,
		planType,
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
		return nil, fmt.Errorf("failed to reconstruct plan entity: %w", err)
	}

	return entity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *planMapper) ToModel(entity *subscription.Plan) (*models.PlanModel, error) {
	if entity == nil {
		return nil, nil
	}

	// Marshal limits to JSON
	var limitsJSON datatypes.JSON
	if features := entity.Features(); features != nil && features.Limits != nil {
		data, err := json.Marshal(features.Limits)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal limits: %w", err)
		}
		limitsJSON = data
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

	model := &models.PlanModel{
		ID:           entity.ID(),
		SID:          entity.SID(),
		Name:         entity.Name(),
		Slug:         entity.Slug(),
		PlanType:     entity.PlanType().String(),
		Description:  entity.Description(),
		TrialDays:    entity.TrialDays(),
		Status:       string(entity.Status()),
		Limits:       limitsJSON,
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
func (m *planMapper) ToEntities(models []*models.PlanModel) ([]*subscription.Plan, error) {
	entities := make([]*subscription.Plan, 0, len(models))

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
func (m *planMapper) ToModels(entities []*subscription.Plan) ([]*models.PlanModel, error) {
	models := make([]*models.PlanModel, 0, len(entities))

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
