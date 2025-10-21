package models

import (
	"time"

	"orris/internal/shared/constants"
)

type RoleModel struct {
	ID          uint      `gorm:"primarykey"`
	Name        string    `gorm:"not null;size:50"`
	Slug        string    `gorm:"uniqueIndex;not null;size:50"`
	Description string    `gorm:"type:text"`
	Status      string    `gorm:"not null;default:active;size:20"`
	IsSystem    bool      `gorm:"default:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (RoleModel) TableName() string {
	return constants.TableRoles
}
