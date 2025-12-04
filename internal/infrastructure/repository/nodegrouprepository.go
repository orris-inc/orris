package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// NodeGroupRepositoryImpl implements the node.NodeGroupRepository interface
type NodeGroupRepositoryImpl struct {
	db                    *gorm.DB
	mapper                mappers.NodeGroupMapper
	nodeMapper            mappers.NodeMapper
	trojanConfigRepo      *TrojanConfigRepository
	shadowsocksConfigRepo *ShadowsocksConfigRepository
	logger                logger.Interface
}

// NewNodeGroupRepository creates a new node group repository instance
func NewNodeGroupRepository(db *gorm.DB, logger logger.Interface) node.NodeGroupRepository {
	return &NodeGroupRepositoryImpl{
		db:                    db,
		mapper:                mappers.NewNodeGroupMapper(),
		nodeMapper:            mappers.NewNodeMapper(),
		trojanConfigRepo:      NewTrojanConfigRepository(db, logger),
		shadowsocksConfigRepo: NewShadowsocksConfigRepository(db, logger),
		logger:                logger,
	}
}

// Create creates a new node group in the database
func (r *NodeGroupRepositoryImpl) Create(ctx context.Context, group *node.NodeGroup) error {
	model, err := r.mapper.ToModel(group)
	if err != nil {
		r.logger.Errorw("failed to map node group entity to model", "error", err)
		return fmt.Errorf("failed to map node group entity: %w", err)
	}

	// Start a transaction to handle main entity and relationships
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create the main node group entity
		if err := tx.Create(model).Error; err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "duplicate key") {
				if strings.Contains(err.Error(), "name") {
					return errors.NewConflictError("node group with this name already exists")
				}
				return errors.NewConflictError("node group already exists")
			}
			r.logger.Errorw("failed to create node group in database", "error", err)
			return fmt.Errorf("failed to create node group: %w", err)
		}

		// Set the ID back to the entity
		if err := group.SetID(model.ID); err != nil {
			r.logger.Errorw("failed to set node group ID", "error", err)
			return fmt.Errorf("failed to set node group ID: %w", err)
		}

		// Create node associations
		nodeIDs := mappers.GetNodeGroupNodeIDs(group)
		if len(nodeIDs) > 0 {
			if err := r.createNodeAssociations(tx, model.ID, nodeIDs); err != nil {
				return err
			}
		}

		// Create plan associations
		planIDs := mappers.GetNodeGroupSubscriptionPlanIDs(group)
		if len(planIDs) > 0 {
			if err := r.createPlanAssociations(tx, model.ID, planIDs); err != nil {
				return err
			}
		}

		r.logger.Infow("node group created successfully", "id", model.ID, "name", model.Name)
		return nil
	})
}

// GetByID retrieves a node group by its ID
func (r *NodeGroupRepositoryImpl) GetByID(ctx context.Context, id uint) (*node.NodeGroup, error) {
	var model models.NodeGroupModel

	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("node group not found")
		}
		r.logger.Errorw("failed to get node group by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get node group: %w", err)
	}

	// Load node associations
	nodeIDs, err := r.loadNodeIDs(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load plan associations
	planIDs, err := r.loadPlanIDs(ctx, id)
	if err != nil {
		return nil, err
	}

	// Map to entity
	entity, err := r.mapper.ToEntity(&model, nodeIDs, planIDs)
	if err != nil {
		r.logger.Errorw("failed to map node group model to entity", "id", id, "error", err)
		return nil, fmt.Errorf("failed to map node group: %w", err)
	}

	return entity, nil
}

