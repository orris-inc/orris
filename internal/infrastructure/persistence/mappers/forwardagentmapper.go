package mappers

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// ForwardAgentMapper handles the conversion between domain entities and persistence models.
type ForwardAgentMapper interface {
	// ToEntity converts a persistence model to a domain entity.
	ToEntity(model *models.ForwardAgentModel) (*forward.ForwardAgent, error)

	// ToModel converts a domain entity to a persistence model.
	ToModel(entity *forward.ForwardAgent) (*models.ForwardAgentModel, error)

	// ToEntities converts multiple persistence models to domain entities.
	ToEntities(models []*models.ForwardAgentModel) ([]*forward.ForwardAgent, error)
}

// ForwardAgentMapperImpl is the concrete implementation of ForwardAgentMapper.
type ForwardAgentMapperImpl struct{}

// NewForwardAgentMapper creates a new forward agent mapper.
func NewForwardAgentMapper() ForwardAgentMapper {
	return &ForwardAgentMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity.
func (m *ForwardAgentMapperImpl) ToEntity(model *models.ForwardAgentModel) (*forward.ForwardAgent, error) {
	if model == nil {
		return nil, nil
	}

	status := forward.AgentStatus(model.Status)
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid agent status: %s", model.Status)
	}

	// Parse subscription plan IDs from JSON
	var planIDs []uint
	if model.PlanIDs != nil {
		if err := json.Unmarshal(model.PlanIDs, &planIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal plan_ids: %w", err)
		}
	}

	entity, err := forward.ReconstructForwardAgent(
		model.ID,
		model.ShortID,
		model.Name,
		model.TokenHash,
		model.APIToken,
		status,
		model.PublicAddress,
		model.TunnelAddress,
		model.Remark,
		planIDs,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct forward agent entity: %w", err)
	}

	return entity, nil
}

// ToModel converts a domain entity to a persistence model.
func (m *ForwardAgentMapperImpl) ToModel(entity *forward.ForwardAgent) (*models.ForwardAgentModel, error) {
	if entity == nil {
		return nil, nil
	}

	// Prepare subscription plan IDs JSON
	var planIDsJSON datatypes.JSON
	planIDs := entity.PlanIDs()
	if len(planIDs) > 0 {
		idsBytes, err := json.Marshal(planIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal plan_ids: %w", err)
		}
		planIDsJSON = idsBytes
	} else {
		planIDsJSON = []byte("[]")
	}

	return &models.ForwardAgentModel{
		ID:            entity.ID(),
		ShortID:       entity.ShortID(),
		Name:          entity.Name(),
		TokenHash:     entity.TokenHash(),
		APIToken:      entity.GetAPIToken(),
		PublicAddress: entity.PublicAddress(),
		TunnelAddress: entity.TunnelAddress(),
		Status:        string(entity.Status()),
		Remark:        entity.Remark(),
		PlanIDs:       planIDsJSON,
		CreatedAt:     entity.CreatedAt(),
		UpdatedAt:     entity.UpdatedAt(),
	}, nil
}

// ToEntities converts multiple persistence models to domain entities.
func (m *ForwardAgentMapperImpl) ToEntities(models []*models.ForwardAgentModel) ([]*forward.ForwardAgent, error) {
	entities := make([]*forward.ForwardAgent, 0, len(models))

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
