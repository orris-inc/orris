package mappers

import (
	"fmt"

	"orris/internal/domain/forward"
	vo "orris/internal/domain/forward/value_objects"
	"orris/internal/infrastructure/persistence/models"
)

// ForwardChainMapper handles the conversion between domain entities and persistence models.
type ForwardChainMapper interface {
	// ToEntity converts a persistence model to a domain entity.
	ToEntity(model *models.ForwardChainModel) (*forward.ForwardChain, error)

	// ToModel converts a domain entity to a persistence model.
	ToModel(entity *forward.ForwardChain) (*models.ForwardChainModel, error)

	// ToEntities converts multiple persistence models to domain entities.
	ToEntities(models []*models.ForwardChainModel) ([]*forward.ForwardChain, error)
}

// ForwardChainMapperImpl is the concrete implementation of ForwardChainMapper.
type ForwardChainMapperImpl struct{}

// NewForwardChainMapper creates a new forward chain mapper.
func NewForwardChainMapper() ForwardChainMapper {
	return &ForwardChainMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity.
func (m *ForwardChainMapperImpl) ToEntity(model *models.ForwardChainModel) (*forward.ForwardChain, error) {
	if model == nil {
		return nil, nil
	}

	protocol := vo.ForwardProtocol(model.Protocol)
	if !protocol.IsValid() {
		return nil, fmt.Errorf("invalid protocol: %s", model.Protocol)
	}

	status := vo.ForwardStatus(model.Status)
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid status: %s", model.Status)
	}

	// Convert nodes
	nodes := make([]forward.ChainNode, len(model.Nodes))
	for i, nodeModel := range model.Nodes {
		nodes[i] = forward.ChainNode{
			AgentID:    nodeModel.AgentID,
			ListenPort: nodeModel.ListenPort,
			Sequence:   nodeModel.Sequence,
		}
	}

	entity, err := forward.ReconstructForwardChain(
		model.ID,
		model.Name,
		protocol,
		status,
		nodes,
		model.TargetAddress,
		model.TargetPort,
		model.Remark,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct forward chain entity: %w", err)
	}

	return entity, nil
}

// ToModel converts a domain entity to a persistence model.
func (m *ForwardChainMapperImpl) ToModel(entity *forward.ForwardChain) (*models.ForwardChainModel, error) {
	if entity == nil {
		return nil, nil
	}

	// Convert nodes
	nodeModels := make([]models.ForwardChainNodeModel, len(entity.Nodes()))
	for i, node := range entity.Nodes() {
		nodeModels[i] = models.ForwardChainNodeModel{
			ChainID:    entity.ID(),
			AgentID:    node.AgentID,
			ListenPort: node.ListenPort,
			Sequence:   node.Sequence,
		}
	}

	return &models.ForwardChainModel{
		ID:            entity.ID(),
		Name:          entity.Name(),
		Protocol:      entity.Protocol().String(),
		Status:        entity.Status().String(),
		TargetAddress: entity.TargetAddress(),
		TargetPort:    entity.TargetPort(),
		Remark:        entity.Remark(),
		CreatedAt:     entity.CreatedAt(),
		UpdatedAt:     entity.UpdatedAt(),
		Nodes:         nodeModels,
	}, nil
}

// ToEntities converts multiple persistence models to domain entities.
func (m *ForwardChainMapperImpl) ToEntities(models []*models.ForwardChainModel) ([]*forward.ForwardChain, error) {
	entities := make([]*forward.ForwardChain, 0, len(models))

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
