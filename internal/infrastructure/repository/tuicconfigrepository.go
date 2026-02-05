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

// TUICConfigRepository handles persistence operations for TUICConfig
type TUICConfigRepository struct {
	db     *gorm.DB
	mapper mappers.TUICConfigMapper
	logger logger.Interface
}

// NewTUICConfigRepository creates a new TUICConfigRepository
func NewTUICConfigRepository(db *gorm.DB, logger logger.Interface) *TUICConfigRepository {
	return &TUICConfigRepository{
		db:     db,
		mapper: mappers.NewTUICConfigMapper(),
		logger: logger,
	}
}

// Create creates a new TUICConfig record for a node
func (r *TUICConfigRepository) Create(ctx context.Context, nodeID uint, config *vo.TUICConfig) error {
	if config == nil {
		return nil
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map TUIC config to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create TUIC config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to create TUIC config: %w", err)
	}

	r.logger.Infow("TUIC config created", "node_id", nodeID, "id", model.ID)
	return nil
}

// GetByNodeID retrieves TUICConfig for a specific node
func (r *TUICConfigRepository) GetByNodeID(ctx context.Context, nodeID uint) (*vo.TUICConfig, error) {
	var model models.TUICConfigModel
	if err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Errorw("failed to get TUIC config", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to get TUIC config: %w", err)
	}

	return r.mapper.ToValueObject(&model, "", "")
}

// GetByNodeIDs retrieves TUICConfigs for multiple nodes
// Returns a map of nodeID -> TUICConfig
func (r *TUICConfigRepository) GetByNodeIDs(ctx context.Context, nodeIDs []uint) (map[uint]*vo.TUICConfig, error) {
	if len(nodeIDs) == 0 {
		return make(map[uint]*vo.TUICConfig), nil
	}

	var tuicModels []models.TUICConfigModel
	if err := r.db.WithContext(ctx).
		Select("node_id", "congestion_control", "udp_relay_mode", "alpn", "sni", "allow_insecure", "disable_sni").
		Where("node_id IN ?", nodeIDs).
		Find(&tuicModels).Error; err != nil {
		r.logger.Errorw("failed to get TUIC configs by node IDs", "node_ids", nodeIDs, "error", err)
		return nil, fmt.Errorf("failed to get TUIC configs: %w", err)
	}

	result := make(map[uint]*vo.TUICConfig)
	for _, model := range tuicModels {
		config, err := r.mapper.ToValueObject(&model, "", "")
		if err != nil {
			r.logger.Warnw("failed to map TUIC config", "node_id", model.NodeID, "error", err)
			continue
		}
		result[model.NodeID] = config
	}

	return result, nil
}

// Update updates the TUICConfig for a node
func (r *TUICConfigRepository) Update(ctx context.Context, nodeID uint, config *vo.TUICConfig) error {
	if config == nil {
		// If config is nil, delete the existing record
		return r.DeleteByNodeID(ctx, nodeID)
	}

	// Check if record exists
	var existing models.TUICConfigModel
	err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new record
		return r.Create(ctx, nodeID, config)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing TUIC config: %w", err)
	}

	// Update existing record
	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map TUIC config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt // Preserve original creation time

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		r.logger.Errorw("failed to update TUIC config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to update TUIC config: %w", err)
	}

	r.logger.Infow("TUIC config updated", "node_id", nodeID)
	return nil
}

// DeleteByNodeID deletes the TUICConfig for a node
func (r *TUICConfigRepository) DeleteByNodeID(ctx context.Context, nodeID uint) error {
	result := r.db.WithContext(ctx).Where("node_id = ?", nodeID).Delete(&models.TUICConfigModel{})
	if result.Error != nil {
		r.logger.Errorw("failed to delete TUIC config", "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to delete TUIC config: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		r.logger.Infow("TUIC config deleted", "node_id", nodeID)
	}
	return nil
}

// CreateInTx creates a TUICConfig record within a transaction
func (r *TUICConfigRepository) CreateInTx(tx *gorm.DB, nodeID uint, config *vo.TUICConfig) error {
	if config == nil {
		return nil
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map TUIC config to model: %w", err)
	}

	if err := tx.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create TUIC config: %w", err)
	}

	return nil
}

// UpdateInTx updates a TUICConfig record within a transaction
func (r *TUICConfigRepository) UpdateInTx(tx *gorm.DB, nodeID uint, config *vo.TUICConfig) error {
	if config == nil {
		return r.deleteInTx(tx, nodeID)
	}

	var existing models.TUICConfigModel
	err := tx.Where("node_id = ?", nodeID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.CreateInTx(tx, nodeID, config)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing TUIC config: %w", err)
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map TUIC config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt // Preserve original creation time

	if err := tx.Save(model).Error; err != nil {
		return fmt.Errorf("failed to update TUIC config: %w", err)
	}

	return nil
}

// deleteInTx deletes a TUICConfig record within a transaction
func (r *TUICConfigRepository) deleteInTx(tx *gorm.DB, nodeID uint) error {
	return tx.Where("node_id = ?", nodeID).Delete(&models.TUICConfigModel{}).Error
}

// DeleteInTx deletes a TUICConfig record within a transaction (public method)
func (r *TUICConfigRepository) DeleteInTx(tx *gorm.DB, nodeID uint) error {
	return r.deleteInTx(tx, nodeID)
}
