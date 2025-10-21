package permission

import "context"

type RoleRepository interface {
	Create(ctx context.Context, role *Role) error
	GetByID(ctx context.Context, id uint) (*Role, error)
	GetBySlug(ctx context.Context, slug string) (*Role, error)
	List(ctx context.Context, filter RoleFilter) ([]*Role, int64, error)
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uint) error
	Exists(ctx context.Context, id uint) (bool, error)
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	AssignPermissions(ctx context.Context, roleID uint, permissionIDs []uint) error
	RemovePermissions(ctx context.Context, roleID uint, permissionIDs []uint) error
	GetPermissions(ctx context.Context, roleID uint) ([]*Permission, error)

	AssignToUser(ctx context.Context, userID uint, roleIDs []uint) error
	RemoveFromUser(ctx context.Context, userID uint, roleIDs []uint) error
	GetUserRoles(ctx context.Context, userID uint) ([]*Role, error)
}

type PermissionRepository interface {
	Create(ctx context.Context, permission *Permission) error
	GetByID(ctx context.Context, id uint) (*Permission, error)
	GetByCode(ctx context.Context, resource, action string) (*Permission, error)
	List(ctx context.Context, filter PermissionFilter) ([]*Permission, int64, error)
	Update(ctx context.Context, permission *Permission) error
	Delete(ctx context.Context, id uint) error
	Exists(ctx context.Context, id uint) (bool, error)

	GetByIDs(ctx context.Context, ids []uint) ([]*Permission, error)
}

type RoleFilter struct {
	Name   string
	Slug   string
	Status RoleStatus
	Page   int
	PageSize int
}

type PermissionFilter struct {
	Resource string
	Action   string
	Page     int
	PageSize int
}
