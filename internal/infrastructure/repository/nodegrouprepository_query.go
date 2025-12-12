package repository

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/errors"
	"gorm.io/gorm"
)

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
