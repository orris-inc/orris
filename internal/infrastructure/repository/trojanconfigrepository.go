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

// TrojanConfigRepository handles persistence operations for TrojanConfig
type TrojanConfigRepository struct {
	db     *gorm.DB
	mapper mappers.TrojanConfigMapper
	logger logger.Interface
}

// NewTrojanConfigRepository creates a new TrojanConfigRepository
func NewTrojanConfigRepository(db *gorm.DB, logger logger.Interface) *TrojanConfigRepository {
	return &TrojanConfigRepository{
		db:     db,
		mapper: mappers.NewTrojanConfigMapper(),
		logger: logger,
	}
}

// Create creates a new TrojanConfig record for a node
func (r *TrojanConfigRepository) Create(ctx context.Context, nodeID uint, config *vo.TrojanConfig) error {
	if config == nil {
		return nil
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map trojan config to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create trojan config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to create trojan config: %w", err)
	}

	r.logger.Infow("trojan config created", "node_id", nodeID, "id", model.ID)
	return nil
}

// GetByNodeID retrieves TrojanConfig for a specific node
func (r *TrojanConfigRepository) GetByNodeID(ctx context.Context, nodeID uint) (*vo.TrojanConfig, error) {
	var model models.TrojanConfigModel
	if err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get trojan config", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to get trojan config: %w", err)
	}

	return r.mapper.ToValueObject(&model, "placeholder")
}

// GetByNodeIDs retrieves TrojanConfigs for multiple nodes
// Returns a map of nodeID -> TrojanConfig
func (r *TrojanConfigRepository) GetByNodeIDs(ctx context.Context, nodeIDs []uint) (map[uint]*vo.TrojanConfig, error) {
	if len(nodeIDs) == 0 {
		return make(map[uint]*vo.TrojanConfig), nil
	}

	var trojanModels []models.TrojanConfigModel
	if err := r.db.WithContext(ctx).
		Select("node_id", "transport_protocol", "host", "path", "sni", "allow_insecure").
		Where("node_id IN ?", nodeIDs).
		Find(&trojanModels).Error; err != nil {
		r.logger.Errorw("failed to get trojan configs by node IDs", "node_ids", nodeIDs, "error", err)
		return nil, fmt.Errorf("failed to get trojan configs: %w", err)
	}

	result := make(map[uint]*vo.TrojanConfig)
	for _, model := range trojanModels {
		config, err := r.mapper.ToValueObject(&model, "placeholder")
		if err != nil {
			r.logger.Warnw("failed to map trojan config", "node_id", model.NodeID, "error", err)
			continue
		}
		result[model.NodeID] = config
	}

	return result, nil
}

// Update updates the TrojanConfig for a node
func (r *TrojanConfigRepository) Update(ctx context.Context, nodeID uint, config *vo.TrojanConfig) error {
	if config == nil {
		// If config is nil, delete the existing record
		return r.DeleteByNodeID(ctx, nodeID)
	}

	// Check if record exists
	var existing models.TrojanConfigModel
	err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		// Create new record
		return r.Create(ctx, nodeID, config)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing trojan config: %w", err)
	}

	// Update existing record
	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map trojan config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt // Preserve original creation time

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		r.logger.Errorw("failed to update trojan config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to update trojan config: %w", err)
	}

	r.logger.Infow("trojan config updated", "node_id", nodeID)
	return nil
}

// DeleteByNodeID deletes the TrojanConfig for a node
func (r *TrojanConfigRepository) DeleteByNodeID(ctx context.Context, nodeID uint) error {
	result := r.db.WithContext(ctx).Where("node_id = ?", nodeID).Delete(&models.TrojanConfigModel{})
	if result.Error != nil {
		r.logger.Errorw("failed to delete trojan config", "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to delete trojan config: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		r.logger.Infow("trojan config deleted", "node_id", nodeID)
	}
	return nil
}

// CreateInTx creates a TrojanConfig record within a transaction
func (r *TrojanConfigRepository) CreateInTx(tx *gorm.DB, nodeID uint, config *vo.TrojanConfig) error {
	if config == nil {
		return nil
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map trojan config to model: %w", err)
	}

	if err := tx.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create trojan config: %w", err)
	}

	return nil
}

// UpdateInTx updates a TrojanConfig record within a transaction
func (r *TrojanConfigRepository) UpdateInTx(tx *gorm.DB, nodeID uint, config *vo.TrojanConfig) error {
	if config == nil {
		return r.deleteInTx(tx, nodeID)
	}

	var existing models.TrojanConfigModel
	err := tx.Where("node_id = ?", nodeID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.CreateInTx(tx, nodeID, config)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing trojan config: %w", err)
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map trojan config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt // Preserve original creation time

	if err := tx.Save(model).Error; err != nil {
		return fmt.Errorf("failed to update trojan config: %w", err)
	}

	return nil
}

// deleteInTx deletes a TrojanConfig record within a transaction
func (r *TrojanConfigRepository) deleteInTx(tx *gorm.DB, nodeID uint) error {
	return tx.Where("node_id = ?", nodeID).Delete(&models.TrojanConfigModel{}).Error
}

// DeleteInTx deletes a TrojanConfig record within a transaction (public method)
func (r *TrojanConfigRepository) DeleteInTx(tx *gorm.DB, nodeID uint) error {
	return r.deleteInTx(tx, nodeID)
}
