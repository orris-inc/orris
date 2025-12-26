package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// allowedNodeOrderByFields defines the whitelist of allowed ORDER BY fields
// to prevent SQL injection attacks.
var allowedNodeOrderByFields = map[string]bool{
	"id":             true,
	"sid":            true,
	"name":           true,
	"server_address": true,
	"agent_port":     true,
	"protocol":       true,
	"status":         true,
	"user_id":        true,
	"region":         true,
	"sort_order":     true,
	"last_seen_at":   true,
	"created_at":     true,
	"updated_at":     true,
}

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

// GetBySID retrieves a node by its SID
func (r *NodeRepositoryImpl) GetBySID(ctx context.Context, sid string) (*node.Node, error) {
	var model models.NodeModel

	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("node not found")
		}
		r.logger.Errorw("failed to get node by SID", "sid", sid, "error", err)
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
		r.logger.Errorw("failed to map node model to entity", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to map node: %w", err)
	}

	return entity, nil
}

// GetByIDs retrieves nodes by their IDs
func (r *NodeRepositoryImpl) GetByIDs(ctx context.Context, ids []uint) ([]*node.Node, error) {
	if len(ids) == 0 {
		return []*node.Node{}, nil
	}

	var nodeModels []*models.NodeModel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&nodeModels).Error; err != nil {
		r.logger.Errorw("failed to get nodes by IDs", "ids", ids, "error", err)
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	// Collect node IDs for batch loading protocol configs
	nodeIDs := make([]uint, len(nodeModels))
	for i, m := range nodeModels {
		nodeIDs[i] = m.ID
	}

	// Load shadowsocks configs
	ssConfigsRaw, err := r.shadowsocksConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load shadowsocks configs", "error", err)
		ssConfigsRaw = make(map[uint]*ShadowsocksConfigData)
	}

	// Convert to mapper format
	ssConfigs := make(map[uint]*mappers.ShadowsocksConfigData)
	for nodeID, data := range ssConfigsRaw {
		ssConfigs[nodeID] = &mappers.ShadowsocksConfigData{
			EncryptionConfig: data.EncryptionConfig,
			PluginConfig:     data.PluginConfig,
		}
	}

	// Load trojan configs
	trojanConfigs, err := r.trojanConfigRepo.GetByNodeIDs(ctx, nodeIDs)
	if err != nil {
		r.logger.Warnw("failed to load trojan configs", "error", err)
		trojanConfigs = make(map[uint]*vo.TrojanConfig)
	}

	// Convert to entities
	entities, err := r.mapper.ToEntities(nodeModels, ssConfigs, trojanConfigs)
	if err != nil {
		r.logger.Errorw("failed to map node models to entities", "error", err)
		return nil, fmt.Errorf("failed to map nodes: %w", err)
	}

	return entities, nil
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

	// Calculate the expected previous version for optimistic locking
	// Domain layer increments version on each update, so we check against version - 1
	expectedVersion := model.Version - 1
	if expectedVersion < 1 {
		expectedVersion = 1
	}

	// Use transaction to update node and protocol config atomically
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Use Select to explicitly specify fields to update, including nullable fields like group_ids
		// This ensures GORM updates NULL values correctly (without Select, GORM ignores nil values in map)
		// Use optimistic locking: WHERE id = ? AND version = expectedVersion
		result := tx.Model(&models.NodeModel{}).
			Where("id = ? AND version = ?", model.ID, expectedVersion).
			Select(
				"name", "server_address", "agent_port", "subscription_port",
				"protocol", "status", "region", "tags", "sort_order",
				"maintenance_reason", "token_hash", "api_token", "group_ids", "version", "updated_at",
			).
			Updates(model)

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
			// Check if the record exists to distinguish between not found and version conflict
			var count int64
			if err := tx.Model(&models.NodeModel{}).Scopes(db.NotDeleted()).Where("id = ?", model.ID).Count(&count).Error; err == nil && count > 0 {
				return errors.NewConflictError("node was modified by another request, please retry")
			}
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
	if filter.AdminOnly != nil && *filter.AdminOnly {
		query = query.Where("user_id IS NULL")
	} else if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Name != nil && *filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Status != nil && *filter.Status != "" {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Tag != nil && *filter.Tag != "" {
		// Search in JSON tags array using proper JSON encoding to handle special characters
		tagJSON, _ := json.Marshal(*filter.Tag)
		query = query.Where("JSON_CONTAINS(tags, ?)", string(tagJSON))
	}
	if len(filter.GroupIDs) > 0 {
		// Use JSON_OVERLAPS to check if group_ids array contains any of the filter group IDs
		// JSON_OVERLAPS returns true if two JSON arrays have at least one element in common
		query = query.Where("JSON_OVERLAPS(group_ids, ?)", fmt.Sprintf("[%s]", uintSliceToString(filter.GroupIDs)))
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count nodes", "error", err)
		return nil, 0, fmt.Errorf("failed to count nodes: %w", err)
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	sortBy := strings.ToLower(filter.SortFilter.SortBy)
	if sortBy != "" && allowedNodeOrderByFields[sortBy] {
		order := "ASC"
		if filter.SortFilter.IsDescending() {
			order = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", sortBy, order))
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

// ExistsByName checks if a node with the given name exists (excluding soft-deleted records)
func (r *NodeRepositoryImpl) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Scopes(db.NotDeleted()).
		Where("name = ?", name).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by name", "name", name, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByNameExcluding checks if a node with the given name exists, excluding a specific node ID (excluding soft-deleted records)
func (r *NodeRepositoryImpl) ExistsByNameExcluding(ctx context.Context, name string, excludeID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Scopes(db.NotDeleted()).
		Where("name = ? AND id != ?", name, excludeID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by name", "name", name, "exclude_id", excludeID, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByAddress checks if a node with the given address and port exists (excluding soft-deleted records)
func (r *NodeRepositoryImpl) ExistsByAddress(ctx context.Context, address string, port int) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Scopes(db.NotDeleted()).
		Where("server_address = ? AND agent_port = ?", address, port).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by address", "address", address, "port", port, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByAddressExcluding checks if a node with the given address and port exists, excluding a specific node ID (excluding soft-deleted records)
func (r *NodeRepositoryImpl) ExistsByAddressExcluding(ctx context.Context, address string, port int, excludeID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Scopes(db.NotDeleted()).
		Where("server_address = ? AND agent_port = ? AND id != ?", address, port, excludeID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by address", "address", address, "port", port, "exclude_id", excludeID, "error", err)
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

// UpdateLastSeenAt updates the last_seen_at timestamp and public IPs for a node
// Uses conditional update to avoid race conditions: only updates if last_seen_at is NULL
// or older than the threshold (2 minutes). This moves the throttling logic to the database
// layer for atomic operation.
func (r *NodeRepositoryImpl) UpdateLastSeenAt(ctx context.Context, nodeID uint, publicIPv4, publicIPv6 string) error {
	updates := map[string]interface{}{
		"last_seen_at": gorm.Expr("NOW()"),
	}

	// Only update public IPs if provided
	if publicIPv4 != "" {
		updates["public_ipv4"] = publicIPv4
	}
	if publicIPv6 != "" {
		updates["public_ipv6"] = publicIPv6
	}

	// Use conditional update to prevent race conditions
	// Only update if last_seen_at is NULL or older than 2 minutes
	result := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Where("id = ? AND (last_seen_at IS NULL OR last_seen_at < NOW() - INTERVAL 2 MINUTE)", nodeID).
		Updates(updates)

	if result.Error != nil {
		r.logger.Errorw("failed to update last_seen_at", "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to update last_seen_at: %w", result.Error)
	}

	// RowsAffected == 0 is normal when throttled, not an error
	if result.RowsAffected > 0 {
		r.logger.Debugw("last_seen_at updated successfully",
			"node_id", nodeID,
			"public_ipv4", publicIPv4,
			"public_ipv6", publicIPv6,
		)
	}
	return nil
}

// GetLastSeenAt retrieves just the last_seen_at timestamp for a node (lightweight query)
// Returns NotFoundError if the node does not exist
func (r *NodeRepositoryImpl) GetLastSeenAt(ctx context.Context, nodeID uint) (*time.Time, error) {
	var model models.NodeModel
	err := r.db.WithContext(ctx).
		Select("id", "last_seen_at").
		Where("id = ?", nodeID).
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("node not found")
		}
		r.logger.Errorw("failed to get last_seen_at", "node_id", nodeID, "error", err)
		return nil, fmt.Errorf("failed to get last_seen_at: %w", err)
	}

	return model.LastSeenAt, nil
}

// uintSliceToString converts a slice of uint to a comma-separated string
// Used for JSON_OVERLAPS query parameter
func uintSliceToString(ids []uint) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return strings.Join(parts, ",")
}

// ListByUserID returns nodes owned by a specific user
func (r *NodeRepositoryImpl) ListByUserID(ctx context.Context, userID uint, filter node.NodeFilter) ([]*node.Node, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.NodeModel{}).Where("user_id = ?", userID)

	// Apply filters
	if filter.Name != nil && *filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Status != nil && *filter.Status != "" {
		query = query.Where("status = ?", *filter.Status)
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count user nodes", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("failed to count user nodes: %w", err)
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	sortBy := strings.ToLower(filter.SortFilter.SortBy)
	if sortBy != "" && allowedNodeOrderByFields[sortBy] {
		order := "ASC"
		if filter.SortFilter.IsDescending() {
			order = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", sortBy, order))
	} else {
		query = query.Order("created_at DESC")
	}

	// Apply pagination
	offset := filter.PageFilter.Offset()
	limit := filter.PageFilter.Limit()
	query = query.Offset(offset).Limit(limit)

	// Execute query
	var nodeModels []*models.NodeModel
	if err := query.Find(&nodeModels).Error; err != nil {
		r.logger.Errorw("failed to list user nodes", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("failed to list user nodes: %w", err)
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

	entities, err := r.mapper.ToEntities(nodeModels, ssConfigs, trojanConfigs)
	if err != nil {
		r.logger.Errorw("failed to map node models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map nodes: %w", err)
	}

	return entities, total, nil
}

// CountByUserID counts nodes owned by a specific user (excluding soft-deleted records)
func (r *NodeRepositoryImpl) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Scopes(db.NotDeleted()).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to count user nodes", "user_id", userID, "error", err)
		return 0, fmt.Errorf("failed to count user nodes: %w", err)
	}
	return count, nil
}

// ExistsByNameForUser checks if a node with the given name exists for a specific user (excluding soft-deleted records)
func (r *NodeRepositoryImpl) ExistsByNameForUser(ctx context.Context, name string, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Scopes(db.NotDeleted()).
		Where("name = ? AND user_id = ?", name, userID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by name for user", "name", name, "user_id", userID, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByNameForUserExcluding checks if a node with the given name exists for a user, excluding a specific node (excluding soft-deleted records)
func (r *NodeRepositoryImpl) ExistsByNameForUserExcluding(ctx context.Context, name string, userID uint, excludeID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Scopes(db.NotDeleted()).
		Where("name = ? AND user_id = ? AND id != ?", name, userID, excludeID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by name for user", "name", name, "user_id", userID, "exclude_id", excludeID, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByAddressForUser checks if a node with the given address and port exists for a specific user (excluding soft-deleted records)
func (r *NodeRepositoryImpl) ExistsByAddressForUser(ctx context.Context, address string, port int, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Scopes(db.NotDeleted()).
		Where("server_address = ? AND agent_port = ? AND user_id = ?", address, port, userID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by address for user", "address", address, "port", port, "user_id", userID, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}

// ExistsByAddressForUserExcluding checks if a node with the given address and port exists for a user, excluding a specific node (excluding soft-deleted records)
func (r *NodeRepositoryImpl) ExistsByAddressForUserExcluding(ctx context.Context, address string, port int, userID uint, excludeID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeModel{}).
		Scopes(db.NotDeleted()).
		Where("server_address = ? AND agent_port = ? AND user_id = ? AND id != ?", address, port, userID, excludeID).
		Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node existence by address for user", "address", address, "port", port, "user_id", userID, "exclude_id", excludeID, "error", err)
		return false, fmt.Errorf("failed to check node existence: %w", err)
	}
	return count > 0, nil
}
