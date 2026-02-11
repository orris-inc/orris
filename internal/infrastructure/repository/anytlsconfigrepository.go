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

// AnyTLSConfigRepository handles persistence operations for AnyTLSConfig
type AnyTLSConfigRepository struct {
	db     *gorm.DB
	mapper mappers.AnyTLSConfigMapper
	logger logger.Interface
}

// NewAnyTLSConfigRepository creates a new AnyTLSConfigRepository
func NewAnyTLSConfigRepository(db *gorm.DB, logger logger.Interface) *AnyTLSConfigRepository {
	return &AnyTLSConfigRepository{
		db:     db,
		mapper: mappers.NewAnyTLSConfigMapper(),
		logger: logger,
	}
}

// Create creates a new AnyTLSConfig record for a node
func (r *AnyTLSConfigRepository) Create(ctx context.Context, nodeID uint, config *vo.AnyTLSConfig) error {
	if config == nil {
		return nil
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map anytls config to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Errorw("failed to create anytls config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to create anytls config: %w", err)
	}

	r.logger.Infow("anytls config created", "node_id", nodeID, "id", model.ID)
	return nil
}

// GetByNodeID retrieves AnyTLSConfig for a specific node
func (r *AnyTLSConfigRepository) GetByNodeID(ctx context.Context, nodeID uint) (*vo.AnyTLSConfig, error) {
	var model models.AnyTLSConfigModel
	if err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get anytls config", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to get anytls config: %w", err)
	}

	return r.mapper.ToValueObject(&model, mappers.PlaceholderPassword)
}

// GetByNodeIDs retrieves AnyTLSConfigs for multiple nodes
// Returns a map of nodeID -> AnyTLSConfig
func (r *AnyTLSConfigRepository) GetByNodeIDs(ctx context.Context, nodeIDs []uint) (map[uint]*vo.AnyTLSConfig, error) {
	if len(nodeIDs) == 0 {
		return make(map[uint]*vo.AnyTLSConfig), nil
	}

	var anytlsModels []models.AnyTLSConfigModel
	if err := r.db.WithContext(ctx).
		Select("node_id", "sni", "allow_insecure", "fingerprint", "idle_session_check_interval", "idle_session_timeout", "min_idle_session").
		Where("node_id IN ?", nodeIDs).
		Find(&anytlsModels).Error; err != nil {
		r.logger.Errorw("failed to get anytls configs by node IDs", "node_ids", nodeIDs, "error", err)
		return nil, fmt.Errorf("failed to get anytls configs: %w", err)
	}

	result := make(map[uint]*vo.AnyTLSConfig)
	for _, model := range anytlsModels {
		config, err := r.mapper.ToValueObject(&model, mappers.PlaceholderPassword)
		if err != nil {
			r.logger.Warnw("failed to map anytls config", "node_id", model.NodeID, "error", err)
			continue
		}
		result[model.NodeID] = config
	}

	return result, nil
}

// Update updates the AnyTLSConfig for a node
func (r *AnyTLSConfigRepository) Update(ctx context.Context, nodeID uint, config *vo.AnyTLSConfig) error {
	if config == nil {
		return r.DeleteByNodeID(ctx, nodeID)
	}

	var existing models.AnyTLSConfigModel
	err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.Create(ctx, nodeID, config)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing anytls config: %w", err)
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map anytls config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		r.logger.Errorw("failed to update anytls config", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to update anytls config: %w", err)
	}

	r.logger.Infow("anytls config updated", "node_id", nodeID)
	return nil
}

// DeleteByNodeID deletes the AnyTLSConfig for a node
func (r *AnyTLSConfigRepository) DeleteByNodeID(ctx context.Context, nodeID uint) error {
	result := r.db.WithContext(ctx).Where("node_id = ?", nodeID).Delete(&models.AnyTLSConfigModel{})
	if result.Error != nil {
		r.logger.Errorw("failed to delete anytls config", "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to delete anytls config: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		r.logger.Infow("anytls config deleted", "node_id", nodeID)
	}
	return nil
}

// CreateInTx creates an AnyTLSConfig record within a transaction
func (r *AnyTLSConfigRepository) CreateInTx(tx *gorm.DB, nodeID uint, config *vo.AnyTLSConfig) error {
	if config == nil {
		return nil
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map anytls config to model: %w", err)
	}

	if err := tx.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create anytls config: %w", err)
	}

	return nil
}

// UpdateInTx updates an AnyTLSConfig record within a transaction
func (r *AnyTLSConfigRepository) UpdateInTx(tx *gorm.DB, nodeID uint, config *vo.AnyTLSConfig) error {
	if config == nil {
		return r.DeleteInTx(tx, nodeID)
	}

	var existing models.AnyTLSConfigModel
	err := tx.Where("node_id = ?", nodeID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.CreateInTx(tx, nodeID, config)
	}
	if err != nil {
		return fmt.Errorf("failed to check existing anytls config: %w", err)
	}

	model, err := r.mapper.ToModel(nodeID, config)
	if err != nil {
		return fmt.Errorf("failed to map anytls config to model: %w", err)
	}
	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt

	if err := tx.Save(model).Error; err != nil {
		return fmt.Errorf("failed to update anytls config: %w", err)
	}

	return nil
}

// DeleteInTx deletes an AnyTLSConfig record within a transaction
func (r *AnyTLSConfigRepository) DeleteInTx(tx *gorm.DB, nodeID uint) error {
	return tx.Where("node_id = ?", nodeID).Delete(&models.AnyTLSConfigModel{}).Error
}
