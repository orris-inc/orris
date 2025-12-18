package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// EntitlementRepositoryImpl implements the subscription.EntitlementRepository interface
type EntitlementRepositoryImpl struct {
	db     *gorm.DB
	logger logger.Interface
}

// NewEntitlementRepository creates a new entitlement repository instance
func NewEntitlementRepository(db *gorm.DB, logger logger.Interface) subscription.EntitlementRepository {
	return &EntitlementRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

// Create creates a new entitlement association
func (r *EntitlementRepositoryImpl) Create(ctx context.Context, resource *subscription.Entitlement) error {
	model := &models.EntitlementModel{
		PlanID:       resource.PlanID(),
		ResourceType: string(resource.ResourceType()),
		ResourceID:   resource.ResourceID(),
		CreatedAt:    resource.CreatedAt(),
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		if errors.IsDuplicateError(err) {
			return errors.NewConflictError("entitlement association already exists")
		}
		r.logger.Errorw("failed to create entitlement",
			"plan_id", resource.PlanID(),
			"resource_type", resource.ResourceType(),
			"resource_id", resource.ResourceID(),
			"error", err)
		return fmt.Errorf("failed to create entitlement: %w", err)
	}

	if err := resource.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set entitlement ID", "error", err)
		return fmt.Errorf("failed to set entitlement ID: %w", err)
	}

	r.logger.Infow("entitlement created",
		"id", model.ID,
		"plan_id", model.PlanID,
		"resource_type", model.ResourceType,
		"resource_id", model.ResourceID)

	return nil
}

// Delete removes an entitlement association by ID
func (r *EntitlementRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.EntitlementModel{}, id)
	if result.Error != nil {
		r.logger.Errorw("failed to delete entitlement", "id", id, "error", result.Error)
		return fmt.Errorf("failed to delete entitlement: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("entitlement not found")
	}

	r.logger.Infow("entitlement deleted", "id", id)
	return nil
}

// DeleteByPlanAndResource removes a specific association
func (r *EntitlementRepositoryImpl) DeleteByPlanAndResource(ctx context.Context, planID uint, resourceType subscription.EntitlementResourceType, resourceID uint) error {
	result := r.db.WithContext(ctx).
		Where("plan_id = ? AND resource_type = ? AND resource_id = ?", planID, string(resourceType), resourceID).
		Delete(&models.EntitlementModel{})

	if result.Error != nil {
		r.logger.Errorw("failed to delete entitlement",
			"plan_id", planID,
			"resource_type", resourceType,
			"resource_id", resourceID,
			"error", result.Error)
		return fmt.Errorf("failed to delete entitlement: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("entitlement association not found")
	}

	r.logger.Infow("entitlement deleted",
		"plan_id", planID,
		"resource_type", resourceType,
		"resource_id", resourceID)
	return nil
}

// GetByPlan returns all resources for a subscription plan
func (r *EntitlementRepositoryImpl) GetByPlan(ctx context.Context, planID uint) ([]*subscription.Entitlement, error) {
	var models []models.EntitlementModel
	if err := r.db.WithContext(ctx).
		Where("plan_id = ?", planID).
		Find(&models).Error; err != nil {
		r.logger.Errorw("failed to get entitlements", "plan_id", planID, "error", err)
		return nil, fmt.Errorf("failed to get entitlements: %w", err)
	}

	resources := make([]*subscription.Entitlement, len(models))
	for i, model := range models {
		resource, err := subscription.ReconstructEntitlement(
			model.ID,
			model.PlanID,
			model.ResourceType,
			model.ResourceID,
			model.CreatedAt,
		)
		if err != nil {
			r.logger.Errorw("failed to reconstruct entitlement", "id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to reconstruct entitlement: %w", err)
		}
		resources[i] = resource
	}

	return resources, nil
}

// GetByPlanAndType returns resources of a specific type for a subscription plan
func (r *EntitlementRepositoryImpl) GetByPlanAndType(ctx context.Context, planID uint, resourceType subscription.EntitlementResourceType) ([]*subscription.Entitlement, error) {
	var models []models.EntitlementModel
	if err := r.db.WithContext(ctx).
		Where("plan_id = ? AND resource_type = ?", planID, string(resourceType)).
		Find(&models).Error; err != nil {
		r.logger.Errorw("failed to get entitlements by type",
			"plan_id", planID,
			"resource_type", resourceType,
			"error", err)
		return nil, fmt.Errorf("failed to get entitlements: %w", err)
	}

	resources := make([]*subscription.Entitlement, len(models))
	for i, model := range models {
		resource, err := subscription.ReconstructEntitlement(
			model.ID,
			model.PlanID,
			model.ResourceType,
			model.ResourceID,
			model.CreatedAt,
		)
		if err != nil {
			r.logger.Errorw("failed to reconstruct entitlement", "id", model.ID, "error", err)
			return nil, fmt.Errorf("failed to reconstruct entitlement: %w", err)
		}
		resources[i] = resource
	}

	return resources, nil
}

// GetResourceIDs returns resource IDs of a specific type for a subscription plan
func (r *EntitlementRepositoryImpl) GetResourceIDs(ctx context.Context, planID uint, resourceType subscription.EntitlementResourceType) ([]uint, error) {
	var resourceIDs []uint
	if err := r.db.WithContext(ctx).
		Model(&models.EntitlementModel{}).
		Where("plan_id = ? AND resource_type = ?", planID, string(resourceType)).
		Pluck("resource_id", &resourceIDs).Error; err != nil {
		r.logger.Errorw("failed to get resource IDs",
			"plan_id", planID,
			"resource_type", resourceType,
			"error", err)
		return nil, fmt.Errorf("failed to get resource IDs: %w", err)
	}

	return resourceIDs, nil
}

// Exists checks if a specific association exists
func (r *EntitlementRepositoryImpl) Exists(ctx context.Context, planID uint, resourceType subscription.EntitlementResourceType, resourceID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.EntitlementModel{}).
		Where("plan_id = ? AND resource_type = ? AND resource_id = ?", planID, string(resourceType), resourceID).
		Count(&count).Error; err != nil {
		r.logger.Errorw("failed to check entitlement existence",
			"plan_id", planID,
			"resource_type", resourceType,
			"resource_id", resourceID,
			"error", err)
		return false, fmt.Errorf("failed to check entitlement existence: %w", err)
	}

	return count > 0, nil
}

