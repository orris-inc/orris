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

// Hysteria2ConfigRepository handles persistence operations for Hysteria2Config
type Hysteria2ConfigRepository struct {
	db     *gorm.DB
	mapper mappers.Hysteria2ConfigMapper
	logger logger.Interface
}

// NewHysteria2ConfigRepository creates a new Hysteria2ConfigRepository
func NewHysteria2ConfigRepository(db *gorm.DB, logger logger.Interface) *Hysteria2ConfigRepository {
	return &Hysteria2ConfigRepository{
		db:     db,
		mapper: mappers.NewHysteria2ConfigMapper(),
		logger: logger,
	}
}

// Create creates a new Hysteria2Config record for a node
func (r *Hysteria2ConfigRepository) Create(ctx context.Context, nodeID uint, config *vo.Hysteria2Config) error {
	if config == nil {
		return nil
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map hysteria2 config to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create hysteria2 config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to create hysteria2 config: %w", err)
	}

	r.logger.Infow("hysteria2 config created", "node_id", nodeID, "id", model.ID)
	return nil
}

// GetByNodeID retrieves Hysteria2Config for a specific node
func (r *Hysteria2ConfigRepository) GetByNodeID(ctx context.Context, nodeID uint) (*vo.Hysteria2Config, error) {
	var model models.Hysteria2ConfigModel
	if err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Errorw("failed to get hysteria2 config", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to get hysteria2 config: %w", err)
	}

	return r.mapper.ToValueObject(&model, "placeholder")
}

// GetByNodeIDs retrieves Hysteria2Configs for multiple nodes
// Returns a map of nodeID -> Hysteria2Config
func (r *Hysteria2ConfigRepository) GetByNodeIDs(ctx context.Context, nodeIDs []uint) (map[uint]*vo.Hysteria2Config, error) {
	if len(nodeIDs) == 0 {
		return make(map[uint]*vo.Hysteria2Config), nil
	}

	var hysteria2Models []models.Hysteria2ConfigModel
	if err := r.db.WithContext(ctx).Where("node_id IN ?", nodeIDs).Find(&hysteria2Models).Error; err != nil {
		r.logger.Errorw("failed to get hysteria2 configs by node IDs", "node_ids", nodeIDs, "error", err)
		return nil, fmt.Errorf("failed to get hysteria2 configs: %w", err)
	}

	result := make(map[uint]*vo.Hysteria2Config)
	for _, model := range hysteria2Models {
		config, err := r.mapper.ToValueObject(&model, "placeholder")
		if err != nil {
			r.logger.Warnw("failed to map hysteria2 config", "node_id", model.NodeID, "error", err)
			continue
		}
		result[model.NodeID] = config
	}

	return result, nil
}

// Update updates the Hysteria2Config for a node
func (r *Hysteria2ConfigRepository) Update(ctx context.Context, nodeID uint, config *vo.Hysteria2Config) error {
	if config == nil {
		// If config is nil, delete the existing record
		return r.DeleteByNodeID(ctx, nodeID)
	}

	// Check if record exists
	var existing models.Hysteria2ConfigModel
	err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new record
		return r.Create(ctx, nodeID, config)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing hysteria2 config: %w", err)
	}

	// Update existing record
	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map hysteria2 config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt // Preserve original creation time

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		r.logger.Errorw("failed to update hysteria2 config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to update hysteria2 config: %w", err)
	}

	r.logger.Infow("hysteria2 config updated", "node_id", nodeID)
	return nil
}

// DeleteByNodeID deletes the Hysteria2Config for a node
func (r *Hysteria2ConfigRepository) DeleteByNodeID(ctx context.Context, nodeID uint) error {
	result := r.db.WithContext(ctx).Where("node_id = ?", nodeID).Delete(&models.Hysteria2ConfigModel{})
	if result.Error != nil {
		r.logger.Errorw("failed to delete hysteria2 config", "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to delete hysteria2 config: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		r.logger.Infow("hysteria2 config deleted", "node_id", nodeID)
	}
	return nil
}

// CreateInTx creates a Hysteria2Config record within a transaction
func (r *Hysteria2ConfigRepository) CreateInTx(tx *gorm.DB, nodeID uint, config *vo.Hysteria2Config) error {
	if config == nil {
		return nil
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map hysteria2 config to model: %w", err)
	}

	if err := tx.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create hysteria2 config: %w", err)
	}

	return nil
}

// UpdateInTx updates a Hysteria2Config record within a transaction
func (r *Hysteria2ConfigRepository) UpdateInTx(tx *gorm.DB, nodeID uint, config *vo.Hysteria2Config) error {
	if config == nil {
		return r.deleteInTx(tx, nodeID)
	}

	var existing models.Hysteria2ConfigModel
	err := tx.Where("node_id = ?", nodeID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.CreateInTx(tx, nodeID, config)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing hysteria2 config: %w", err)
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map hysteria2 config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt // Preserve original creation time

	if err := tx.Save(model).Error; err != nil {
		return fmt.Errorf("failed to update hysteria2 config: %w", err)
	}

	return nil
}

// deleteInTx deletes a Hysteria2Config record within a transaction
func (r *Hysteria2ConfigRepository) deleteInTx(tx *gorm.DB, nodeID uint) error {
	return tx.Where("node_id = ?", nodeID).Delete(&models.Hysteria2ConfigModel{}).Error
}

// DeleteInTx deletes a Hysteria2Config record within a transaction (public method)
func (r *Hysteria2ConfigRepository) DeleteInTx(tx *gorm.DB, nodeID uint) error {
	return r.deleteInTx(tx, nodeID)
}
