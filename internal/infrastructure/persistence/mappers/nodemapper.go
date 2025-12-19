package mappers

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/id"
)

// Note: Protocol-specific configs are now stored in separate tables:
// - shadowsocks_configs for Shadowsocks protocol
// - trojan_configs for Trojan protocol
// NodeMapper receives these configs as parameters rather than reading from NodeModel.

// NodeMapper handles the conversion between domain entities and persistence models
type NodeMapper interface {
	// ToEntity converts a persistence model to a domain entity
	// Protocol-specific configs are loaded separately from their respective tables
	ToEntity(model *models.NodeModel, encryptionConfig vo.EncryptionConfig, pluginConfig *vo.PluginConfig, trojanConfig *vo.TrojanConfig) (*node.Node, error)

	// ToModel converts a domain entity to a persistence model
	// Note: Protocol-specific configs are handled separately via their respective mappers
	ToModel(entity *node.Node) (*models.NodeModel, error)

	// ToEntities converts multiple persistence models to domain entities
	// ssConfigs is a map of nodeID -> ShadowsocksConfigData
	// trojanConfigs is a map of nodeID -> TrojanConfig
	ToEntities(models []*models.NodeModel, ssConfigs map[uint]*ShadowsocksConfigData, trojanConfigs map[uint]*vo.TrojanConfig) ([]*node.Node, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*node.Node) ([]*models.NodeModel, error)
}

// ShadowsocksConfigData holds encryption and plugin config data
type ShadowsocksConfigData struct {
	EncryptionConfig vo.EncryptionConfig
	PluginConfig     *vo.PluginConfig
}

// NodeMapperImpl is the concrete implementation of NodeMapper
type NodeMapperImpl struct{}

// NewNodeMapper creates a new node mapper
func NewNodeMapper() NodeMapper {
	return &NodeMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity
// Protocol-specific configs are loaded separately and passed in
func (m *NodeMapperImpl) ToEntity(model *models.NodeModel, encryptionConfig vo.EncryptionConfig, pluginConfig *vo.PluginConfig, trojanConfig *vo.TrojanConfig) (*node.Node, error) {
	if model == nil {
		return nil, nil
	}

	// Convert ServerAddress value object
	serverAddress, err := vo.NewServerAddress(model.ServerAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create server address value object: %w", err)
	}

	// Convert Protocol value object
	protocol := vo.Protocol(model.Protocol)
	if !protocol.IsValid() {
		return nil, fmt.Errorf("invalid protocol: %s", model.Protocol)
	}

	// Convert NodeStatus value object
	nodeStatus := vo.NodeStatus(model.Status)
	if !nodeStatus.IsValid() {
		return nil, fmt.Errorf("invalid node status: %s", model.Status)
	}

	// Parse tags from JSON
	var tags []string
	if model.Tags != nil {
		if err := json.Unmarshal(model.Tags, &tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}

	// Parse subscription plan IDs from JSON
	var planIDs []uint
	if model.PlanIDs != nil {
		if err := json.Unmarshal(model.PlanIDs, &planIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal plan_ids: %w", err)
		}
	}

	// Get region value
	region := ""
	if model.Region != nil {
		region = *model.Region
	}

	// Create NodeMetadata value object
	metadata := vo.NewNodeMetadata(region, tags, "")

	// Generate SID if not present (for legacy nodes without sid)
	sid := model.SID
	if sid == "" {
		sid = id.MustGenerate(id.DefaultLength)
	}

	// Reconstruct the domain entity
	// Protocol-specific configs are passed from caller
	nodeEntity, err := node.ReconstructNode(
		model.ID,
		sid,
		model.Name,
		serverAddress,
		model.AgentPort,
		model.SubscriptionPort,
		protocol,
		encryptionConfig,
		pluginConfig,
		trojanConfig,
		nodeStatus,
		metadata,
		model.GroupID,
		planIDs,
		model.TokenHash,
		model.APIToken,
		model.SortOrder,
		model.MaintenanceReason,
		model.LastSeenAt,
		model.PublicIPv4,
		model.PublicIPv6,
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct node entity: %w", err)
	}

	return nodeEntity, nil
}

// ToModel converts a domain entity to a persistence model
// Note: Protocol-specific configs are handled separately via their respective mappers
func (m *NodeMapperImpl) ToModel(entity *node.Node) (*models.NodeModel, error) {
	if entity == nil {
		return nil, nil
	}

	// Prepare tags JSON
	var tagsJSON datatypes.JSON
	tags := entity.Metadata().Tags()
	if len(tags) > 0 {
		tagsBytes, err := json.Marshal(tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tags: %w", err)
		}
		tagsJSON = tagsBytes
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

	// Prepare region
	var region *string
	if entity.Metadata().Region() != "" {
		r := entity.Metadata().Region()
		region = &r
	}

	model := &models.NodeModel{
		ID:                entity.ID(),
		SID:               entity.SID(),
		Name:              entity.Name(),
		ServerAddress:     entity.ServerAddress().Value(),
		AgentPort:         entity.AgentPort(),
		SubscriptionPort:  entity.SubscriptionPort(),
		Protocol:          entity.Protocol().String(),
		Status:            entity.Status().String(),
		GroupID:           entity.GroupID(),
		Region:            region,
		Tags:              tagsJSON,
		PlanIDs:           planIDsJSON,
		SortOrder:         entity.SortOrder(),
		MaintenanceReason: entity.MaintenanceReason(),
		TokenHash:         entity.TokenHash(),
		APIToken:          entity.GetAPIToken(),
		Version:           entity.Version(),
		CreatedAt:         entity.CreatedAt(),
		UpdatedAt:         entity.UpdatedAt(),
	}

	// Handle soft delete
	if entity.Status().String() == "deleted" {
		now := entity.UpdatedAt()
		model.DeletedAt = gorm.DeletedAt{
			Time:  now,
			Valid: true,
		}
	}

	return model, nil
}

// ToEntities converts multiple persistence models to domain entities
// ssConfigs is a map of nodeID -> ShadowsocksConfigData
// trojanConfigs is a map of nodeID -> TrojanConfig
func (m *NodeMapperImpl) ToEntities(nodeModels []*models.NodeModel, ssConfigs map[uint]*ShadowsocksConfigData, trojanConfigs map[uint]*vo.TrojanConfig) ([]*node.Node, error) {
	entities := make([]*node.Node, 0, len(nodeModels))

	for _, model := range nodeModels {
		// Get protocol-specific configs for this node
		var encryptionConfig vo.EncryptionConfig
		var pluginConfig *vo.PluginConfig
		var trojanConfig *vo.TrojanConfig

		switch model.Protocol {
		case "shadowsocks":
			if ssConfigs != nil {
				if ssData := ssConfigs[model.ID]; ssData != nil {
					encryptionConfig = ssData.EncryptionConfig
					pluginConfig = ssData.PluginConfig
				}
			}
		case "trojan":
			if trojanConfigs != nil {
				trojanConfig = trojanConfigs[model.ID]
			}
		}

		entity, err := m.ToEntity(model, encryptionConfig, pluginConfig, trojanConfig)
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
func (m *NodeMapperImpl) ToModels(entities []*node.Node) ([]*models.NodeModel, error) {
	models := make([]*models.NodeModel, 0, len(entities))

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
