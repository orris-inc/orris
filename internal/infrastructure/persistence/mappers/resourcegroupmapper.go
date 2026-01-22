package mappers

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/mapper"
)

// ResourceGroupMapper handles the conversion between domain entities and persistence models.
type ResourceGroupMapper interface {
	// ToEntity converts a persistence model to a domain entity.
	ToEntity(model *models.ResourceGroupModel) (*resource.ResourceGroup, error)

	// ToModel converts a domain entity to a persistence model.
	ToModel(entity *resource.ResourceGroup) (*models.ResourceGroupModel, error)

	// ToEntities converts multiple persistence models to domain entities.
	ToEntities(models []*models.ResourceGroupModel) ([]*resource.ResourceGroup, error)
}

// ResourceGroupMapperImpl is the concrete implementation of ResourceGroupMapper.
type ResourceGroupMapperImpl struct{}

// NewResourceGroupMapper creates a new resource group mapper.
func NewResourceGroupMapper() ResourceGroupMapper {
	return &ResourceGroupMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity.
func (m *ResourceGroupMapperImpl) ToEntity(model *models.ResourceGroupModel) (*resource.ResourceGroup, error) {
	if model == nil {
		return nil, nil
	}

	entity, err := resource.ReconstructResourceGroup(
		model.ID,
		model.SID,
		model.Name,
		model.PlanID,
		model.Description,
		model.Status,
		model.CreatedAt,
		model.UpdatedAt,
		model.Version,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct resource group entity: %w", err)
	}

	return entity, nil
}

// ToModel converts a domain entity to a persistence model.
func (m *ResourceGroupMapperImpl) ToModel(entity *resource.ResourceGroup) (*models.ResourceGroupModel, error) {
	if entity == nil {
		return nil, nil
	}

	return &models.ResourceGroupModel{
		ID:          entity.ID(),
		SID:         entity.SID(),
		Name:        entity.Name(),
		PlanID:      entity.PlanID(),
		Description: entity.Description(),
		Status:      entity.Status().String(),
		CreatedAt:   entity.CreatedAt(),
		UpdatedAt:   entity.UpdatedAt(),
		Version:     entity.Version(),
	}, nil
}

// ToEntities converts multiple persistence models to domain entities.
func (m *ResourceGroupMapperImpl) ToEntities(modelList []*models.ResourceGroupModel) ([]*resource.ResourceGroup, error) {
	return mapper.MapSlicePtrWithID(modelList, m.ToEntity, func(model *models.ResourceGroupModel) uint { return model.ID })
}
