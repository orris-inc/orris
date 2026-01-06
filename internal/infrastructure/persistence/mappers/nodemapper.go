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

// RouteConfigJSON represents the JSON structure for RouteConfig persistence
type RouteConfigJSON struct {
	Rules       []RouteRuleJSON `json:"rules,omitempty"`
	FinalAction string          `json:"final_action"`
}

// RouteRuleJSON represents the JSON structure for a single routing rule
type RouteRuleJSON struct {
	Domain        []string `json:"domain,omitempty"`
	DomainSuffix  []string `json:"domain_suffix,omitempty"`
	DomainKeyword []string `json:"domain_keyword,omitempty"`
	DomainRegex   []string `json:"domain_regex,omitempty"`
	IPCIDR        []string `json:"ip_cidr,omitempty"`
	SourceIPCIDR  []string `json:"source_ip_cidr,omitempty"`
	IPIsPrivate   bool     `json:"ip_is_private,omitempty"`
	GeoIP         []string `json:"geoip,omitempty"`
	GeoSite       []string `json:"geosite,omitempty"`
	Port          []uint16 `json:"port,omitempty"`
	SourcePort    []uint16 `json:"source_port,omitempty"`
	Protocol      []string `json:"protocol,omitempty"`
	Network       []string `json:"network,omitempty"`
	RuleSet       []string `json:"rule_set,omitempty"`
	Outbound      string   `json:"outbound"`
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

	// Parse groupIDs from JSON
	var groupIDs []uint
	if len(model.GroupIDs) > 0 {
		if err := json.Unmarshal(model.GroupIDs, &groupIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal group_ids: %w", err)
		}
	}

	// Parse routeConfig from JSON
	var routeConfig *vo.RouteConfig
	if len(model.RouteConfig) > 0 {
		var routeJSON RouteConfigJSON
		if err := json.Unmarshal(model.RouteConfig, &routeJSON); err != nil {
			return nil, fmt.Errorf("failed to unmarshal route_config: %w", err)
		}
		routeConfig = routeConfigFromJSON(&routeJSON)
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
		groupIDs,
		model.UserID,
		model.TokenHash,
		model.APIToken,
		model.SortOrder,
		model.MuteNotification,
		model.MaintenanceReason,
		routeConfig,
		model.LastSeenAt,
		model.PublicIPv4,
		model.PublicIPv6,
		model.AgentVersion,
		model.Platform,
		model.Arch,
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

	// Prepare region
	var region *string
	if entity.Metadata().Region() != "" {
		r := entity.Metadata().Region()
		region = &r
	}

	// Prepare groupIDs JSON
	var groupIDsJSON datatypes.JSON
	groupIDs := entity.GroupIDs()
	if len(groupIDs) > 0 {
		groupIDsBytes, err := json.Marshal(groupIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal group_ids: %w", err)
		}
		groupIDsJSON = groupIDsBytes
	}

	// Prepare routeConfig JSON
	var routeConfigJSON datatypes.JSON
	if entity.RouteConfig() != nil {
		routeJSON := routeConfigToJSON(entity.RouteConfig())
		routeBytes, err := json.Marshal(routeJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal route_config: %w", err)
		}
		routeConfigJSON = routeBytes
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
		GroupIDs:          groupIDsJSON,
		UserID:            entity.UserID(),
		Region:            region,
		Tags:              tagsJSON,
		SortOrder:         entity.SortOrder(),
		MuteNotification:  entity.MuteNotification(),
		MaintenanceReason: entity.MaintenanceReason(),
		RouteConfig:       routeConfigJSON,
		TokenHash:         entity.TokenHash(),
		APIToken:          entity.GetAPIToken(),
		AgentVersion:      entity.AgentVersion(),
		Platform:          entity.AgentPlatform(),
		Arch:              entity.AgentArch(),
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

// routeConfigToJSON converts domain RouteConfig to JSON structure
func routeConfigToJSON(rc *vo.RouteConfig) *RouteConfigJSON {
	if rc == nil {
		return nil
	}

	rules := make([]RouteRuleJSON, 0, len(rc.Rules()))
	for _, rule := range rc.Rules() {
		rules = append(rules, RouteRuleJSON{
			Domain:        rule.Domain(),
			DomainSuffix:  rule.DomainSuffix(),
			DomainKeyword: rule.DomainKeyword(),
			DomainRegex:   rule.DomainRegex(),
			IPCIDR:        rule.IPCIDR(),
			SourceIPCIDR:  rule.SourceIPCIDR(),
			IPIsPrivate:   rule.IPIsPrivate(),
			GeoIP:         rule.GeoIP(),
			GeoSite:       rule.GeoSite(),
			Port:          rule.Port(),
			SourcePort:    rule.SourcePort(),
			Protocol:      rule.Protocol(),
			Network:       rule.Network(),
			RuleSet:       rule.RuleSet(),
			Outbound:      rule.Outbound().String(),
		})
	}

	return &RouteConfigJSON{
		Rules:       rules,
		FinalAction: rc.FinalAction().String(),
	}
}

// routeConfigFromJSON converts JSON structure to domain RouteConfig
func routeConfigFromJSON(rcJSON *RouteConfigJSON) *vo.RouteConfig {
	if rcJSON == nil {
		return nil
	}

	finalAction := vo.OutboundType(rcJSON.FinalAction)
	if !finalAction.IsValid() {
		finalAction = vo.OutboundDirect // default fallback to direct
	}

	rules := make([]vo.RouteRule, 0, len(rcJSON.Rules))
	for _, ruleJSON := range rcJSON.Rules {
		outbound := vo.OutboundType(ruleJSON.Outbound)
		if !outbound.IsValid() {
			outbound = vo.OutboundDirect // default fallback to direct
		}

		rule := vo.ReconstructRouteRule(
			ruleJSON.Domain,
			ruleJSON.DomainSuffix,
			ruleJSON.DomainKeyword,
			ruleJSON.DomainRegex,
			ruleJSON.IPCIDR,
			ruleJSON.SourceIPCIDR,
			ruleJSON.IPIsPrivate,
			ruleJSON.GeoIP,
			ruleJSON.GeoSite,
			ruleJSON.Port,
			ruleJSON.SourcePort,
			ruleJSON.Protocol,
			ruleJSON.Network,
			ruleJSON.RuleSet,
			outbound,
		)
		rules = append(rules, *rule)
	}

	return vo.ReconstructRouteConfig(rules, finalAction)
}
