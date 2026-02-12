package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/errors"
)

// GetByID retrieves a node by its ID
func (r *NodeRepositoryImpl) GetByID(ctx context.Context, id uint) (*node.Node, error) {
	var model models.NodeModel

	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("node not found")
		}
		r.logger.Errorw("failed to get node by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Load protocol-specific config
	var trojanConfig *vo.TrojanConfig
	var encryptionConfig vo.EncryptionConfig
	var pluginConfig *vo.PluginConfig
	var vlessConfig *vo.VLESSConfig
	var vmessConfig *vo.VMessConfig
	var hysteria2Config *vo.Hysteria2Config
	var tuicConfig *vo.TUICConfig
	var anytlsConfig *vo.AnyTLSConfig

	switch model.Protocol {
	case "shadowsocks":
		var err error
		encryptionConfig, pluginConfig, err = r.shadowsocksConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get shadowsocks config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get shadowsocks config: %w", err)
		}
	case "trojan":
		var err error
		trojanConfig, err = r.trojanConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get trojan config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get trojan config: %w", err)
		}
	case "vless":
		var err error
		vlessConfig, err = r.vlessConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get vless config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get vless config: %w", err)
		}
	case "vmess":
		var err error
		vmessConfig, err = r.vmessConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get vmess config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get vmess config: %w", err)
		}
	case "hysteria2":
		var err error
		hysteria2Config, err = r.hysteria2ConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get hysteria2 config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get hysteria2 config: %w", err)
		}
	case "tuic":
		var err error
		tuicConfig, err = r.tuicConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get tuic config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get tuic config: %w", err)
		}
	case "anytls":
		var err error
		anytlsConfig, err = r.anytlsConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get anytls config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get anytls config: %w", err)
		}
	}

	entity, err := r.mapper.ToEntity(&model, encryptionConfig, pluginConfig, trojanConfig, vlessConfig, vmessConfig, hysteria2Config, tuicConfig, anytlsConfig)
	if err != nil {
		r.logger.Errorw("failed to map node model to entity", "id", id, "error", err)
		return nil, fmt.Errorf("failed to map node: %w", err)
	}

	return entity, nil
}

// GetBySID retrieves a node by its SID
func (r *NodeRepositoryImpl) GetBySID(ctx context.Context, sid string) (*node.Node, error) {
	var model models.NodeModel

	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("node not found")
		}
		r.logger.Errorw("failed to get node by SID", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Load protocol-specific config
	var trojanConfig *vo.TrojanConfig
	var encryptionConfig vo.EncryptionConfig
	var pluginConfig *vo.PluginConfig
	var vlessConfig *vo.VLESSConfig
	var vmessConfig *vo.VMessConfig
	var hysteria2Config *vo.Hysteria2Config
	var tuicConfig *vo.TUICConfig
	var anytlsConfig *vo.AnyTLSConfig

	switch model.Protocol {
	case "shadowsocks":
		var err error
		encryptionConfig, pluginConfig, err = r.shadowsocksConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get shadowsocks config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get shadowsocks config: %w", err)
		}
	case "trojan":
		var err error
		trojanConfig, err = r.trojanConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get trojan config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get trojan config: %w", err)
		}
	case "vless":
		var err error
		vlessConfig, err = r.vlessConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get vless config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get vless config: %w", err)
		}
	case "vmess":
		var err error
		vmessConfig, err = r.vmessConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get vmess config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get vmess config: %w", err)
		}
	case "hysteria2":
		var err error
		hysteria2Config, err = r.hysteria2ConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get hysteria2 config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get hysteria2 config: %w", err)
		}
	case "tuic":
		var err error
		tuicConfig, err = r.tuicConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get tuic config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get tuic config: %w", err)
		}
	case "anytls":
		var err error
		anytlsConfig, err = r.anytlsConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get anytls config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get anytls config: %w", err)
		}
	}

	entity, err := r.mapper.ToEntity(&model, encryptionConfig, pluginConfig, trojanConfig, vlessConfig, vmessConfig, hysteria2Config, tuicConfig, anytlsConfig)
	if err != nil {
		r.logger.Errorw("failed to map node model to entity", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to map node: %w", err)
	}

	return entity, nil
}

