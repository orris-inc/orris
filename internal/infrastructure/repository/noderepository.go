package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/value_objects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// NodeRepositoryImpl implements the node.NodeRepository interface
type NodeRepositoryImpl struct {
	db                    *gorm.DB
	mapper                mappers.NodeMapper
	trojanConfigRepo      *TrojanConfigRepository
	shadowsocksConfigRepo *ShadowsocksConfigRepository
	logger                logger.Interface
}

// NewNodeRepository creates a new node repository instance
func NewNodeRepository(db *gorm.DB, logger logger.Interface) node.NodeRepository {
	return &NodeRepositoryImpl{
		db:                    db,
		mapper:                mappers.NewNodeMapper(),
		trojanConfigRepo:      NewTrojanConfigRepository(db, logger),
		shadowsocksConfigRepo: NewShadowsocksConfigRepository(db, logger),
		logger:                logger,
	}
}

// Create creates a new node in the database
// Uses transaction to ensure node and protocol-specific configs are created atomically
func (r *NodeRepositoryImpl) Create(ctx context.Context, nodeEntity *node.Node) error {
	model, err := r.mapper.ToModel(nodeEntity)
	if err != nil {
		r.logger.Errorw("failed to map node entity to model", "error", err)
		return fmt.Errorf("failed to map node entity: %w", err)
	}

	// Use transaction to create node and protocol config atomically
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create node
		if err := tx.Create(model).Error; err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "duplicate key") {
				if strings.Contains(err.Error(), "name") {
					return errors.NewConflictError("node with this name already exists")
				}
				if strings.Contains(err.Error(), "token_hash") {
					return errors.NewConflictError("node with this token already exists")
				}
				return errors.NewConflictError("node already exists")
			}
			return fmt.Errorf("failed to create node: %w", err)
		}

		// Create protocol-specific config based on protocol type
		switch nodeEntity.Protocol() {
		case vo.ProtocolShadowsocks:
			if err := r.shadowsocksConfigRepo.CreateInTx(tx, model.ID, nodeEntity.EncryptionConfig(), nodeEntity.PluginConfig()); err != nil {
				return fmt.Errorf("failed to create shadowsocks config: %w", err)
			}
		case vo.ProtocolTrojan:
			if nodeEntity.TrojanConfig() != nil {
				if err := r.trojanConfigRepo.CreateInTx(tx, model.ID, nodeEntity.TrojanConfig()); err != nil {
					return fmt.Errorf("failed to create trojan config: %w", err)
				}
			}
		}

		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to create node in database", "error", err)
		return err
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

	// Load protocol-specific config
	var trojanConfig *vo.TrojanConfig
	var encryptionConfig vo.EncryptionConfig
	var pluginConfig *vo.PluginConfig

	switch model.Protocol {
	case "shadowsocks":
		var err error
		encryptionConfig, pluginConfig, err = r.shadowsocksConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get shadowsocks config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get shadowsocks config: %w", err)
		}
	case "trojan":
		var err error
		trojanConfig, err = r.trojanConfigRepo.GetByNodeID(ctx, id)
		if err != nil {
			r.logger.Errorw("failed to get trojan config", "node_id", id, "error", err)
			return nil, fmt.Errorf("failed to get trojan config: %w", err)
		}
	}

	entity, err := r.mapper.ToEntity(&model, encryptionConfig, pluginConfig, trojanConfig)
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

	// Load protocol-specific config
	var trojanConfig *vo.TrojanConfig
	var encryptionConfig vo.EncryptionConfig
	var pluginConfig *vo.PluginConfig

	switch model.Protocol {
	case "shadowsocks":
		var err error
		encryptionConfig, pluginConfig, err = r.shadowsocksConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get shadowsocks config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get shadowsocks config: %w", err)
		}
	case "trojan":
		var err error
		trojanConfig, err = r.trojanConfigRepo.GetByNodeID(ctx, model.ID)
		if err != nil {
			r.logger.Errorw("failed to get trojan config", "node_id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to get trojan config: %w", err)
		}
	}

	entity, err := r.mapper.ToEntity(&model, encryptionConfig, pluginConfig, trojanConfig)
	if err != nil {
		r.logger.Errorw("failed to map node model to entity", "token_hash", tokenHash, "error", err)
		return nil, fmt.Errorf("failed to map node: %w", err)
	}

	return entity, nil
}

