package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ShadowsocksConfigRepository handles persistence operations for ShadowsocksConfig
type ShadowsocksConfigRepository struct {
	db     *gorm.DB
	mapper mappers.ShadowsocksConfigMapper
	logger logger.Interface
}

// NewShadowsocksConfigRepository creates a new ShadowsocksConfigRepository
func NewShadowsocksConfigRepository(db *gorm.DB, logger logger.Interface) *ShadowsocksConfigRepository {
	return &ShadowsocksConfigRepository{
		db:     db,
		mapper: mappers.NewShadowsocksConfigMapper(),
		logger: logger,
	}
}

// Create creates a new ShadowsocksConfig record for a node
func (r *ShadowsocksConfigRepository) Create(ctx context.Context, nodeID uint, encryptionConfig vo.EncryptionConfig, pluginConfig *vo.PluginConfig) error {
	model, err := r.mapper.ToModel(nodeID, encryptionConfig, pluginConfig)
	if err != nil {
		return fmt.Errorf("failed to map shadowsocks config to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create shadowsocks config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to create shadowsocks config: %w", err)
	}

	r.logger.Infow("shadowsocks config created", "node_id", nodeID, "id", model.ID)
	return nil
}

// GetByNodeID retrieves ShadowsocksConfig for a specific node
func (r *ShadowsocksConfigRepository) GetByNodeID(ctx context.Context, nodeID uint) (vo.EncryptionConfig, *vo.PluginConfig, error) {
	var model models.ShadowsocksConfigModel
	if err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return vo.EncryptionConfig{}, nil, nil
		}
		r.logger.Errorw("failed to get shadowsocks config", "node_id", nodeID, "error", err)
		return vo.EncryptionConfig{}, nil, fmt.Errorf("failed to get shadowsocks config: %w", err)
	}

	return r.mapper.ToValueObjects(&model)
}

// ShadowsocksConfigData holds the encryption and plugin config for a node
type ShadowsocksConfigData struct {
	EncryptionConfig vo.EncryptionConfig
	PluginConfig     *vo.PluginConfig
}

// GetByNodeIDs retrieves ShadowsocksConfigs for multiple nodes
// Returns a map of nodeID -> ShadowsocksConfigData
func (r *ShadowsocksConfigRepository) GetByNodeIDs(ctx context.Context, nodeIDs []uint) (map[uint]*ShadowsocksConfigData, error) {
	if len(nodeIDs) == 0 {
		return make(map[uint]*ShadowsocksConfigData), nil
	}

	var ssModels []models.ShadowsocksConfigModel
	if err := r.db.WithContext(ctx).
		Select("node_id", "encryption_method", "plugin", "plugin_opts").
		Where("node_id IN ?", nodeIDs).
		Find(&ssModels).Error; err != nil {
		r.logger.Errorw("failed to get shadowsocks configs by node IDs", "node_ids", nodeIDs, "error", err)
		return nil, fmt.Errorf("failed to get shadowsocks configs: %w", err)
	}

	result := make(map[uint]*ShadowsocksConfigData)
	for _, model := range ssModels {
		encryptionConfig, pluginConfig, err := r.mapper.ToValueObjects(&model)
		if err != nil {
			r.logger.Warnw("failed to map shadowsocks config", "node_id", model.NodeID, "error", err)
			continue
		}
		result[model.NodeID] = &ShadowsocksConfigData{
			EncryptionConfig: encryptionConfig,
			PluginConfig:     pluginConfig,
		}
	}

	return result, nil
}

// Update updates the ShadowsocksConfig for a node
func (r *ShadowsocksConfigRepository) Update(ctx context.Context, nodeID uint, encryptionConfig vo.EncryptionConfig, pluginConfig *vo.PluginConfig) error {
	// Check if record exists
	var existing models.ShadowsocksConfigModel
	err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		// Create new record
		return r.Create(ctx, nodeID, encryptionConfig, pluginConfig)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing shadowsocks config: %w", err)
	}

	// Update existing record
	model, err := r.mapper.ToModel(nodeID, encryptionConfig, pluginConfig)
	if err != nil {
		return fmt.Errorf("failed to map shadowsocks config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt // Preserve original creation time

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		r.logger.Errorw("failed to update shadowsocks config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to update shadowsocks config: %w", err)
	}

	r.logger.Infow("shadowsocks config updated", "node_id", nodeID)
	return nil
}

// DeleteByNodeID deletes the ShadowsocksConfig for a node
func (r *ShadowsocksConfigRepository) DeleteByNodeID(ctx context.Context, nodeID uint) error {
	result := r.db.WithContext(ctx).Where("node_id = ?", nodeID).Delete(&models.ShadowsocksConfigModel{})
	if result.Error != nil {
		r.logger.Errorw("failed to delete shadowsocks config", "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to delete shadowsocks config: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		r.logger.Infow("shadowsocks config deleted", "node_id", nodeID)
	}
	return nil
}

// CreateInTx creates a ShadowsocksConfig record within a transaction
func (r *ShadowsocksConfigRepository) CreateInTx(tx *gorm.DB, nodeID uint, encryptionConfig vo.EncryptionConfig, pluginConfig *vo.PluginConfig) error {
	model, err := r.mapper.ToModel(nodeID, encryptionConfig, pluginConfig)
	if err != nil {
		return fmt.Errorf("failed to map shadowsocks config to model: %w", err)
	}

	if err := tx.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create shadowsocks config: %w", err)
	}

	return nil
}

// UpdateInTx updates a ShadowsocksConfig record within a transaction
func (r *ShadowsocksConfigRepository) UpdateInTx(tx *gorm.DB, nodeID uint, encryptionConfig vo.EncryptionConfig, pluginConfig *vo.PluginConfig) error {
	var existing models.ShadowsocksConfigModel
	err := tx.Where("node_id = ?", nodeID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.CreateInTx(tx, nodeID, encryptionConfig, pluginConfig)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing shadowsocks config: %w", err)
	}

	model, err := r.mapper.ToModel(nodeID, encryptionConfig, pluginConfig)
	if err != nil {
		return fmt.Errorf("failed to map shadowsocks config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt // Preserve original creation time

	if err := tx.Save(model).Error; err != nil {
		return fmt.Errorf("failed to update shadowsocks config: %w", err)
	}

	return nil
}

// DeleteInTx deletes a ShadowsocksConfig record within a transaction
func (r *ShadowsocksConfigRepository) DeleteInTx(tx *gorm.DB, nodeID uint) error {
	return tx.Where("node_id = ?", nodeID).Delete(&models.ShadowsocksConfigModel{}).Error
}
