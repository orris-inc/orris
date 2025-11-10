package mappers

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"

	"orris/internal/domain/node"
	"orris/internal/infrastructure/persistence/models"
)

// NodeGroupMapper handles the conversion between domain entities and persistence models
type NodeGroupMapper interface {
	// ToEntity converts a persistence model to a domain entity
	// Note: This method does not load the many-to-many relationships (nodeIDs, subscriptionPlanIDs)
	// Those should be loaded separately by the repository
	ToEntity(model *models.NodeGroupModel, nodeIDs []uint, subscriptionPlanIDs []uint) (*node.NodeGroup, error)

	// ToModel converts a domain entity to a persistence model
	// Note: This method only converts the main entity fields
	// Many-to-many relationships should be handled separately by the repository
	ToModel(entity *node.NodeGroup) (*models.NodeGroupModel, error)

	// ToEntities converts multiple persistence models to domain entities
	// Note: This method does not load the many-to-many relationships
	// Those should be loaded separately by the repository
	ToEntities(models []*models.NodeGroupModel, nodeIDsMap map[uint][]uint, planIDsMap map[uint][]uint) ([]*node.NodeGroup, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*node.NodeGroup) ([]*models.NodeGroupModel, error)
}

// NodeGroupMapperImpl is the concrete implementation of NodeGroupMapper
type NodeGroupMapperImpl struct{}

// NewNodeGroupMapper creates a new node group mapper
func NewNodeGroupMapper() NodeGroupMapper {
	return &NodeGroupMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity
func (m *NodeGroupMapperImpl) ToEntity(
	model *models.NodeGroupModel,
	nodeIDs []uint,
	subscriptionPlanIDs []uint,
) (*node.NodeGroup, error) {
	if model == nil {
		return nil, nil
	}

	// Parse metadata from JSON
	var metadata map[string]interface{}
	if model.Metadata != nil {
		if err := json.Unmarshal(model.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	// Get description value
	description := ""
	if model.Description != nil {
		description = *model.Description
	}

	// Initialize slices if nil
	if nodeIDs == nil {
		nodeIDs = []uint{}
	}
	if subscriptionPlanIDs == nil {
		subscriptionPlanIDs = []uint{}
	}

	// Reconstruct the domain entity
	nodeGroupEntity, err := node.ReconstructNodeGroup(
		model.ID,
		model.Name,
		description,
		nodeIDs,
		subscriptionPlanIDs,
		model.IsPublic,
		model.SortOrder,
		metadata,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct node group entity: %w", err)
	}

	return nodeGroupEntity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *NodeGroupMapperImpl) ToModel(entity *node.NodeGroup) (*models.NodeGroupModel, error) {
	if entity == nil {
		return nil, nil
	}

	// Prepare description
	var description *string
	if entity.Description() != "" {
		desc := entity.Description()
		description = &desc
	}

	// Prepare metadata JSON
	var metadataJSON datatypes.JSON
	metadata := entity.Metadata()
	if len(metadata) > 0 {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = metadataBytes
	}

	model := &models.NodeGroupModel{
		ID:          entity.ID(),
		Name:        entity.Name(),
		Description: description,
		IsPublic:    entity.IsPublic(),
		SortOrder:   entity.SortOrder(),
		Metadata:    metadataJSON,
		Version:     entity.Version(),
		CreatedAt:   entity.CreatedAt(),
		UpdatedAt:   entity.UpdatedAt(),
	}

	// Note: Many-to-many relationships (nodeIDs, subscriptionPlanIDs) are not handled here
	// They should be managed separately by the repository using NodeGroupNodeModel and NodeGroupPlanModel

	return model, nil
}

// ToEntities converts multiple persistence models to domain entities
func (m *NodeGroupMapperImpl) ToEntities(
	models []*models.NodeGroupModel,
	nodeIDsMap map[uint][]uint,
	planIDsMap map[uint][]uint,
) ([]*node.NodeGroup, error) {
	entities := make([]*node.NodeGroup, 0, len(models))

	for _, model := range models {
		// Get the nodeIDs and planIDs for this group from the maps
		nodeIDs := nodeIDsMap[model.ID]
		planIDs := planIDsMap[model.ID]

		entity, err := m.ToEntity(model, nodeIDs, planIDs)
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
func (m *NodeGroupMapperImpl) ToModels(entities []*node.NodeGroup) ([]*models.NodeGroupModel, error) {
	models := make([]*models.NodeGroupModel, 0, len(entities))

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

// GetNodeIDs extracts the node IDs from a domain entity
// Helper method for repository to use when persisting many-to-many relationships
func GetNodeGroupNodeIDs(entity *node.NodeGroup) []uint {
	if entity == nil {
		return []uint{}
	}
	return entity.NodeIDs()
}

// GetSubscriptionPlanIDs extracts the subscription plan IDs from a domain entity
// Helper method for repository to use when persisting many-to-many relationships
func GetNodeGroupSubscriptionPlanIDs(entity *node.NodeGroup) []uint {
	if entity == nil {
		return []uint{}
	}
	return entity.SubscriptionPlanIDs()
}
