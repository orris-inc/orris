package models

import (
	"time"

	"orris/internal/shared/constants"
)

type PermissionModel struct {
	ID          uint      `gorm:"primarykey"`
	Resource    string    `gorm:"not null;size:50;uniqueIndex:idx_resource_action"`
	Action      string    `gorm:"not null;size:20;uniqueIndex:idx_resource_action"`
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (PermissionModel) TableName() string {
	return constants.TablePermissions
}
