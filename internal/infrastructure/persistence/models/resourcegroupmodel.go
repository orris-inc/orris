package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// ResourceGroupModel represents the database persistence model for resource groups.
type ResourceGroupModel struct {
	ID          uint           `gorm:"primarykey"`
	SID         string         `gorm:"not null;size:32;uniqueIndex:idx_resource_group_sid"` // Stripe-style ID: rg_xxxxxxxx
	Name        string         `gorm:"not null;size:100;index:idx_resource_group_name"`
	PlanID      uint           `gorm:"not null;index:idx_resource_group_plan_id"`
	Description string         `gorm:"size:500"`
	Status      string         `gorm:"not null;default:active;size:20;index:idx_resource_group_status"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Version     int            `gorm:"not null;default:1"`
}

// TableName specifies the table name for GORM.
func (ResourceGroupModel) TableName() string {
	return constants.TableResourceGroups
}

// BeforeCreate hook for GORM.
func (m *ResourceGroupModel) BeforeCreate(tx *gorm.DB) error {
	if m.Status == "" {
		m.Status = "active"
	}
	if m.Version == 0 {
		m.Version = 1
	}
	return nil
}
