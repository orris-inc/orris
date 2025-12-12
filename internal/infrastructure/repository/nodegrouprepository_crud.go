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
)

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

// Update updates an existing node group
func (r *NodeGroupRepositoryImpl) Update(ctx context.Context, group *node.NodeGroup) error {
	model, err := r.mapper.ToModel(group)
	if err != nil {
		r.logger.Errorw("failed to map node group entity to model", "error", err)
		return fmt.Errorf("failed to map node group entity: %w", err)
	}

	// Use transaction to ensure atomicity of updates including associations
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.NodeGroupModel{}).
			Where("id = ?", model.ID).
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
			return errors.NewNotFoundError("node group not found")
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
