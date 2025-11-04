package authorization

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

func (r UserRole) String() string {
	return string(r)
}

func (r UserRole) IsAdmin() bool {
	return r == RoleAdmin
}

func (r UserRole) IsValid() bool {
	return r == RoleAdmin || r == RoleUser
}

func ParseUserRole(s string) UserRole {
	role := UserRole(s)
	if role.IsValid() {
		return role
	}
	return RoleUser
}
