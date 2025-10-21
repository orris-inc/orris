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

type RoleRepositoryImpl struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) permission.RoleRepository {
	return &RoleRepositoryImpl{db: db}
}

func (r *RoleRepositoryImpl) Create(ctx context.Context, role *permission.Role) error {
	model := &models.RoleModel{
		Name:        role.Name(),
		Slug:        role.Slug(),
		Description: role.Description(),
		Status:      string(role.Status()),
		IsSystem:    role.IsSystem(),
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return role.SetID(model.ID)
}

func (r *RoleRepositoryImpl) GetByID(ctx context.Context, id uint) (*permission.Role, error) {
	var model models.RoleModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return permission.ReconstructRole(
		model.ID,
		model.Name,
		model.Slug,
		model.Description,
		permission.RoleStatus(model.Status),
		model.IsSystem,
		model.CreatedAt,
		model.UpdatedAt,
	)
}

func (r *RoleRepositoryImpl) GetBySlug(ctx context.Context, slug string) (*permission.Role, error) {
	var model models.RoleModel
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get role by slug: %w", err)
	}

	return permission.ReconstructRole(
		model.ID,
		model.Name,
		model.Slug,
		model.Description,
		permission.RoleStatus(model.Status),
		model.IsSystem,
		model.CreatedAt,
		model.UpdatedAt,
	)
}

func (r *RoleRepositoryImpl) List(ctx context.Context, filter permission.RoleFilter) ([]*permission.Role, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.RoleModel{})

	if filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+filter.Name+"%")
	}
	if filter.Slug != "" {
		query = query.Where("slug = ?", filter.Slug)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count roles: %w", err)
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
	query = query.Offset(offset).Limit(pageSize).Order("created_at DESC")

	var roleModels []*models.RoleModel
	if err := query.Find(&roleModels).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list roles: %w", err)
	}

	roles := make([]*permission.Role, 0, len(roleModels))
	for _, model := range roleModels {
		role, err := permission.ReconstructRole(
			model.ID,
			model.Name,
			model.Slug,
			model.Description,
			permission.RoleStatus(model.Status),
			model.IsSystem,
			model.CreatedAt,
			model.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to reconstruct role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, total, nil
}

func (r *RoleRepositoryImpl) Update(ctx context.Context, role *permission.Role) error {
	result := r.db.WithContext(ctx).Model(&models.RoleModel{}).
		Where("id = ?", role.ID()).
		Updates(map[string]interface{}{
			"name":        role.Name(),
			"description": role.Description(),
			"status":      string(role.Status()),
			"updated_at":  role.UpdatedAt(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update role: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("role not found")
	}

	return nil
}

func (r *RoleRepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.RoleModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete role: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("role not found")
	}
	return nil
}

func (r *RoleRepositoryImpl) Exists(ctx context.Context, id uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.RoleModel{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check role existence: %w", err)
	}
	return count > 0, nil
}

func (r *RoleRepositoryImpl) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.RoleModel{}).Where("slug = ?", slug).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check role slug existence: %w", err)
	}
	return count > 0, nil
}

func (r *RoleRepositoryImpl) AssignPermissions(ctx context.Context, roleID uint, permissionIDs []uint) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	rolePermissions := make([]models.RolePermissionModel, 0, len(permissionIDs))
	for _, permID := range permissionIDs {
		rolePermissions = append(rolePermissions, models.RolePermissionModel{
			RoleID:       roleID,
			PermissionID: permID,
		})
	}

	return r.db.WithContext(ctx).
		Create(&rolePermissions).Error
}

func (r *RoleRepositoryImpl) RemovePermissions(ctx context.Context, roleID uint, permissionIDs []uint) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Where("role_id = ? AND permission_id IN ?", roleID, permissionIDs).
		Delete(&models.RolePermissionModel{}).Error
}

func (r *RoleRepositoryImpl) GetPermissions(ctx context.Context, roleID uint) ([]*permission.Permission, error) {
	var permModels []*models.PermissionModel
	err := r.db.WithContext(ctx).
		Table(constants.TablePermissions).
		Joins("INNER JOIN "+constants.TableRolePermissions+" ON "+constants.TablePermissions+".id = "+constants.TableRolePermissions+".permission_id").
		Where(constants.TableRolePermissions+".role_id = ?", roleID).
		Find(&permModels).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	return r.permissionModelsToEntities(permModels)
}

func (r *RoleRepositoryImpl) AssignToUser(ctx context.Context, userID uint, roleIDs []uint) error {
	if len(roleIDs) == 0 {
		return nil
	}

	userRoles := make([]models.UserRoleModel, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		userRoles = append(userRoles, models.UserRoleModel{
			UserID: userID,
			RoleID: roleID,
		})
	}

	return r.db.WithContext(ctx).
		Create(&userRoles).Error
}

func (r *RoleRepositoryImpl) RemoveFromUser(ctx context.Context, userID uint, roleIDs []uint) error {
	if len(roleIDs) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Where("user_id = ? AND role_id IN ?", userID, roleIDs).
		Delete(&models.UserRoleModel{}).Error
}

func (r *RoleRepositoryImpl) GetUserRoles(ctx context.Context, userID uint) ([]*permission.Role, error) {
	var roleModels []*models.RoleModel
	err := r.db.WithContext(ctx).
		Table(constants.TableRoles).
		Joins("INNER JOIN "+constants.TableUserRoles+" ON "+constants.TableRoles+".id = "+constants.TableUserRoles+".role_id").
		Where(constants.TableUserRoles+".user_id = ?", userID).
		Find(&roleModels).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	roles := make([]*permission.Role, 0, len(roleModels))
	for _, model := range roleModels {
		role, err := permission.ReconstructRole(
			model.ID,
			model.Name,
			model.Slug,
			model.Description,
			permission.RoleStatus(model.Status),
			model.IsSystem,
			model.CreatedAt,
			model.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to reconstruct role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

func (r *RoleRepositoryImpl) permissionModelsToEntities(models []*models.PermissionModel) ([]*permission.Permission, error) {
	permissions := make([]*permission.Permission, 0, len(models))
	for _, model := range models {
		resource, err := vo.NewResource(model.Resource)
		if err != nil {
			return nil, fmt.Errorf("invalid resource: %w", err)
		}

		action, err := vo.NewAction(model.Action)
		if err != nil {
			return nil, fmt.Errorf("invalid action: %w", err)
		}

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
