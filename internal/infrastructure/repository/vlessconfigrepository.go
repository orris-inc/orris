package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// VLESSConfigRepository handles persistence operations for VLESSConfig
type VLESSConfigRepository struct {
	db     *gorm.DB
	mapper mappers.VLESSConfigMapper
	logger logger.Interface
}

// NewVLESSConfigRepository creates a new VLESSConfigRepository
func NewVLESSConfigRepository(db *gorm.DB, logger logger.Interface) *VLESSConfigRepository {
	return &VLESSConfigRepository{
		db:     db,
		mapper: mappers.NewVLESSConfigMapper(),
		logger: logger,
	}
}

// Create creates a new VLESSConfig record for a node
func (r *VLESSConfigRepository) Create(ctx context.Context, nodeID uint, config *vo.VLESSConfig) error {
	if config == nil {
		return nil
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map VLESS config to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create VLESS config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to create VLESS config: %w", err)
	}

	r.logger.Infow("VLESS config created", "node_id", nodeID, "id", model.ID)
	return nil
}

// GetByNodeID retrieves VLESSConfig for a specific node
func (r *VLESSConfigRepository) GetByNodeID(ctx context.Context, nodeID uint) (*vo.VLESSConfig, error) {
	var model models.VLESSConfigModel
	if err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Errorw("failed to get VLESS config", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to get VLESS config: %w", err)
	}

	return r.mapper.ToValueObject(&model)
}

// GetByNodeIDs retrieves VLESSConfigs for multiple nodes
// Returns a map of nodeID -> VLESSConfig
func (r *VLESSConfigRepository) GetByNodeIDs(ctx context.Context, nodeIDs []uint) (map[uint]*vo.VLESSConfig, error) {
	if len(nodeIDs) == 0 {
		return make(map[uint]*vo.VLESSConfig), nil
	}

	var vlessModels []models.VLESSConfigModel
	if err := r.db.WithContext(ctx).Where("node_id IN ?", nodeIDs).Find(&vlessModels).Error; err != nil {
		r.logger.Errorw("failed to get VLESS configs by node IDs", "node_ids", nodeIDs, "error", err)
		return nil, fmt.Errorf("failed to get VLESS configs: %w", err)
	}

	result := make(map[uint]*vo.VLESSConfig)
	for _, model := range vlessModels {
		config, err := r.mapper.ToValueObject(&model)
		if err != nil {
			r.logger.Warnw("failed to map VLESS config", "node_id", model.NodeID, "error", err)
			continue
		}
		result[model.NodeID] = config
	}

	return result, nil
}

// Update updates the VLESSConfig for a node
func (r *VLESSConfigRepository) Update(ctx context.Context, nodeID uint, config *vo.VLESSConfig) error {
	if config == nil {
		// If config is nil, delete the existing record
		return r.DeleteByNodeID(ctx, nodeID)
	}

	// Check if record exists
	var existing models.VLESSConfigModel
	err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new record
		return r.Create(ctx, nodeID, config)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing VLESS config: %w", err)
	}

	// Update existing record
	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map VLESS config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt // Preserve original creation time

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		r.logger.Errorw("failed to update VLESS config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to update VLESS config: %w", err)
	}

	r.logger.Infow("VLESS config updated", "node_id", nodeID)
	return nil
}

// DeleteByNodeID deletes the VLESSConfig for a node
func (r *VLESSConfigRepository) DeleteByNodeID(ctx context.Context, nodeID uint) error {
	result := r.db.WithContext(ctx).Where("node_id = ?", nodeID).Delete(&models.VLESSConfigModel{})
	if result.Error != nil {
		r.logger.Errorw("failed to delete VLESS config", "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to delete VLESS config: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		r.logger.Infow("VLESS config deleted", "node_id", nodeID)
	}
	return nil
}

// CreateInTx creates a VLESSConfig record within a transaction
func (r *VLESSConfigRepository) CreateInTx(tx *gorm.DB, nodeID uint, config *vo.VLESSConfig) error {
	if config == nil {
		return nil
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map VLESS config to model: %w", err)
	}

	if err := tx.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create VLESS config: %w", err)
	}

	return nil
}

// UpdateInTx updates a VLESSConfig record within a transaction
func (r *VLESSConfigRepository) UpdateInTx(tx *gorm.DB, nodeID uint, config *vo.VLESSConfig) error {
	if config == nil {
		return r.deleteInTx(tx, nodeID)
	}

	var existing models.VLESSConfigModel
	err := tx.Where("node_id = ?", nodeID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.CreateInTx(tx, nodeID, config)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing VLESS config: %w", err)
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map VLESS config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt // Preserve original creation time

	if err := tx.Save(model).Error; err != nil {
		return fmt.Errorf("failed to update VLESS config: %w", err)
	}

	return nil
}

// deleteInTx deletes a VLESSConfig record within a transaction
func (r *VLESSConfigRepository) deleteInTx(tx *gorm.DB, nodeID uint) error {
	return tx.Where("node_id = ?", nodeID).Delete(&models.VLESSConfigModel{}).Error
}

// DeleteInTx deletes a VLESSConfig record within a transaction (public method)
func (r *VLESSConfigRepository) DeleteInTx(tx *gorm.DB, nodeID uint) error {
	return r.deleteInTx(tx, nodeID)
}