// Update updates an existing node group with optimistic locking
func (r *NodeGroupRepositoryImpl) Update(ctx context.Context, group *node.NodeGroup) error {
	model, err := r.mapper.ToModel(group)
	if err != nil {
		r.logger.Errorw("failed to map node group entity to model", "error", err)
		return fmt.Errorf("failed to map node group entity: %w", err)
	}

	// Use transaction to ensure atomicity of updates including associations
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Use optimistic locking by checking version
		// The domain entity has already incremented the version, so we need to check against the previous version
		previousVersion := model.Version - 1
		result := tx.Model(&models.NodeGroupModel{}).
			Where("id = ? AND version = ?", model.ID, previousVersion).
			Updates(map[string]interface{}{
				"name":        model.Name,
				"description": model.Description,
				"is_public":   model.IsPublic,
				"sort_order":  model.SortOrder,
				"metadata":    model.Metadata,
				"updated_at":  model.UpdatedAt,
				"version":     model.Version,
			})

		if result.Error != nil {
			if strings.Contains(result.Error.Error(), "Duplicate entry") || strings.Contains(result.Error.Error(), "duplicate key") {
				if strings.Contains(result.Error.Error(), "name") {
					return errors.NewConflictError("node group with this name already exists")
				}
				return errors.NewConflictError("node group already exists")
			}
			r.logger.Errorw("failed to update node group", "id", model.ID, "error", result.Error)
			return fmt.Errorf("failed to update node group: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return errors.NewConflictError("node group has been modified by another transaction or not found")
		}

		// Synchronize node associations
		nodeIDs := mappers.GetNodeGroupNodeIDs(group)
		if err := r.syncNodeAssociations(tx, model.ID, nodeIDs); err != nil {
			return err
		}

		// Synchronize plan associations
		planIDs := mappers.GetNodeGroupSubscriptionPlanIDs(group)
		if err := r.syncPlanAssociations(tx, model.ID, planIDs); err != nil {
			return err
		}

		r.logger.Infow("node group updated successfully", "id", model.ID, "name", model.Name)
		return nil
	})
}

// Delete soft deletes a node group
// Note: Foreign key constraints have been removed for flexibility.
// Associated records in node_group_nodes and node_group_plans are not deleted
// and will be filtered by deleted_at when querying.
func (r *NodeGroupRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.NodeGroupModel{}, id)
	if result.Error != nil {
		r.logger.Errorw("failed to delete node group", "id", id, "error", result.Error)
		return fmt.Errorf("failed to delete node group: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("node group not found")
	}

	r.logger.Infow("node group deleted successfully", "id", id)
	return nil
}

// List retrieves a paginated list of node groups with filtering
func (r *NodeGroupRepositoryImpl) List(ctx context.Context, filter node.NodeGroupFilter) ([]*node.NodeGroup, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.NodeGroupModel{})

	// Apply filters
	if filter.Name != nil && *filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+*filter.Name+"%")
	}
	if filter.IsPublic != nil {
		query = query.Where("is_public = ?", *filter.IsPublic)
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count node groups", "error", err)
		return nil, 0, fmt.Errorf("failed to count node groups: %w", err)
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
	var groupModels []*models.NodeGroupModel
	if err := query.Find(&groupModels).Error; err != nil {
		r.logger.Errorw("failed to list node groups", "error", err)
		return nil, 0, fmt.Errorf("failed to list node groups: %w", err)
	}

	// Load associations for all groups
	nodeIDsMap := make(map[uint][]uint)
	planIDsMap := make(map[uint][]uint)

	for _, model := range groupModels {
		nodeIDs, err := r.loadNodeIDs(ctx, model.ID)
		if err != nil {
			return nil, 0, err
		}
		nodeIDsMap[model.ID] = nodeIDs

		planIDs, err := r.loadPlanIDs(ctx, model.ID)
		if err != nil {
			return nil, 0, err
		}
		planIDsMap[model.ID] = planIDs
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(groupModels, nodeIDsMap, planIDsMap)
	if err != nil {
		r.logger.Errorw("failed to map node group models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map node groups: %w", err)
	}

	return entities, total, nil
}

// AddNode adds a node to a node group
func (r *NodeGroupRepositoryImpl) AddNode(ctx context.Context, groupID, nodeID uint) error {
	association := &models.NodeGroupNodeModel{
		NodeGroupID: groupID,
		NodeID:      nodeID,
	}

	if err := r.db.WithContext(ctx).Create(association).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "duplicate key") {
			// Already associated, not an error
			return nil
		}
		r.logger.Errorw("failed to add node to group", "group_id", groupID, "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to add node to group: %w", err)
	}

	r.logger.Infow("node added to group successfully", "group_id", groupID, "node_id", nodeID)
	return nil
}

