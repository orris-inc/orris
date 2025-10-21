package permission

import (
	"context"
	"fmt"

	"orris/internal/domain/permission"
	permissionInfra "orris/internal/infrastructure/permission"
	"orris/internal/shared/logger"
)

type Service struct {
	roleRepo       permission.RoleRepository
	permissionRepo permission.PermissionRepository
	enforcer       *permissionInfra.Enforcer
	logger         logger.Interface
}

func NewService(
	roleRepo permission.RoleRepository,
	permissionRepo permission.PermissionRepository,
	enforcer *permissionInfra.Enforcer,
	logger logger.Interface,
) *Service {
	return &Service{
		roleRepo:       roleRepo,
		permissionRepo: permissionRepo,
		enforcer:       enforcer,
		logger:         logger,
	}
}

func (s *Service) CheckPermission(ctx context.Context, userID uint, resource, action string) (bool, error) {
	return s.enforcer.Enforce(fmt.Sprintf("%d", userID), resource, action)
}

func (s *Service) AssignRoleToUser(ctx context.Context, userID uint, roleIDs []uint) error {
	if err := s.roleRepo.AssignToUser(ctx, userID, roleIDs); err != nil {
		return fmt.Errorf("failed to assign roles to user: %w", err)
	}

	for _, roleID := range roleIDs {
		role, err := s.roleRepo.GetByID(ctx, roleID)
		if err != nil {
			return fmt.Errorf("failed to get role: %w", err)
		}
		if role == nil {
			continue
		}

		if err := s.enforcer.AddRoleForUser(fmt.Sprintf("%d", userID), role.Slug()); err != nil {
			s.logger.Errorw("failed to add role to enforcer", "error", err)
		}
	}

	return nil
}

func (s *Service) GetUserRoles(ctx context.Context, userID uint) ([]*permission.Role, error) {
	return s.roleRepo.GetUserRoles(ctx, userID)
}

func (s *Service) GetUserPermissions(ctx context.Context, userID uint) ([]*permission.Permission, error) {
	roles, err := s.roleRepo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	permissionMap := make(map[uint]*permission.Permission)
	for _, role := range roles {
		permissions, err := s.roleRepo.GetPermissions(ctx, role.ID())
		if err != nil {
			return nil, fmt.Errorf("failed to get role permissions: %w", err)
		}

		for _, perm := range permissions {
			permissionMap[perm.ID()] = perm
		}
	}

	permissions := make([]*permission.Permission, 0, len(permissionMap))
	for _, perm := range permissionMap {
		permissions = append(permissions, perm)
	}

	return permissions, nil
}