// BatchCreate creates multiple entitlement associations
func (r *EntitlementRepositoryImpl) BatchCreate(ctx context.Context, resources []*subscription.Entitlement) error {
	if len(resources) == 0 {
		return nil
	}

	resourceModels := make([]models.EntitlementModel, len(resources))
	for i, resource := range resources {
		resourceModels[i] = models.EntitlementModel{
			PlanID:       resource.PlanID(),
			ResourceType: string(resource.ResourceType()),
			ResourceID:   resource.ResourceID(),
			CreatedAt:    resource.CreatedAt(),
		}
	}

	if err := r.db.WithContext(ctx).Create(&resourceModels).Error; err != nil {
		if errors.IsDuplicateError(err) {
			return errors.NewConflictError("one or more entitlement associations already exist")
		}
		r.logger.Errorw("failed to batch create entitlements", "count", len(resources), "error", err)
		return fmt.Errorf("failed to batch create entitlements: %w", err)
	}

	// Set IDs back to entities
	for i := range resources {
		if err := resources[i].SetID(resourceModels[i].ID); err != nil {
			r.logger.Warnw("failed to set entitlement ID after batch create", "index", i, "error", err)
		}
	}

	r.logger.Infow("entitlements batch created", "count", len(resources))
	return nil
}

// BatchDelete removes multiple entitlement associations by IDs
func (r *EntitlementRepositoryImpl) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).Delete(&models.EntitlementModel{}, ids)
	if result.Error != nil {
		r.logger.Errorw("failed to batch delete entitlements", "ids", ids, "error", result.Error)
		return fmt.Errorf("failed to batch delete entitlements: %w", result.Error)
	}

	r.logger.Infow("entitlements batch deleted", "count", result.RowsAffected)
	return nil
}

// DeleteAllByPlan removes all resources for a subscription plan
func (r *EntitlementRepositoryImpl) DeleteAllByPlan(ctx context.Context, planID uint) error {
	result := r.db.WithContext(ctx).
		Where("plan_id = ?", planID).
		Delete(&models.EntitlementModel{})

	if result.Error != nil {
		r.logger.Errorw("failed to delete all entitlements", "plan_id", planID, "error", result.Error)
		return fmt.Errorf("failed to delete all entitlements: %w", result.Error)
	}

	r.logger.Infow("all entitlements deleted", "plan_id", planID, "count", result.RowsAffected)
	return nil
}

// DeleteAllByResource removes all associations for a specific resource
func (r *EntitlementRepositoryImpl) DeleteAllByResource(ctx context.Context, resourceType subscription.EntitlementResourceType, resourceID uint) error {
	result := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", string(resourceType), resourceID).
		Delete(&models.EntitlementModel{})

	if result.Error != nil {
		r.logger.Errorw("failed to delete all resource associations",
			"resource_type", resourceType,
			"resource_id", resourceID,
			"error", result.Error)
		return fmt.Errorf("failed to delete all resource associations: %w", result.Error)
	}

	r.logger.Infow("all resource associations deleted",
		"resource_type", resourceType,
		"resource_id", resourceID,
		"count", result.RowsAffected)
	return nil
}