// GetBySIDs retrieves nodes by their SIDs
func (r *NodeRepositoryImpl) GetBySIDs(ctx context.Context, sids []string) ([]*node.Node, error) {
	if len(sids) == 0 {
		return []*node.Node{}, nil
	}

	var nodeModels []*models.NodeModel
	if err := r.db.WithContext(ctx).Where("sid IN ?", sids).Find(&nodeModels).Error; err != nil {
		r.logger.Errorw("failed to get nodes by SIDs", "sids", sids, "error", err)
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	// Collect node IDs for batch loading protocol configs
	nodeIDs := make([]uint, len(nodeModels))
	for i, m := range nodeModels {
		nodeIDs[i] = m.ID
	}

	// Load shadowsocks configs
	ssConfigsRaw, err := r.shadowsocksConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load shadowsocks configs", "error", err)
		ssConfigsRaw = make(map[uint]*ShadowsocksConfigData)
	}

	// Convert to mapper format
	ssConfigs := make(map[uint]*mappers.ShadowsocksConfigData)
	for nodeID, data := range ssConfigsRaw {
		ssConfigs[nodeID] = &mappers.ShadowsocksConfigData{
			EncryptionConfig: data.EncryptionConfig,
			PluginConfig:     data.PluginConfig,
		}
	}

	// Load trojan configs
	trojanConfigs, err := r.trojanConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load trojan configs", "error", err)
		trojanConfigs = make(map[uint]*vo.TrojanConfig)
	}

	// Load vless configs
	vlessConfigs, err := r.vlessConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load vless configs", "error", err)
		vlessConfigs = make(map[uint]*vo.VLESSConfig)
	}

	// Load vmess configs
	vmessConfigs, err := r.vmessConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load vmess configs", "error", err)
		vmessConfigs = make(map[uint]*vo.VMessConfig)
	}

	// Load hysteria2 configs
	hysteria2Configs, err := r.hysteria2ConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load hysteria2 configs", "error", err)
		hysteria2Configs = make(map[uint]*vo.Hysteria2Config)
	}

	// Load tuic configs
	tuicConfigs, err := r.tuicConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load tuic configs", "error", err)
		tuicConfigs = make(map[uint]*vo.TUICConfig)
	}

	// Load anytls configs
	anytlsConfigs, err := r.anytlsConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load anytls configs", "error", err)
		anytlsConfigs = make(map[uint]*vo.AnyTLSConfig)
	}

	// Convert to entities
	entities, err := r.mapper.ToEntities(nodeModels, ssConfigs, trojanConfigs, vlessConfigs, vmessConfigs, hysteria2Configs, tuicConfigs, anytlsConfigs)
	if err != nil {
		r.logger.Errorw("failed to map node models to entities", "error", err)
		return nil, fmt.Errorf("failed to map nodes: %w", err)
	}

	return entities, nil
}

