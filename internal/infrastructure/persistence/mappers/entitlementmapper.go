package mappers

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"

	"github.com/orris-inc/orris/internal/domain/entitlement"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// EntitlementMapper handles the conversion between domain entities and persistence models
type EntitlementMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.EntitlementModel) (*entitlement.Entitlement, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *entitlement.Entitlement) (*models.EntitlementModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.EntitlementModel) ([]*entitlement.Entitlement, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*entitlement.Entitlement) ([]*models.EntitlementModel, error)
}

// EntitlementMapperImpl is the concrete implementation of EntitlementMapper
type EntitlementMapperImpl struct{}

// NewEntitlementMapper creates a new entitlement mapper
func NewEntitlementMapper() EntitlementMapper {
	return &EntitlementMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity
func (m *EntitlementMapperImpl) ToEntity(model *models.EntitlementModel) (*entitlement.Entitlement, error) {
	if model == nil {
		return nil, nil
	}

	// Parse value objects
	subjectType := entitlement.SubjectType(model.SubjectType)
	if !subjectType.IsValid() {
		return nil, fmt.Errorf("invalid subject type: %s", model.SubjectType)
	}

	resourceType := entitlement.ResourceType(model.ResourceType)
	if !resourceType.IsValid() {
		return nil, fmt.Errorf("invalid resource type: %s", model.ResourceType)
	}

	sourceType := entitlement.SourceType(model.SourceType)
	if !sourceType.IsValid() {
		return nil, fmt.Errorf("invalid source type: %s", model.SourceType)
	}

	status := entitlement.EntitlementStatus(model.Status)
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid entitlement status: %s", model.Status)
	}

	// Unmarshal metadata
	var metadata map[string]any
	if model.Metadata != nil {
		if err := json.Unmarshal(model.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	if metadata == nil {
		metadata = make(map[string]any)
	}

	// Reconstruct entitlement using domain factory method
	entity, err := entitlement.ReconstructEntitlement(
		model.ID,
		subjectType,
		model.SubjectID,
		resourceType,
		model.ResourceID,
		sourceType,
		model.SourceID,
		status,
		model.ExpiresAt,
		metadata,
		model.CreatedAt,
		model.UpdatedAt,
		model.Version,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct entitlement entity: %w", err)
	}

	return entity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *EntitlementMapperImpl) ToModel(entity *entitlement.Entitlement) (*models.EntitlementModel, error) {
	if entity == nil {
		return nil, nil
	}

	// Marshal metadata
	var metadataJSON datatypes.JSON
	if metadata := entity.Metadata(); len(metadata) > 0 {
		data, err := json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = data
	}

	model := &models.EntitlementModel{
		ID:           entity.ID(),
		SubjectType:  entity.SubjectType().String(),
		SubjectID:    entity.SubjectID(),
		ResourceType: entity.ResourceType().String(),
		ResourceID:   entity.ResourceID(),
		SourceType:   entity.SourceType().String(),
		SourceID:     entity.SourceID(),
		Status:       entity.Status().String(),
		ExpiresAt:    entity.ExpiresAt(),
		Metadata:     metadataJSON,
		CreatedAt:    entity.CreatedAt(),
		UpdatedAt:    entity.UpdatedAt(),
		Version:      entity.Version(),
	}

	return model, nil
}

// ToEntities converts multiple persistence models to domain entities
func (m *EntitlementMapperImpl) ToEntities(models []*models.EntitlementModel) ([]*entitlement.Entitlement, error) {
	entities := make([]*entitlement.Entitlement, 0, len(models))

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
func (m *EntitlementMapperImpl) ToModels(entities []*entitlement.Entitlement) ([]*models.EntitlementModel, error) {
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