// RemoveNode removes a node from a node group
func (r *NodeGroupRepositoryImpl) RemoveNode(ctx context.Context, groupID, nodeID uint) error {
	result := r.db.WithContext(ctx).
		Where("node_group_id = ? AND node_id = ?", groupID, nodeID).
		Delete(&models.NodeGroupNodeModel{})

	if result.Error != nil {
		r.logger.Errorw("failed to remove node from group", "group_id", groupID, "node_id", nodeID, "error", result.Error)
		return fmt.Errorf("failed to remove node from group: %w", result.Error)
	}

	r.logger.Infow("node removed from group successfully", "group_id", groupID, "node_id", nodeID)
	return nil
}

// GetNodesByGroupID retrieves all nodes in a node group
func (r *NodeGroupRepositoryImpl) GetNodesByGroupID(ctx context.Context, groupID uint) ([]*node.Node, error) {
	var nodeModels []*models.NodeModel

	err := r.db.WithContext(ctx).
		Joins("JOIN node_group_nodes ON nodes.id = node_group_nodes.node_id").
		Where("node_group_nodes.node_group_id = ?", groupID).
		Order("nodes.sort_order ASC").
		Find(&nodeModels).Error

	if err != nil {
		r.logger.Errorw("failed to get nodes by group ID", "group_id", groupID, "error", err)
		return nil, fmt.Errorf("failed to get nodes: %w", err)
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
		return nil, fmt.Errorf("failed to get shadowsocks configs: %w", err)
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
		return nil, fmt.Errorf("failed to get trojan configs: %w", err)
	}

	// Convert models to entities
	entities, err := r.nodeMapper.ToEntities(nodeModels, ssConfigs, trojanConfigs)
	if err != nil {
		r.logger.Errorw("failed to map node models to entities", "error", err)
		return nil, fmt.Errorf("failed to map nodes: %w", err)
	}

	return entities, nil
}

// AssociateSubscriptionPlan associates a subscription plan with a node group
func (r *NodeGroupRepositoryImpl) AssociateSubscriptionPlan(ctx context.Context, groupID, planID uint) error {
	association := &models.NodeGroupPlanModel{
		NodeGroupID:        groupID,
		SubscriptionPlanID: planID,
	}

	if err := r.db.WithContext(ctx).Create(association).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "duplicate key") {
			// Already associated, not an error
			return nil
		}
		r.logger.Errorw("failed to associate plan with group", "group_id", groupID, "plan_id", planID, "error", err)
		return fmt.Errorf("failed to associate plan with group: %w", err)
	}

	r.logger.Infow("plan associated with group successfully", "group_id", groupID, "plan_id", planID)
	return nil
}

// DisassociateSubscriptionPlan removes a subscription plan association from a node group
func (r *NodeGroupRepositoryImpl) DisassociateSubscriptionPlan(ctx context.Context, groupID, planID uint) error {
	result := r.db.WithContext(ctx).
		Where("node_group_id = ? AND subscription_plan_id = ?", groupID, planID).
		Delete(&models.NodeGroupPlanModel{})

	if result.Error != nil {
		r.logger.Errorw("failed to disassociate plan from group", "group_id", groupID, "plan_id", planID, "error", result.Error)
		return fmt.Errorf("failed to disassociate plan from group: %w", result.Error)
	}

	r.logger.Infow("plan disassociated from group successfully", "group_id", groupID, "plan_id", planID)
	return nil
}

// GetSubscriptionPlansByGroupID retrieves all subscription plan IDs associated with a node group
func (r *NodeGroupRepositoryImpl) GetSubscriptionPlansByGroupID(ctx context.Context, groupID uint) ([]uint, error) {
	return r.loadPlanIDs(ctx, groupID)
}

// ExistsByName checks if a node group with the given name exists
func (r *NodeGroupRepositoryImpl) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.NodeGroupModel{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check node group existence by name", "name", name, "error", err)
		return false, fmt.Errorf("failed to check node group existence: %w", err)
	}
	return count > 0, nil
}

