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

	var subscriptionID *uint
	if model.SubscriptionID != nil {
		subscriptionID = model.SubscriptionID
	}

	var targetNodeID *uint
	if model.TargetNodeID != nil {
		targetNodeID = model.TargetNodeID
	}

	// Parse chain_agent_ids JSON
	var chainAgentIDs []uint
	if len(model.ChainAgentIDs) > 0 {
		if err := json.Unmarshal(model.ChainAgentIDs, &chainAgentIDs); err != nil {
			return nil, fmt.Errorf("failed to parse chain_agent_ids: %w", err)
		}
	}

	// Parse chain_port_config JSON
	var chainPortConfig map[uint]uint16
	if len(model.ChainPortConfig) > 0 {
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

	// Parse group_ids JSON
	var groupIDs []uint
	if len(model.GroupIDs) > 0 {
		if err := json.Unmarshal(model.GroupIDs, &groupIDs); err != nil {
			return nil, fmt.Errorf("failed to parse group_ids: %w", err)
		}
	}

	ipVersion := vo.IPVersion(model.IPVersion)
	tunnelType := vo.TunnelType(model.TunnelType)

	// Handle external rule fields
	var serverAddress string
	if model.ServerAddress != nil {
		serverAddress = *model.ServerAddress
	}
	var externalSource string
	if model.ExternalSource != nil {
		externalSource = *model.ExternalSource
	}
	var externalRuleID string
	if model.ExternalRuleID != nil {
		externalRuleID = *model.ExternalRuleID
	}

	entity, err := forward.ReconstructForwardRule(
		model.ID,
		model.SID,
		model.AgentID,
		userID,
		subscriptionID,
		ruleType,
		exitAgentID,
		chainAgentIDs,
		chainPortConfig,
		model.TunnelHops,
		tunnelType,
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
		model.SortOrder,
		groupIDs,
		serverAddress,
		externalSource,
		externalRuleID,
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

	var subscriptionID *uint
	if entity.SubscriptionID() != nil {
		subscriptionID = entity.SubscriptionID()
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

	// Serialize group_ids to JSON
	var groupIDsJSON datatypes.JSON
	if len(entity.GroupIDs()) > 0 {
		jsonBytes, err := json.Marshal(entity.GroupIDs())
		if err != nil {
			return nil, fmt.Errorf("failed to serialize group_ids: %w", err)
		}
		groupIDsJSON = jsonBytes
	}

	// Handle external rule fields
	var serverAddress *string
	if entity.ServerAddress() != "" {
		s := entity.ServerAddress()
		serverAddress = &s
	}
	var externalSource *string
	if entity.ExternalSource() != "" {
		s := entity.ExternalSource()
		externalSource = &s
	}
	var externalRuleID *string
	if entity.ExternalRuleID() != "" {
		s := entity.ExternalRuleID()
		externalRuleID = &s
	}

	return &models.ForwardRuleModel{
		ID:                entity.ID(),
		SID:               entity.SID(),
		AgentID:           entity.AgentID(),
		UserID:            userID,
		SubscriptionID:    subscriptionID,
		RuleType:          entity.RuleType().String(),
		ExitAgentID:       exitAgentID,
		ChainAgentIDs:     chainAgentIDsJSON,
		ChainPortConfig:   chainPortConfigJSON,
		TunnelHops:        entity.TunnelHops(),
		TunnelType:        entity.TunnelType().String(),
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
		SortOrder:         entity.SortOrder(),
		GroupIDs:          groupIDsJSON,
		ServerAddress:     serverAddress,
		ExternalSource:    externalSource,
		ExternalRuleID:    externalRuleID,
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
