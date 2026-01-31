package nodeutil

import (
	"context"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ConfigLoader loads protocol configurations from the database.
type ConfigLoader struct {
	db     *gorm.DB
	logger logger.Interface
}

// NewConfigLoader creates a new ConfigLoader instance.
func NewConfigLoader(db *gorm.DB, logger logger.Interface) *ConfigLoader {
	return &ConfigLoader{
		db:     db,
		logger: logger,
	}
}

// LoadProtocolConfigs loads protocol configs for the given node models.
func (l *ConfigLoader) LoadProtocolConfigs(ctx context.Context, nodeModels []models.NodeModel) ProtocolConfigs {
	configs := NewProtocolConfigs()

	// Collect node IDs by protocol
	nodeIDsByProtocol := classifyNodesByProtocol(nodeModels)

	// Load configs using unified loader pattern
	l.loadConfigsIntoMap(ctx, nodeIDsByProtocol["trojan"], &configs.Trojan)
	l.loadConfigsIntoMap(ctx, nodeIDsByProtocol["shadowsocks"], &configs.Shadowsocks)
	l.loadConfigsIntoMap(ctx, nodeIDsByProtocol["vless"], &configs.VLESS)
	l.loadConfigsIntoMap(ctx, nodeIDsByProtocol["vmess"], &configs.VMess)
	l.loadConfigsIntoMap(ctx, nodeIDsByProtocol["hysteria2"], &configs.Hysteria2)
	l.loadConfigsIntoMap(ctx, nodeIDsByProtocol["tuic"], &configs.TUIC)

	return configs
}

// classifyNodesByProtocol separates node IDs by their protocol type.
func classifyNodesByProtocol(nodeModels []models.NodeModel) map[string][]uint {
	result := make(map[string][]uint)
	for _, nm := range nodeModels {
		protocol := nm.Protocol
		if protocol == "" {
			protocol = "shadowsocks"
		}
		result[protocol] = append(result[protocol], nm.ID)
	}
	return result
}

// loadConfigsIntoMap is a generic loader that loads protocol configs into the provided map.
// T must be a pointer to the config model type.
func (l *ConfigLoader) loadConfigsIntoMap(ctx context.Context, nodeIDs []uint, targetMap any) {
	if len(nodeIDs) == 0 {
		return
	}

	// Use type switch to handle different config types
	switch m := targetMap.(type) {
	case *map[uint]*models.TrojanConfigModel:
		l.loadTrojanConfigs(ctx, nodeIDs, m)
	case *map[uint]*models.ShadowsocksConfigModel:
		l.loadShadowsocksConfigs(ctx, nodeIDs, m)
	case *map[uint]*models.VLESSConfigModel:
		l.loadVLESSConfigs(ctx, nodeIDs, m)
	case *map[uint]*models.VMessConfigModel:
		l.loadVMessConfigs(ctx, nodeIDs, m)
	case *map[uint]*models.Hysteria2ConfigModel:
		l.loadHysteria2Configs(ctx, nodeIDs, m)
	case *map[uint]*models.TUICConfigModel:
		l.loadTUICConfigs(ctx, nodeIDs, m)
	}
}

// loadTrojanConfigs loads trojan configs into the provided map.
func (l *ConfigLoader) loadTrojanConfigs(ctx context.Context, nodeIDs []uint, targetMap *map[uint]*models.TrojanConfigModel) {
	var configs []models.TrojanConfigModel
	if err := l.db.WithContext(ctx).
		Where("node_id IN ?", nodeIDs).
		Find(&configs).Error; err != nil {
		l.logger.Warnw("failed to query trojan configs", "error", err)
		return
	}
	for i := range configs {
		(*targetMap)[configs[i].NodeID] = &configs[i]
	}
}

// loadShadowsocksConfigs loads shadowsocks configs into the provided map.
func (l *ConfigLoader) loadShadowsocksConfigs(ctx context.Context, nodeIDs []uint, targetMap *map[uint]*models.ShadowsocksConfigModel) {
	var configs []models.ShadowsocksConfigModel
	if err := l.db.WithContext(ctx).
		Where("node_id IN ?", nodeIDs).
		Find(&configs).Error; err != nil {
		l.logger.Warnw("failed to query shadowsocks configs", "error", err)
		return
	}
	for i := range configs {
		(*targetMap)[configs[i].NodeID] = &configs[i]
	}
}

// loadVLESSConfigs loads VLESS configs into the provided map.
func (l *ConfigLoader) loadVLESSConfigs(ctx context.Context, nodeIDs []uint, targetMap *map[uint]*models.VLESSConfigModel) {
	var configs []models.VLESSConfigModel
	if err := l.db.WithContext(ctx).
		Where("node_id IN ?", nodeIDs).
		Find(&configs).Error; err != nil {
		l.logger.Warnw("failed to query VLESS configs", "error", err)
		return
	}
	for i := range configs {
		(*targetMap)[configs[i].NodeID] = &configs[i]
	}
}

// loadVMessConfigs loads VMess configs into the provided map.
func (l *ConfigLoader) loadVMessConfigs(ctx context.Context, nodeIDs []uint, targetMap *map[uint]*models.VMessConfigModel) {
	var configs []models.VMessConfigModel
	if err := l.db.WithContext(ctx).
		Where("node_id IN ?", nodeIDs).
		Find(&configs).Error; err != nil {
		l.logger.Warnw("failed to query VMess configs", "error", err)
		return
	}
	for i := range configs {
		(*targetMap)[configs[i].NodeID] = &configs[i]
	}
}

// loadHysteria2Configs loads Hysteria2 configs into the provided map.
func (l *ConfigLoader) loadHysteria2Configs(ctx context.Context, nodeIDs []uint, targetMap *map[uint]*models.Hysteria2ConfigModel) {
	var configs []models.Hysteria2ConfigModel
	if err := l.db.WithContext(ctx).
		Where("node_id IN ?", nodeIDs).
		Find(&configs).Error; err != nil {
		l.logger.Warnw("failed to query Hysteria2 configs", "error", err)
		return
	}
	for i := range configs {
		(*targetMap)[configs[i].NodeID] = &configs[i]
	}
}

// loadTUICConfigs loads TUIC configs into the provided map.
func (l *ConfigLoader) loadTUICConfigs(ctx context.Context, nodeIDs []uint, targetMap *map[uint]*models.TUICConfigModel) {
	var configs []models.TUICConfigModel
	if err := l.db.WithContext(ctx).
		Where("node_id IN ?", nodeIDs).
		Find(&configs).Error; err != nil {
		l.logger.Warnw("failed to query TUIC configs", "error", err)
		return
	}
	for i := range configs {
		(*targetMap)[configs[i].NodeID] = &configs[i]
	}
}
