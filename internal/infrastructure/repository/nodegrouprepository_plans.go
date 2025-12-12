package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

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
