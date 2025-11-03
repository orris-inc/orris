package permission

import (
	"context"
	"fmt"

	"orris/internal/domain/permission"
	"orris/internal/shared/logger"
)

type Service struct {
	roleRepo       permission.RoleRepository
	permissionRepo permission.PermissionRepository
	enforcer       permission.PermissionEnforcer
	logger         logger.Interface
}

func NewService(
	roleRepo permission.RoleRepository,
	permissionRepo permission.PermissionRepository,
	enforcer permission.PermissionEnforcer,
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

func (s *Service) GrantPermissionsToUser(ctx context.Context, userID uint, permissionNames []string) error {
	s.logger.Infow("granting permissions to user", "user_id", userID, "permissions", permissionNames)

	for _, permName := range permissionNames {
		resource := extractResource(permName)
		action := extractAction(permName)

		perm, err := s.permissionRepo.GetByCode(ctx, resource, action)
		if err != nil {
			s.logger.Warnw("permission not found, skipping", "permission", permName, "error", err)
			continue
		}
		if perm == nil {
			s.logger.Warnw("permission not found, skipping", "permission", permName)
			continue
		}

		if err := s.enforcer.AddPolicy(
			fmt.Sprintf("user:%d", userID),
			resource,
			action,
		); err != nil {
			s.logger.Errorw("failed to add permission to enforcer", "error", err, "permission", permName)
			return fmt.Errorf("failed to grant permission %s: %w", permName, err)
		}
	}

	s.logger.Infow("permissions granted successfully", "user_id", userID, "count", len(permissionNames))
	return nil
}

func (s *Service) RevokePermissionsFromUser(ctx context.Context, userID uint, permissionNames []string) error {
	s.logger.Infow("revoking permissions from user", "user_id", userID, "permissions", permissionNames)

	for _, permName := range permissionNames {
		resource := extractResource(permName)
		action := extractAction(permName)

		perm, err := s.permissionRepo.GetByCode(ctx, resource, action)
		if err != nil {
			continue
		}
		if perm == nil {
			continue
		}

		if err := s.enforcer.RemovePolicy(
			fmt.Sprintf("user:%d", userID),
			resource,
			action,
		); err != nil {
			s.logger.Errorw("failed to remove permission from enforcer", "error", err)
		}
	}

	s.logger.Infow("permissions revoked successfully", "user_id", userID, "count", len(permissionNames))
	return nil
}

func extractResource(permissionName string) string {
	for i := 0; i < len(permissionName); i++ {
		if permissionName[i] == ':' {
			return permissionName[:i]
		}
	}
	return permissionName
}

func extractAction(permissionName string) string {
	for i := 0; i < len(permissionName); i++ {
		if permissionName[i] == ':' {
			return permissionName[i+1:]
		}
	}
	return ""
}