// GetByIDs retrieves nodes by their IDs
func (r *NodeRepositoryImpl) GetByIDs(ctx context.Context, ids []uint) ([]*node.Node, error) {
	if len(ids) == 0 {
		return []*node.Node{}, nil
	}

	var nodeModels []*models.NodeModel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&nodeModels).Error; err != nil {
		r.logger.Errorw("failed to get nodes by IDs", "ids", ids, "error", err)
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	// Collect node IDs for batch loading protocol configs
	nodeIDs := make([]uint, len(nodeModels))
	for i, m := range nodeModels {
		nodeIDs[i] = m.ID
	}

	// Load shadowsocks configs
	ssConfigsRaw, err := r.shadowsocksConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load shadowsocks configs", "error", err)
		ssConfigsRaw = make(map[uint]*ShadowsocksConfigData)
	}

	// Convert to mapper format
	ssConfigs := make(map[uint]*mappers.ShadowsocksConfigData)
	for nodeID, data := range ssConfigsRaw {
		ssConfigs[nodeID] = &mappers.ShadowsocksConfigData{
			EncryptionConfig: data.EncryptionConfig,
			PluginConfig:     data.PluginConfig,
		}
	}

	// Load trojan configs
	trojanConfigs, err := r.trojanConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load trojan configs", "error", err)
		trojanConfigs = make(map[uint]*vo.TrojanConfig)
	}

	// Load vless configs
	vlessConfigs, err := r.vlessConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load vless configs", "error", err)
		vlessConfigs = make(map[uint]*vo.VLESSConfig)
	}

	// Load vmess configs
	vmessConfigs, err := r.vmessConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load vmess configs", "error", err)
		vmessConfigs = make(map[uint]*vo.VMessConfig)
	}

	// Load hysteria2 configs
	hysteria2Configs, err := r.hysteria2ConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load hysteria2 configs", "error", err)
		hysteria2Configs = make(map[uint]*vo.Hysteria2Config)
	}

	// Load tuic configs
	tuicConfigs, err := r.tuicConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load tuic configs", "error", err)
		tuicConfigs = make(map[uint]*vo.TUICConfig)
	}

	// Load anytls configs
	anytlsConfigs, err := r.anytlsConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load anytls configs", "error", err)
		anytlsConfigs = make(map[uint]*vo.AnyTLSConfig)
	}

	// Convert to entities
	entities, err := r.mapper.ToEntities(nodeModels, ssConfigs, trojanConfigs, vlessConfigs, vmessConfigs, hysteria2Configs, tuicConfigs, anytlsConfigs)
	if err != nil {
		r.logger.Errorw("failed to map node models to entities", "error", err)
		return nil, fmt.Errorf("failed to map nodes: %w", err)
	}

	return entities, nil
}

// GetByToken retrieves a node by its API token hash
func (r *NodeRepositoryImpl) GetByToken(ctx context.Context, tokenHash string) (*node.Node, error) {
	var model models.NodeModel

	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("node not found")
		}
		r.logger.Errorw("failed to get node by token", "error", err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Load protocol-specific config
	var trojanConfig *vo.TrojanConfig
	var encryptionConfig vo.EncryptionConfig
	var pluginConfig *vo.PluginConfig
	var vlessConfig *vo.VLESSConfig
	var vmessConfig *vo.VMessConfig
	var hysteria2Config *vo.Hysteria2Config
	var tuicConfig *vo.TUICConfig
	var anytlsConfig *vo.AnyTLSConfig

	switch model.Protocol {
	case "shadowsocks":
		var err error
		encryptionConfig, pluginConfig, err = r.shadowsocksConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get shadowsocks config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get shadowsocks config: %w", err)
		}
	case "trojan":
		var err error
		trojanConfig, err = r.trojanConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get trojan config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get trojan config: %w", err)
		}
	case "vless":
		var err error
		vlessConfig, err = r.vlessConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get vless config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get vless config: %w", err)
		}
	case "vmess":
		var err error
		vmessConfig, err = r.vmessConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get vmess config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get vmess config: %w", err)
		}
	case "hysteria2":
		var err error
		hysteria2Config, err = r.hysteria2ConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get hysteria2 config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get hysteria2 config: %w", err)
		}
	case "tuic":
		var err error
		tuicConfig, err = r.tuicConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get tuic config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get tuic config: %w", err)
		}
	case "anytls":
		var err error
		anytlsConfig, err = r.anytlsConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get anytls config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get anytls config: %w", err)
		}
	}

	entity, err := r.mapper.ToEntity(&model, encryptionConfig, pluginConfig, trojanConfig, vlessConfig, vmessConfig, hysteria2Config, tuicConfig, anytlsConfig)
	if err != nil {
		r.logger.Errorw("failed to map node model to entity", "token_hash", tokenHash, "error", err)
		return nil, fmt.Errorf("failed to map node: %w", err)
	}

	return entity, nil
}
