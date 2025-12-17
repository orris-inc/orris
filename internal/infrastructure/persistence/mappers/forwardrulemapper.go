package mappers

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
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

	ruleType := vo.ForwardRuleType(model.RuleType)
	if !ruleType.IsValid() {
		return nil, fmt.Errorf("invalid rule type: %s", model.RuleType)
	}

	// Handle nullable fields
	var exitAgentID uint
	if model.ExitAgentID != nil {
		exitAgentID = *model.ExitAgentID
	}

	var userID *uint
	if model.UserID != nil {
		userID = model.UserID
	}

	var targetNodeID *uint
	if model.TargetNodeID != nil {
		targetNodeID = model.TargetNodeID
	}

	// Parse chain_agent_ids JSON
	var chainAgentIDs []uint
	if model.ChainAgentIDs != nil && len(model.ChainAgentIDs) > 0 {
		if err := json.Unmarshal(model.ChainAgentIDs, &chainAgentIDs); err != nil {
			return nil, fmt.Errorf("failed to parse chain_agent_ids: %w", err)
		}
	}

	// Parse chain_port_config JSON
	var chainPortConfig map[uint]uint16
	if model.ChainPortConfig != nil && len(model.ChainPortConfig) > 0 {
		// JSON unmarshals numeric keys as strings, so we need to handle this
		var rawConfig map[string]uint16
		if err := json.Unmarshal(model.ChainPortConfig, &rawConfig); err != nil {
			return nil, fmt.Errorf("failed to parse chain_port_config: %w", err)
		}
		// Convert string keys to uint
		chainPortConfig = make(map[uint]uint16)
		for k, v := range rawConfig {
			var agentID uint
			if _, err := fmt.Sscanf(k, "%d", &agentID); err != nil {
				return nil, fmt.Errorf("failed to parse agent ID in chain_port_config: %w", err)
			}
			chainPortConfig[agentID] = v
		}
	}

	ipVersion := vo.IPVersion(model.IPVersion)

	entity, err := forward.ReconstructForwardRule(
		model.ID,
		model.ShortID,
		model.AgentID,
		userID,
		ruleType,
		exitAgentID,
		chainAgentIDs,
		chainPortConfig,
		model.Name,
		model.ListenPort,
		model.TargetAddress,
		model.TargetPort,
		targetNodeID,
		model.BindIP,
		ipVersion,
		protocol,
		status,
		model.Remark,
		model.UploadBytes,
		model.DownloadBytes,
		model.TrafficMultiplier,
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

	// Handle nullable fields
	var exitAgentID *uint
	if entity.ExitAgentID() != 0 {
		val := entity.ExitAgentID()
		exitAgentID = &val
	}

	var userID *uint
	if entity.UserID() != nil {
		userID = entity.UserID()
	}

	var targetNodeID *uint
	if entity.TargetNodeID() != nil {
		targetNodeID = entity.TargetNodeID()
	}

	// Serialize chain_agent_ids to JSON
	var chainAgentIDsJSON datatypes.JSON
	if len(entity.ChainAgentIDs()) > 0 {
		jsonBytes, err := json.Marshal(entity.ChainAgentIDs())
		if err != nil {
			return nil, fmt.Errorf("failed to serialize chain_agent_ids: %w", err)
		}
		chainAgentIDsJSON = jsonBytes
	}

	// Serialize chain_port_config to JSON
	// Convert map[uint]uint16 to map[string]uint16 for JSON storage
	var chainPortConfigJSON datatypes.JSON
	if len(entity.ChainPortConfig()) > 0 {
		stringKeyConfig := make(map[string]uint16)
		for k, v := range entity.ChainPortConfig() {
			stringKeyConfig[fmt.Sprintf("%d", k)] = v
		}
		jsonBytes, err := json.Marshal(stringKeyConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize chain_port_config: %w", err)
		}
		chainPortConfigJSON = jsonBytes
	}

	return &models.ForwardRuleModel{
		ID:                entity.ID(),
		ShortID:           entity.ShortID(),
		AgentID:           entity.AgentID(),
		UserID:            userID,
		RuleType:          entity.RuleType().String(),
		ExitAgentID:       exitAgentID,
		ChainAgentIDs:     chainAgentIDsJSON,
		ChainPortConfig:   chainPortConfigJSON,
		Name:              entity.Name(),
		ListenPort:        entity.ListenPort(),
		TargetAddress:     entity.TargetAddress(),
		TargetPort:        entity.TargetPort(),
		TargetNodeID:      targetNodeID,
		BindIP:            entity.BindIP(),
		IPVersion:         entity.IPVersion().String(),
		Protocol:          entity.Protocol().String(),
		Status:            entity.Status().String(),
		Remark:            entity.Remark(),
		UploadBytes:       entity.GetRawUploadBytes(),
		DownloadBytes:     entity.GetRawDownloadBytes(),
		TrafficMultiplier: entity.GetTrafficMultiplier(),
		CreatedAt:         entity.CreatedAt(),
		UpdatedAt:         entity.UpdatedAt(),
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