// Update updates an existing node with optimistic locking
// Uses transaction to ensure node and protocol-specific configs are updated atomically
func (r *NodeRepositoryImpl) Update(ctx context.Context, nodeEntity *node.Node) error {
	model, err := r.mapper.ToModel(nodeEntity)
	if err != nil {
		r.logger.Errorw("failed to map node entity to model", "error", err)
		return fmt.Errorf("failed to map node entity: %w", err)
	}

	// Use transaction to update node and protocol config atomically
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.NodeModel{}).
			Where("id = ?", model.ID).
			Updates(map[string]interface{}{
				"name":               model.Name,
				"server_address":     model.ServerAddress,
				"agent_port":         model.AgentPort,
				"subscription_port":  model.SubscriptionPort,
				"protocol":           model.Protocol,
				"status":             model.Status,
				"region":             model.Region,
				"tags":               model.Tags,
				"sort_order":         model.SortOrder,
				"maintenance_reason": model.MaintenanceReason,
				"token_hash":         model.TokenHash,
				"updated_at":         model.UpdatedAt,
			})

		if result.Error != nil {
			if strings.Contains(result.Error.Error(), "Duplicate entry") || strings.Contains(result.Error.Error(), "duplicate key") {
				if strings.Contains(result.Error.Error(), "name") {
					return errors.NewConflictError("node with this name already exists")
				}
				return errors.NewConflictError("node already exists")
			}
			return fmt.Errorf("failed to update node: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return errors.NewNotFoundError("node not found", fmt.Sprintf("id=%d", model.ID))
		}

		// Update protocol-specific config based on protocol type
		switch nodeEntity.Protocol() {
		case vo.ProtocolShadowsocks:
			if err := r.shadowsocksConfigRepo.UpdateInTx(tx, model.ID, nodeEntity.EncryptionConfig(), nodeEntity.PluginConfig()); err != nil {
				return fmt.Errorf("failed to update shadowsocks config: %w", err)
			}
			// Delete trojan config if it exists (protocol changed)
			if err := r.trojanConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete trojan config: %w", err)
			}
		case vo.ProtocolTrojan:
			if err := r.trojanConfigRepo.UpdateInTx(tx, model.ID, nodeEntity.TrojanConfig()); err != nil {
				return fmt.Errorf("failed to update trojan config: %w", err)
			}
			// Delete shadowsocks config if it exists (protocol changed)
			if err := r.shadowsocksConfigRepo.DeleteInTx(tx, model.ID); err != nil {
				return fmt.Errorf("failed to delete shadowsocks config: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to update node", "id", model.ID, "error", err)
		return err
	}

	r.logger.Infow("node updated successfully", "id", model.ID, "name", model.Name)
	return nil
}

// Delete soft deletes a node and its associated protocol configs
func (r *NodeRepositoryImpl) Delete(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete protocol configs first
		if err := r.shadowsocksConfigRepo.DeleteInTx(tx, id); err != nil {
			return fmt.Errorf("failed to delete shadowsocks config: %w", err)
		}
		if err := r.trojanConfigRepo.DeleteInTx(tx, id); err != nil {
			return fmt.Errorf("failed to delete trojan config: %w", err)
		}

		// Delete node
		result := tx.Delete(&models.NodeModel{}, id)
		if result.Error != nil {
			return fmt.Errorf("failed to delete node: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return errors.NewNotFoundError("node not found")
		}

		return nil
	})

	if err != nil {
		r.logger.Errorw("failed to delete node", "id", id, "error", err)
		return err
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

	// Collect node IDs by protocol
	var ssNodeIDs, trojanNodeIDs []uint
	for _, m := range nodeModels {
		switch m.Protocol {
		case "shadowsocks":
			ssNodeIDs = append(ssNodeIDs, m.ID)
		case "trojan":
			trojanNodeIDs = append(trojanNodeIDs, m.ID)
		}
	}

	// Load protocol-specific configs
	ssConfigsRaw, err := r.shadowsocksConfigRepo.GetByNodeIDs(ctx, ssNodeIDs)
	if err != nil {
		r.logger.Errorw("failed to get shadowsocks configs", "error", err)
		return nil, 0, fmt.Errorf("failed to get shadowsocks configs: %w", err)
	}

	// Convert to mapper format
	ssConfigs := make(map[uint]*mappers.ShadowsocksConfigData)
	for nodeID, data := range ssConfigsRaw {
		ssConfigs[nodeID] = &mappers.ShadowsocksConfigData{
			EncryptionConfig: data.EncryptionConfig,
			PluginConfig:     data.PluginConfig,
		}
	}

	trojanConfigs, err := r.trojanConfigRepo.GetByNodeIDs(ctx, trojanNodeIDs)
	if err != nil {
		r.logger.Errorw("failed to get trojan configs", "error", err)
		return nil, 0, fmt.Errorf("failed to get trojan configs: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(nodeModels, ssConfigs, trojanConfigs)
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

// UpdateLastSeenAt updates the last_seen_at timestamp for a node
func (r *NodeRepositoryImpl) UpdateLastSeenAt(ctx context.Context, nodeID uint) error {
	result := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("id = ?", nodeID).
		UpdateColumn("last_seen_at", gorm.Expr("NOW()"))

	if result.Error != nil {
		r.logger.Errorw("failed to update last_seen_at", "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to update last_seen_at: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("node not found")
	}

	r.logger.Debugw("last_seen_at updated successfully", "node_id", nodeID)
	return nil
}

// GetLastSeenAt retrieves just the last_seen_at timestamp for a node (lightweight query)
func (r *NodeRepositoryImpl) GetLastSeenAt(ctx context.Context, nodeID uint) (*node.Node, error) {
	var model models.NodeModel
	err := r.db.WithContext(ctx).
		Select("id", "last_seen_at").
		Where("id = ?", nodeID).
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get last_seen_at", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to get last_seen_at: %w", err)
	}

	// Create a minimal node entity with just the lastSeenAt field
	// We use empty/default values for other fields since we only need lastSeenAt
	serverAddr, _ := vo.NewServerAddress("0.0.0.0")
	nodeEntity, _ := node.ReconstructNode(
		model.ID,
		"",
		serverAddr,
		1,   // agentPort (placeholder, just needs to be non-zero)
		nil, // subscriptionPort
		vo.Protocol("shadowsocks"),
		vo.EncryptionConfig{},
		nil,
		nil,
		vo.NodeStatusInactive,
		vo.NewNodeMetadata("", nil, ""),
		"placeholder",
		0,
		nil,
		model.LastSeenAt,
		0,
		model.CreatedAt,
		model.UpdatedAt,
	)

	return nodeEntity, nil
}
