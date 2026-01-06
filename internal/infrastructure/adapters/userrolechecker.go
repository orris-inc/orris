package adapters

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/user"
)

// UserRoleCheckerAdapter adapts user.Repository to adminnotification.UserRoleChecker interface
type UserRoleCheckerAdapter struct {
	userRepo user.Repository
}

// NewUserRoleCheckerAdapter creates a new UserRoleCheckerAdapter
func NewUserRoleCheckerAdapter(userRepo user.Repository) *UserRoleCheckerAdapter {
	return &UserRoleCheckerAdapter{userRepo: userRepo}
}

// IsAdmin checks if the user has admin role
func (a *UserRoleCheckerAdapter) IsAdmin(ctx context.Context, userID uint) (bool, error) {
	u, err := a.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if u == nil {
		return false, nil
	}
	return u.Role().IsAdmin(), nil
}
