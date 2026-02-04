package mappers

import (
	"encoding/json"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/mapper"
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

	// Parse allowed port range from JSON string
	var allowedPortRange *vo.PortRange
	if model.AllowedPortRange != nil && *model.AllowedPortRange != "" {
		allowedPortRange = &vo.PortRange{}
		if err := json.Unmarshal([]byte(*model.AllowedPortRange), allowedPortRange); err != nil {
			return nil, fmt.Errorf("failed to parse allowed_port_range: %w", err)
		}
	}

	// Parse blocked protocols from JSON
	var blockedProtocols vo.BlockedProtocols
	if len(model.BlockedProtocols) > 0 {
		var protocols []string
		if err := json.Unmarshal(model.BlockedProtocols, &protocols); err != nil {
			return nil, fmt.Errorf("failed to parse blocked_protocols: %w", err)
		}
		blockedProtocols = vo.NewBlockedProtocols(protocols)
	}

	// Parse group_ids from JSON
	var groupIDs []uint
	if len(model.GroupIDs) > 0 {
		if err := json.Unmarshal(model.GroupIDs, &groupIDs); err != nil {
			return nil, fmt.Errorf("failed to parse group_ids: %w", err)
		}
	}

	entity, err := forward.ReconstructForwardAgent(
		model.ID,
		model.SID,
		model.Name,
		model.TokenHash,
		model.APIToken,
		status,
		model.PublicAddress,
		model.TunnelAddress,
		model.Remark,
		groupIDs,
		model.AgentVersion,
		model.Platform,
		model.Arch,
		allowedPortRange,
		blockedProtocols,
		model.SortOrder,
		model.MuteNotification,
		model.LastSeenAt,
		model.ExpiresAt,
		model.CostLabel,
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

	// Serialize allowed port range to JSON string
	var allowedPortRange *string
	if entity.AllowedPortRange() != nil && !entity.AllowedPortRange().IsEmpty() {
		data, err := json.Marshal(entity.AllowedPortRange())
		if err != nil {
			return nil, fmt.Errorf("failed to serialize allowed_port_range: %w", err)
		}
		jsonStr := string(data)
		allowedPortRange = &jsonStr
	}

	// Serialize blocked protocols to JSON
	var blockedProtocols []byte
	if len(entity.BlockedProtocols()) > 0 {
		var err error
		blockedProtocols, err = json.Marshal(entity.BlockedProtocols().ToStringSlice())
		if err != nil {
			return nil, fmt.Errorf("failed to serialize blocked_protocols: %w", err)
		}
	}

	// Serialize group_ids to JSON
	var groupIDsJSON []byte
	if len(entity.GroupIDs()) > 0 {
		var err error
		groupIDsJSON, err = json.Marshal(entity.GroupIDs())
		if err != nil {
			return nil, fmt.Errorf("failed to serialize group_ids: %w", err)
		}
	}

	return &models.ForwardAgentModel{
		ID:               entity.ID(),
		SID:              entity.SID(),
		Name:             entity.Name(),
		TokenHash:        entity.TokenHash(),
		APIToken:         entity.GetAPIToken(),
		PublicAddress:    entity.PublicAddress(),
		TunnelAddress:    entity.TunnelAddress(),
		Status:           string(entity.Status()),
		Remark:           entity.Remark(),
		GroupIDs:         groupIDsJSON,
		AgentVersion:     entity.AgentVersion(),
		Platform:         entity.Platform(),
		Arch:             entity.Arch(),
		AllowedPortRange: allowedPortRange,
		BlockedProtocols: blockedProtocols,
		SortOrder:        entity.SortOrder(),
		MuteNotification: entity.MuteNotification(),
		LastSeenAt:       entity.LastSeenAt(),
		ExpiresAt:        entity.ExpiresAt(),
		CostLabel:        entity.CostLabel(),
		CreatedAt:        entity.CreatedAt(),
		UpdatedAt:        entity.UpdatedAt(),
	}, nil
}

// ToEntities converts multiple persistence models to domain entities.
func (m *ForwardAgentMapperImpl) ToEntities(modelList []*models.ForwardAgentModel) ([]*forward.ForwardAgent, error) {
	return mapper.MapSlicePtrWithID(modelList, m.ToEntity, func(model *models.ForwardAgentModel) uint { return model.ID })
}
