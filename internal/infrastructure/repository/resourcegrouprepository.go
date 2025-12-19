package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ResourceGroupRepositoryImpl implements the resource.Repository interface.
type ResourceGroupRepositoryImpl struct {
	db     *gorm.DB
	mapper mappers.ResourceGroupMapper
	logger logger.Interface
}

// NewResourceGroupRepository creates a new resource group repository instance.
func NewResourceGroupRepository(db *gorm.DB, logger logger.Interface) resource.Repository {
	return &ResourceGroupRepositoryImpl{
		db:     db,
		mapper: mappers.NewResourceGroupMapper(),
		logger: logger,
	}
}

// Create creates a new resource group in the database.
func (r *ResourceGroupRepositoryImpl) Create(ctx context.Context, group *resource.ResourceGroup) error {
	model, err := r.mapper.ToModel(group)
	if err != nil {
		r.logger.Errorw("failed to map resource group entity to model", "error", err)
		return fmt.Errorf("failed to map resource group entity: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "duplicate key") {
			return errors.NewConflictError("resource group already exists")
		}
		r.logger.Errorw("failed to create resource group in database", "error", err)
		return fmt.Errorf("failed to create resource group: %w", err)
	}

	if err := group.SetID(model.ID); err != nil {
		r.logger.Errorw("failed to set resource group ID", "error", err)
		return fmt.Errorf("failed to set resource group ID: %w", err)
	}

	r.logger.Infow("resource group created successfully", "id", model.ID, "sid", model.SID, "name", model.Name)
	return nil
}

// Update updates an existing resource group.
func (r *ResourceGroupRepositoryImpl) Update(ctx context.Context, group *resource.ResourceGroup) error {
	model, err := r.mapper.ToModel(group)
	if err != nil {
		r.logger.Errorw("failed to map resource group entity to model", "error", err)
		return fmt.Errorf("failed to map resource group entity: %w", err)
	}

	// Optimistic locking: update only if version matches
	result := r.db.WithContext(ctx).Model(&models.ResourceGroupModel{}).
		Where("id = ? AND version = ?", model.ID, model.Version-1).
		Updates(map[string]any{
			"name":        model.Name,
			"plan_id":     model.PlanID,
			"description": model.Description,
			"status":      model.Status,
			"updated_at":  model.UpdatedAt,
			"version":     model.Version,
		})

	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "Duplicate entry") || strings.Contains(result.Error.Error(), "duplicate key") {
			return errors.NewConflictError("resource group already exists")
		}
		r.logger.Errorw("failed to update resource group", "id", model.ID, "error", result.Error)
		return fmt.Errorf("failed to update resource group: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return resource.ErrVersionConflict
	}

	r.logger.Infow("resource group updated successfully", "id", model.ID, "name", model.Name)
	return nil
}

// Delete soft deletes a resource group by ID.
func (r *ResourceGroupRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Model(&models.ResourceGroupModel{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"status":     "inactive",
			"deleted_at": gorm.Expr("NOW()"),
		})

	if result.Error != nil {
		r.logger.Errorw("failed to delete resource group", "id", id, "error", result.Error)
		return fmt.Errorf("failed to delete resource group: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("resource group", fmt.Sprintf("%d", id))
	}

	r.logger.Infow("resource group deleted successfully", "id", id)
	return nil
}

// GetByID retrieves a resource group by its ID.
func (r *ResourceGroupRepositoryImpl) GetByID(ctx context.Context, id uint) (*resource.ResourceGroup, error) {
	var model models.ResourceGroupModel

	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get resource group by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map resource group model to entity", "id", id, "error", err)
		return nil, fmt.Errorf("failed to map resource group: %w", err)
	}

	return entity, nil
}

// GetBySID retrieves a resource group by its Stripe-style ID.
func (r *ResourceGroupRepositoryImpl) GetBySID(ctx context.Context, sid string) (*resource.ResourceGroup, error) {
	var model models.ResourceGroupModel

	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		r.logger.Errorw("failed to get resource group by SID", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}

	entity, err := r.mapper.ToEntity(&model)
	if err != nil {
		r.logger.Errorw("failed to map resource group model to entity", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to map resource group: %w", err)
	}

	return entity, nil
}

// GetByPlanID retrieves all resource groups for a plan.
func (r *ResourceGroupRepositoryImpl) GetByPlanID(ctx context.Context, planID uint) ([]*resource.ResourceGroup, error) {
	var modelList []*models.ResourceGroupModel

	if err := r.db.WithContext(ctx).Where("plan_id = ?", planID).Find(&modelList).Error; err != nil {
		r.logger.Errorw("failed to get resource groups by plan ID", "plan_id", planID, "error", err)
		return nil, fmt.Errorf("failed to get resource groups: %w", err)
	}

	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		r.logger.Errorw("failed to map resource group models to entities", "plan_id", planID, "error", err)
		return nil, fmt.Errorf("failed to map resource groups: %w", err)
	}

	return entities, nil
}

// List retrieves resource groups with optional filters.
func (r *ResourceGroupRepositoryImpl) List(ctx context.Context, filter resource.ListFilter) ([]*resource.ResourceGroup, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.ResourceGroupModel{})

	// Apply filters
	if filter.PlanID != nil {
		query = query.Where("plan_id = ?", *filter.PlanID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", filter.Status.String())
	}
	if filter.Search != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Errorw("failed to count resource groups", "error", err)
		return nil, 0, fmt.Errorf("failed to count resource groups: %w", err)
	}

	// Apply sorting and pagination
	query = query.Order("created_at DESC")
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Execute query
	var modelList []*models.ResourceGroupModel
	if err := query.Find(&modelList).Error; err != nil {
		r.logger.Errorw("failed to list resource groups", "error", err)
		return nil, 0, fmt.Errorf("failed to list resource groups: %w", err)
	}

	// Convert models to entities
	entities, err := r.mapper.ToEntities(modelList)
	if err != nil {
		r.logger.Errorw("failed to map resource group models to entities", "error", err)
		return nil, 0, fmt.Errorf("failed to map resource groups: %w", err)
	}

	return entities, total, nil
}

// ExistsByName checks if a resource group with the given name exists.
func (r *ResourceGroupRepositoryImpl) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.ResourceGroupModel{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		r.logger.Errorw("failed to check resource group existence by name", "name", name, "error", err)
		return false, fmt.Errorf("failed to check resource group existence: %w", err)
	}
	return count > 0, nil
}
