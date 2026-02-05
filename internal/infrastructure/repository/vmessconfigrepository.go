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

// VMessConfigRepository handles persistence operations for VMessConfig
type VMessConfigRepository struct {
	db     *gorm.DB
	mapper mappers.VMessConfigMapper
	logger logger.Interface
}

// NewVMessConfigRepository creates a new VMessConfigRepository
func NewVMessConfigRepository(db *gorm.DB, logger logger.Interface) *VMessConfigRepository {
	return &VMessConfigRepository{
		db:     db,
		mapper: mappers.NewVMessConfigMapper(),
		logger: logger,
	}
}

// Create creates a new VMessConfig record for a node
func (r *VMessConfigRepository) Create(ctx context.Context, nodeID uint, config *vo.VMessConfig) error {
	if config == nil {
		return nil
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map vmess config to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create vmess config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to create vmess config: %w", err)
	}

	r.logger.Infow("vmess config created", "node_id", nodeID, "id", model.ID)
	return nil
}

// GetByNodeID retrieves VMessConfig for a specific node
func (r *VMessConfigRepository) GetByNodeID(ctx context.Context, nodeID uint) (*vo.VMessConfig, error) {
	var model models.VMessConfigModel
	if err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Errorw("failed to get vmess config", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to get vmess config: %w", err)
	}

	return r.mapper.ToValueObject(&model, "")
}

// GetByNodeIDs retrieves VMessConfigs for multiple nodes
// Returns a map of nodeID -> VMessConfig
func (r *VMessConfigRepository) GetByNodeIDs(ctx context.Context, nodeIDs []uint) (map[uint]*vo.VMessConfig, error) {
	if len(nodeIDs) == 0 {
		return make(map[uint]*vo.VMessConfig), nil
	}

	var vmessModels []models.VMessConfigModel
	if err := r.db.WithContext(ctx).
		Select("node_id", "alter_id", "security", "transport_type", "host", "path",
			"service_name", "tls", "sni", "allow_insecure").
		Where("node_id IN ?", nodeIDs).
		Find(&vmessModels).Error; err != nil {
		r.logger.Errorw("failed to get vmess configs by node IDs", "node_ids", nodeIDs, "error", err)
		return nil, fmt.Errorf("failed to get vmess configs: %w", err)
	}

	result := make(map[uint]*vo.VMessConfig)
	for _, model := range vmessModels {
		config, err := r.mapper.ToValueObject(&model, "")
		if err != nil {
			r.logger.Warnw("failed to map vmess config", "node_id", model.NodeID, "error", err)
			continue
		}
		result[model.NodeID] = config
	}

	return result, nil
}

// Update updates the VMessConfig for a node
func (r *VMessConfigRepository) Update(ctx context.Context, nodeID uint, config *vo.VMessConfig) error {
	if config == nil {
		// If config is nil, delete the existing record
		return r.DeleteByNodeID(ctx, nodeID)
	}

	// Check if record exists
	var existing models.VMessConfigModel
	err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new record
		return r.Create(ctx, nodeID, config)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing vmess config: %w", err)
	}

	// Update existing record
	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map vmess config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt // Preserve original creation time

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		r.logger.Errorw("failed to update vmess config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to update vmess config: %w", err)
	}

	r.logger.Infow("vmess config updated", "node_id", nodeID)
	return nil
}

// DeleteByNodeID deletes the VMessConfig for a node
func (r *VMessConfigRepository) DeleteByNodeID(ctx context.Context, nodeID uint) error {
	result := r.db.WithContext(ctx).Where("node_id = ?", nodeID).Delete(&models.VMessConfigModel{})
	if result.Error != nil {
		r.logger.Errorw("failed to delete vmess config", "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to delete vmess config: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		r.logger.Infow("vmess config deleted", "node_id", nodeID)
	}
	return nil
}

// CreateInTx creates a VMessConfig record within a transaction
func (r *VMessConfigRepository) CreateInTx(tx *gorm.DB, nodeID uint, config *vo.VMessConfig) error {
	if config == nil {
		return nil
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map vmess config to model: %w", err)
	}

	if err := tx.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create vmess config: %w", err)
	}

	return nil
}

// UpdateInTx updates a VMessConfig record within a transaction
func (r *VMessConfigRepository) UpdateInTx(tx *gorm.DB, nodeID uint, config *vo.VMessConfig) error {
	if config == nil {
		return r.deleteInTx(tx, nodeID)
	}

	var existing models.VMessConfigModel
	err := tx.Where("node_id = ?", nodeID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.CreateInTx(tx, nodeID, config)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing vmess config: %w", err)
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map vmess config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt // Preserve original creation time

	if err := tx.Save(model).Error; err != nil {
		return fmt.Errorf("failed to update vmess config: %w", err)
	}

	return nil
}

// deleteInTx deletes a VMessConfig record within a transaction
func (r *VMessConfigRepository) deleteInTx(tx *gorm.DB, nodeID uint) error {
	return tx.Where("node_id = ?", nodeID).Delete(&models.VMessConfigModel{}).Error
}

// DeleteInTx deletes a VMessConfig record within a transaction (public method)
func (r *VMessConfigRepository) DeleteInTx(tx *gorm.DB, nodeID uint) error {
	return r.deleteInTx(tx, nodeID)
}
