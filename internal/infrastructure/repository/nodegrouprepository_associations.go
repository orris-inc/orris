package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

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
