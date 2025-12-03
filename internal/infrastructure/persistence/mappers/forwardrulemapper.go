package mappers

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/value_objects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// ForwardRuleMapper handles the conversion between domain entities and persistence models.
type ForwardRuleMapper interface {
	// ToEntity converts a persistence model to a domain entity.
	ToEntity(model *models.ForwardRuleModel) (*forward.ForwardRule, error)

	// ToModel converts a domain entity to a persistence model.
	ToModel(entity *forward.ForwardRule) (*models.ForwardRuleModel, error)

	// ToEntities converts multiple persistence models to domain entities.
	ToEntities(models []*models.ForwardRuleModel) ([]*forward.ForwardRule, error)
}

// ForwardRuleMapperImpl is the concrete implementation of ForwardRuleMapper.
type ForwardRuleMapperImpl struct{}

// NewForwardRuleMapper creates a new forward rule mapper.
func NewForwardRuleMapper() ForwardRuleMapper {
	return &ForwardRuleMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity.
func (m *ForwardRuleMapperImpl) ToEntity(model *models.ForwardRuleModel) (*forward.ForwardRule, error) {
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

	entity, err := forward.ReconstructForwardRule(
		model.ID,
		model.AgentID,
		model.NextAgentID,
		model.Name,
		model.ListenPort,
		model.TargetAddress,
		model.TargetPort,
		protocol,
		status,
		model.Remark,
		model.UploadBytes,
		model.DownloadBytes,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct forward rule entity: %w", err)
	}

	return entity, nil
}

// ToModel converts a domain entity to a persistence model.
func (m *ForwardRuleMapperImpl) ToModel(entity *forward.ForwardRule) (*models.ForwardRuleModel, error) {
	if entity == nil {
		return nil, nil
	}

	return &models.ForwardRuleModel{
		ID:            entity.ID(),
		AgentID:       entity.AgentID(),
		NextAgentID:   entity.NextAgentID(),
		Name:          entity.Name(),
		ListenPort:    entity.ListenPort(),
		TargetAddress: entity.TargetAddress(),
		TargetPort:    entity.TargetPort(),
		Protocol:      entity.Protocol().String(),
		Status:        entity.Status().String(),
		Remark:        entity.Remark(),
		UploadBytes:   entity.UploadBytes(),
		DownloadBytes: entity.DownloadBytes(),
		CreatedAt:     entity.CreatedAt(),
		UpdatedAt:     entity.UpdatedAt(),
	}, nil
}

// ToEntities converts multiple persistence models to domain entities.
func (m *ForwardRuleMapperImpl) ToEntities(models []*models.ForwardRuleModel) ([]*forward.ForwardRule, error) {
	entities := make([]*forward.ForwardRule, 0, len(models))

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
