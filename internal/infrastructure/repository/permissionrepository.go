package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"orris/internal/domain/permission"
	vo "orris/internal/domain/permission/value_objects"
	"orris/internal/infrastructure/persistence/models"
	"orris/internal/shared/constants"
	"orris/internal/shared/errors"
)

type PermissionRepositoryImpl struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) permission.PermissionRepository {
	return &PermissionRepositoryImpl{db: db}
}

func (r *PermissionRepositoryImpl) Create(ctx context.Context, perm *permission.Permission) error {
	model := &models.PermissionModel{
		Resource:    perm.Resource().String(),
		Action:      perm.Action().String(),
		Description: perm.Description(),
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}

	return perm.SetID(model.ID)
}

func (r *PermissionRepositoryImpl) GetByID(ctx context.Context, id uint) (*permission.Permission, error) {
	var model models.PermissionModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	resource, _ := vo.NewResource(model.Resource)
	action, _ := vo.NewAction(model.Action)

	return permission.ReconstructPermission(
		model.ID,
		resource,
		action,
		model.Description,
		model.CreatedAt,
		model.UpdatedAt,
	)
}

func (r *PermissionRepositoryImpl) GetByCode(ctx context.Context, resource, action string) (*permission.Permission, error) {
	var model models.PermissionModel
	if err := r.db.WithContext(ctx).Where("resource = ? AND action = ?", resource, action).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	res, _ := vo.NewResource(model.Resource)
	act, _ := vo.NewAction(model.Action)

	return permission.ReconstructPermission(
		model.ID,
		res,
		act,
		model.Description,
		model.CreatedAt,
		model.UpdatedAt,
	)
}

func (r *PermissionRepositoryImpl) List(ctx context.Context, filter permission.PermissionFilter) ([]*permission.Permission, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.PermissionModel{})

	if filter.Resource != "" {
		query = query.Where("resource = ?", filter.Resource)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count permissions: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = constants.DefaultPage
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxPageSize {
		pageSize = constants.MaxPageSize
	}

	offset := (page - 1) * pageSize
	query = query.Offset(offset).Limit(pageSize).Order("resource, action")

	var permModels []*models.PermissionModel
	if err := query.Find(&permModels).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list permissions: %w", err)
	}

	permissions := make([]*permission.Permission, 0, len(permModels))
	for _, model := range permModels {
		resource, _ := vo.NewResource(model.Resource)
		action, _ := vo.NewAction(model.Action)

		perm, err := permission.ReconstructPermission(
			model.ID,
			resource,
			action,
			model.Description,
			model.CreatedAt,
			model.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to reconstruct permission: %w", err)
		}
		permissions = append(permissions, perm)
	}

	return permissions, total, nil
}

func (r *PermissionRepositoryImpl) Update(ctx context.Context, perm *permission.Permission) error {
	result := r.db.WithContext(ctx).Model(&models.PermissionModel{}).
		Where("id = ?", perm.ID()).
		Updates(map[string]interface{}{
			"description": perm.Description(),
			"updated_at":  perm.UpdatedAt(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update permission: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("permission not found")
	}

	return nil
}

func (r *PermissionRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.PermissionModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete permission: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("permission not found")
	}
	return nil
}

func (r *PermissionRepositoryImpl) Exists(ctx context.Context, id uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.PermissionModel{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check permission existence: %w", err)
	}
	return count > 0, nil
}

func (r *PermissionRepositoryImpl) GetByIDs(ctx context.Context, ids []uint) ([]*permission.Permission, error) {
	var permModels []*models.PermissionModel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&permModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get permissions by IDs: %w", err)
	}

	permissions := make([]*permission.Permission, 0, len(permModels))
	for _, model := range permModels {
		resource, _ := vo.NewResource(model.Resource)
		action, _ := vo.NewAction(model.Action)

		perm, err := permission.ReconstructPermission(
			model.ID,
			resource,
			action,
			model.Description,
			model.CreatedAt,
			model.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to reconstruct permission: %w", err)
		}
		permissions = append(permissions, perm)
	}

	return permissions, nil
}
