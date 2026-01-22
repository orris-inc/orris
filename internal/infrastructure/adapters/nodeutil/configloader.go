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

// LoadProtocolConfigs loads trojan and shadowsocks configs for the given node models.
func (l *ConfigLoader) LoadProtocolConfigs(ctx context.Context, nodeModels []models.NodeModel) ProtocolConfigs {
	configs := NewProtocolConfigs()

	// Collect node IDs by protocol
	trojanNodeIDs, ssNodeIDs := classifyNodesByProtocol(nodeModels)

	// Load configs using unified loader pattern
	l.loadConfigsIntoMap(ctx, trojanNodeIDs, &configs.Trojan)
	l.loadConfigsIntoMap(ctx, ssNodeIDs, &configs.Shadowsocks)

	return configs
}

// classifyNodesByProtocol separates node IDs by their protocol type.
func classifyNodesByProtocol(nodeModels []models.NodeModel) (trojanIDs, ssIDs []uint) {
	for _, nm := range nodeModels {
		switch nm.Protocol {
		case "trojan":
			trojanIDs = append(trojanIDs, nm.ID)
		case "shadowsocks", "":
			ssIDs = append(ssIDs, nm.ID)
		}
	}
	return
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
