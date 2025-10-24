package models

import (
	"time"

	"orris/internal/shared/constants"
)

type RolePermissionModel struct {
	ID           uint `gorm:"primarykey"`
	RoleID       uint `gorm:"not null;uniqueIndex:idx_role_permission"`
	PermissionID uint `gorm:"not null;uniqueIndex:idx_role_permission"`
	CreatedAt    time.Time
}

func (RolePermissionModel) TableName() string {
	return constants.TableRolePermissions
}

type UserRoleModel struct {
	ID        uint `gorm:"primarykey"`
	UserID    uint `gorm:"not null;uniqueIndex:idx_user_role"`
	RoleID    uint `gorm:"not null;uniqueIndex:idx_user_role"`
	CreatedAt time.Time
}

func (UserRoleModel) TableName() string {
	return constants.TableUserRoles
}