// Helper methods

// createNodeAssociations creates node associations in a transaction
func (r *NodeGroupRepositoryImpl) createNodeAssociations(tx *gorm.DB, groupID uint, nodeIDs []uint) error {
	for _, nodeID := range nodeIDs {
		association := &models.NodeGroupNodeModel{
			NodeGroupID: groupID,
			NodeID:      nodeID,
		}
		if err := tx.Create(association).Error; err != nil {
			r.logger.Errorw("failed to create node association", "group_id", groupID, "node_id", nodeID, "error", err)
			return fmt.Errorf("failed to create node association: %w", err)
		}
	}
	return nil
}

// createPlanAssociations creates plan associations in a transaction
func (r *NodeGroupRepositoryImpl) createPlanAssociations(tx *gorm.DB, groupID uint, planIDs []uint) error {
	for _, planID := range planIDs {
		association := &models.NodeGroupPlanModel{
			NodeGroupID:        groupID,
			SubscriptionPlanID: planID,
		}
		if err := tx.Create(association).Error; err != nil {
			r.logger.Errorw("failed to create plan association", "group_id", groupID, "plan_id", planID, "error", err)
			return fmt.Errorf("failed to create plan association: %w", err)
		}
	}
	return nil
}

// loadNodeIDs loads node IDs for a node group
func (r *NodeGroupRepositoryImpl) loadNodeIDs(ctx context.Context, groupID uint) ([]uint, error) {
	var nodeIDs []uint
	err := r.db.WithContext(ctx).Model(&models.NodeGroupNodeModel{}).
		Where("node_group_id = ?", groupID).
		Pluck("node_id", &nodeIDs).Error

	if err != nil {
		r.logger.Errorw("failed to load node IDs", "group_id", groupID, "error", err)
		return nil, fmt.Errorf("failed to load node IDs: %w", err)
	}

	return nodeIDs, nil
}

// loadPlanIDs loads plan IDs for a node group
func (r *NodeGroupRepositoryImpl) loadPlanIDs(ctx context.Context, groupID uint) ([]uint, error) {
	var planIDs []uint
	err := r.db.WithContext(ctx).Model(&models.NodeGroupPlanModel{}).
		Where("node_group_id = ?", groupID).
		Pluck("subscription_plan_id", &planIDs).Error

	if err != nil {
		r.logger.Errorw("failed to load plan IDs", "group_id", groupID, "error", err)
		return nil, fmt.Errorf("failed to load plan IDs: %w", err)
	}

	return planIDs, nil
}

// syncNodeAssociations synchronizes node associations in a transaction
// Deletes all existing associations and creates new ones based on the current state
func (r *NodeGroupRepositoryImpl) syncNodeAssociations(tx *gorm.DB, groupID uint, nodeIDs []uint) error {
	// Delete all existing node associations
	if err := tx.Where("node_group_id = ?", groupID).Delete(&models.NodeGroupNodeModel{}).Error; err != nil {
		r.logger.Errorw("failed to delete existing node associations", "group_id", groupID, "error", err)
		return fmt.Errorf("failed to delete existing node associations: %w", err)
	}

	// Create new node associations
	if len(nodeIDs) > 0 {
		if err := r.createNodeAssociations(tx, groupID, nodeIDs); err != nil {
			return err
		}
	}

	return nil
}

// syncPlanAssociations synchronizes plan associations in a transaction
// Deletes all existing associations and creates new ones based on the current state
func (r *NodeGroupRepositoryImpl) syncPlanAssociations(tx *gorm.DB, groupID uint, planIDs []uint) error {
	// Delete all existing plan associations
	if err := tx.Where("node_group_id = ?", groupID).Delete(&models.NodeGroupPlanModel{}).Error; err != nil {
		r.logger.Errorw("failed to delete existing plan associations", "group_id", groupID, "error", err)
		return fmt.Errorf("failed to delete existing plan associations: %w", err)
	}

	// Create new plan associations
	if len(planIDs) > 0 {
		if err := r.createPlanAssociations(tx, groupID, planIDs); err != nil {
			return err
		}
	}

	return nil
}
