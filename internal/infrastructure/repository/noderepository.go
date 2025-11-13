package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"orris/internal/domain/node"
	"orris/internal/infrastructure/persistence/mappers"
	"orris/internal/infrastructure/persistence/models"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// NodeRepositoryImpl implements the node.NodeRepository interface
type NodeRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.NodeMapper
	logger logger.Interface
}

// NewNodeRepository creates a new node repository instance
func NewNodeRepository(db *gorm.DB, logger logger.Interface) node.NodeRepository {
	return &NodeRepositoryImpl{
		db:     db,
		mapper: mappers.NewNodeMapper(),
		logger: logger,
	}
}

// Create creates a new node in the database
func (r *NodeRepositoryImpl) Create(ctx context.Context, nodeEntity *node.Node) error {
	model, err := r.mapper.ToModel(nodeEntity)
	if err != nil {
		r.logger.Errorw("failed to map node entity to model", "error", err)
		return fmt.Errorf("failed to map node entity: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "duplicate key") {
			if strings.Contains(err.Error(), "name") {
				return errors.NewConflictError("node with this name already exists")
			}
			if strings.Contains(err.Error(), "token_hash") {
				return errors.NewConflictError("node with this token already exists")
			}
			return errors.NewConflictError("node already exists")
		}
		r.logger.Errorw("failed to create node in database", "error", err)
		return fmt.Errorf("failed to create node: %w", err)
	}

	if err := nodeEntity.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set node ID", "error", err)
		return fmt.Errorf("failed to set node ID: %w", err)
	}

	r.logger.Infow("node created successfully", "id", model.ID, "name", model.Name)
	return nil
}

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

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map node model to entity", "id", id, "error", err)
		return nil, fmt.Errorf("failed to map node: %w", err)
	}

	return entity, nil
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

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map node model to entity", "token_hash", tokenHash, "error", err)
		return nil, fmt.Errorf("failed to map node: %w", err)
	}

	return entity, nil
}

// Update updates an existing node with optimistic locking
func (r *NodeRepositoryImpl) Update(ctx context.Context, nodeEntity *node.Node) error {
	model, err := r.mapper.ToModel(nodeEntity)
	if err != nil {
		r.logger.Errorw("failed to map node entity to model", "error", err)
		return fmt.Errorf("failed to map node entity: %w", err)
	}

	// Use optimistic locking by checking version
	// The domain entity has already incremented the version, so we need to check against the previous version
	previousVersion := model.Version - 1
	result := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("id = ? AND version = ?", model.ID, previousVersion).
		Updates(map[string]interface{}{
			"name":               model.Name,
			"server_address":     model.ServerAddress,
			"server_port":        model.ServerPort,
			"encryption_method":  model.EncryptionMethod,
			"plugin":             model.Plugin,
			"plugin_opts":        model.PluginOpts,
			"protocol":           model.Protocol,
			"status":             model.Status,
			"region":             model.Region,
			"tags":               model.Tags,
			"custom_fields":      model.CustomFields,
			"sort_order":         model.SortOrder,
			"maintenance_reason": model.MaintenanceReason,
			"token_hash":         model.TokenHash,
			"updated_at":         model.UpdatedAt,
			"version":            model.Version,
		})

	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "Duplicate entry") || strings.Contains(result.Error.Error(), "duplicate key") {
			if strings.Contains(result.Error.Error(), "name") {
				return errors.NewConflictError("node with this name already exists")
			}
			return errors.NewConflictError("node already exists")
		}
		r.logger.Errorw("failed to update node", "id", model.ID, "error", result.Error)
		return fmt.Errorf("failed to update node: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewConflictError("node has been modified by another transaction or not found")
	}

	r.logger.Infow("node updated successfully", "id", model.ID, "name", model.Name)
	return nil
}

// Delete soft deletes a node
func (r *NodeRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.NodeModel{}, id)
	if result.Error != nil {
		r.logger.Errorw("failed to delete node", "id", id, "error", result.Error)
		return fmt.Errorf("failed to delete node: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("node not found")
	}

	r.logger.Infow("node deleted successfully", "id", id)
	return nil
}

// List retrieves a paginated list of nodes with filtering
func (r *NodeRepositoryImpl) List(ctx context.Context, filter node.NodeFilter) ([]*node.Node, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.NodeModel{})

	// Apply filters
	if filter.Name != nil && *filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Status != nil && *filter.Status != "" {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Tag != nil && *filter.Tag != "" {
		// Search in JSON tags array
		query = query.Where("JSON_CONTAINS(tags, ?)", fmt.Sprintf(`"%s"`, *filter.Tag))
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count nodes", "error", err)
		return nil, 0, fmt.Errorf("failed to count nodes: %w", err)
	}

	// Apply sorting
	orderClause := filter.SortFilter.OrderClause()
	if orderClause != "" {
		query = query.Order(orderClause)
	} else {
		query = query.Order("sort_order ASC, created_at DESC")
	}

	// Apply pagination
	offset := filter.PageFilter.Offset()
	limit := filter.PageFilter.Limit()
	query = query.Offset(offset).Limit(limit)

	// Execute query
	var nodeModels []*models.NodeModel
	if err := query.Find(&nodeModels).Error; err != nil {
		r.logger.Errorw("failed to list nodes", "error", err)
		return nil, 0, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(nodeModels)
	if err != nil {
		r.logger.Errorw("failed to map node models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map nodes: %w", err)
	}

	return entities, total, nil
}

// ExistsByName checks if a node with the given name exists
func (r *NodeRepositoryImpl) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by name", "name", name, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByAddress checks if a node with the given address and port exists
func (r *NodeRepositoryImpl) ExistsByAddress(ctx context.Context, address string, port int) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("server_address = ? AND server_port = ?", address, port).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by address", "address", address, "port", port, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// IncrementTraffic atomically increments the traffic_used field
func (r *NodeRepositoryImpl) IncrementTraffic(ctx context.Context, nodeID uint, amount uint64) error {
	if amount == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("id = ?", nodeID).
		UpdateColumn("traffic_used", gorm.Expr("traffic_used + ?", amount))

	if result.Error != nil {
		r.logger.Errorw("failed to increment traffic", "node_id", nodeID, "amount", amount, "error", result.Error)
		return fmt.Errorf("failed to increment traffic: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("node not found")
	}

	r.logger.Debugw("traffic incremented successfully", "node_id", nodeID, "amount", amount)
	return nil
}
