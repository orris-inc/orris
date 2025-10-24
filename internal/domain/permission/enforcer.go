package permission

type PermissionEnforcer interface {
	Enforce(userID string, resource string, action string) (bool, error)
	AddPolicy(role string, resource string, action string) error
	RemovePolicy(role string, resource string, action string) error
	AddRoleForUser(userID string, role string) error
	DeleteRoleForUser(userID string, role string) error
	GetRolesForUser(userID string) ([]string, error)
	GetPermissionsForUser(userID string) ([][]string, error)
	LoadPolicy() error
}
