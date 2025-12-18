package auth

import "github.com/orris-inc/orris/internal/shared/constants"

// IsAdmin checks if the user has admin role
func IsAdmin(roles []string) bool {
	for _, role := range roles {
		if role == constants.RoleAdmin {
			return true
		}
	}
	return false
}

// IsSupportAgent checks if the user has support agent role
func IsSupportAgent(roles []string) bool {
	for _, role := range roles {
		if role == constants.RoleSupportAgent {
			return true
		}
	}
	return false
}

// IsAdminOrAgent checks if the user is admin or support agent
func IsAdminOrAgent(roles []string) bool {
	return IsAdmin(roles) || IsSupportAgent(roles)
}

// HasRole checks if the user has a specific role
func HasRole(roles []string, targetRole string) bool {
	for _, role := range roles {
		if role == targetRole {
			return true
		}
	}
	return false
}
