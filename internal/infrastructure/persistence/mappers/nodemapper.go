package mappers

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"orris/internal/domain/node"
	vo "orris/internal/domain/node/value_objects"
	"orris/internal/infrastructure/persistence/models"
)

// NodeMapper handles the conversion between domain entities and persistence models
type NodeMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.NodeModel) (*node.Node, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *node.Node) (*models.NodeModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.NodeModel) ([]*node.Node, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*node.Node) ([]*models.NodeModel, error)
}

// NodeMapperImpl is the concrete implementation of NodeMapper
type NodeMapperImpl struct{}

// NewNodeMapper creates a new node mapper
func NewNodeMapper() NodeMapper {
	return &NodeMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity
func (m *NodeMapperImpl) ToEntity(model *models.NodeModel) (*node.Node, error) {
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

	// Convert EncryptionConfig value object
	encryptionConfig, err := vo.NewEncryptionConfig(model.EncryptionMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryption config value object: %w", err)
	}

	// Convert PluginConfig value object (nullable)
	var pluginConfig *vo.PluginConfig
	if model.Plugin != nil && *model.Plugin != "" {
		var opts map[string]string
		if model.PluginOpts != nil {
			if err := json.Unmarshal(model.PluginOpts, &opts); err != nil {
				return nil, fmt.Errorf("failed to unmarshal plugin opts: %w", err)
			}
		}
		pluginConfig, err = vo.NewPluginConfig(*model.Plugin, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create plugin config value object: %w", err)
		}
	}

	// Convert TrojanConfig value object (nullable, stored in CustomFields)
	var trojanConfig *vo.TrojanConfig
	if protocol.IsTrojan() && model.CustomFields != nil {
		var customFields map[string]interface{}
		if err := json.Unmarshal(model.CustomFields, &customFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal custom fields: %w", err)
		}

		if trojanData, ok := customFields["trojan_config"].(map[string]interface{}); ok {
			password := ""
			if p, ok := trojanData["password"].(string); ok {
				password = p
			}
			transportProtocol := ""
			if t, ok := trojanData["transport_protocol"].(string); ok {
				transportProtocol = t
			}
			host := ""
			if h, ok := trojanData["host"].(string); ok {
				host = h
			}
			path := ""
			if p, ok := trojanData["path"].(string); ok {
				path = p
			}
			allowInsecure := false
			if a, ok := trojanData["allow_insecure"].(bool); ok {
				allowInsecure = a
			}
			sni := ""
			if s, ok := trojanData["sni"].(string); ok {
				sni = s
			}

			tc, err := vo.NewTrojanConfig(password, transportProtocol, host, path, allowInsecure, sni)
			if err != nil {
				return nil, fmt.Errorf("failed to create trojan config value object: %w", err)
			}
			trojanConfig = &tc
		}
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

	// Get region value
	region := ""
	if model.Region != nil {
		region = *model.Region
	}

	// Create NodeMetadata value object
	metadata := vo.NewNodeMetadata(region, tags, "")

	// Reconstruct the domain entity
	nodeEntity, err := node.ReconstructNode(
		model.ID,
		model.Name,
		serverAddress,
		model.ServerPort,
		protocol,
		encryptionConfig,
		pluginConfig,
		trojanConfig,
		nodeStatus,
		metadata,
		model.TokenHash,
		model.SortOrder,
		model.MaintenanceReason,
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
func (m *NodeMapperImpl) ToModel(entity *node.Node) (*models.NodeModel, error) {
	if entity == nil {
		return nil, nil
	}

	// Prepare plugin name and opts
	var plugin *string
	var pluginOptsJSON datatypes.JSON
	if entity.PluginConfig() != nil {
		pluginName := entity.PluginConfig().Plugin()
		plugin = &pluginName

		opts := entity.PluginConfig().Opts()
		if len(opts) > 0 {
			optsBytes, err := json.Marshal(opts)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal plugin opts: %w", err)
			}
			pluginOptsJSON = optsBytes
		}
	}

	// Prepare custom fields JSON with TrojanConfig if present
	var customFieldsJSON datatypes.JSON
	if entity.TrojanConfig() != nil {
		customFields := map[string]interface{}{
			"trojan_config": map[string]interface{}{
				"password":           entity.TrojanConfig().Password(),
				"transport_protocol": entity.TrojanConfig().TransportProtocol(),
				"host":               entity.TrojanConfig().Host(),
				"path":               entity.TrojanConfig().Path(),
				"allow_insecure":     entity.TrojanConfig().AllowInsecure(),
				"sni":                entity.TrojanConfig().SNI(),
			},
		}
		customFieldsBytes, err := json.Marshal(customFields)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal custom fields: %w", err)
		}
		customFieldsJSON = customFieldsBytes
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

	// Prepare region
	var region *string
	if entity.Metadata().Region() != "" {
		r := entity.Metadata().Region()
		region = &r
	}

	model := &models.NodeModel{
		ID:                entity.ID(),
		Name:              entity.Name(),
		ServerAddress:     entity.ServerAddress().Value(),
		ServerPort:        entity.ServerPort(),
		EncryptionMethod:  entity.EncryptionConfig().Method(),
		Plugin:            plugin,
		PluginOpts:        pluginOptsJSON,
		Protocol:          entity.Protocol().String(),
		Status:            entity.Status().String(),
		Region:            region,
		Tags:              tagsJSON,
		CustomFields:      customFieldsJSON,
		SortOrder:         entity.SortOrder(),
		MaintenanceReason: entity.MaintenanceReason(),
		TokenHash:         entity.TokenHash(),
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
func (m *NodeMapperImpl) ToEntities(models []*models.NodeModel) ([]*node.Node, error) {
	entities := make([]*node.Node, 0, len(models))

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
