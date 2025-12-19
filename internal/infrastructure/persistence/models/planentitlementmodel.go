package models

import (
	"time"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// PlanEntitlementModel represents the database persistence model for plan entitlements
// This links subscription plans to various resource types (nodes, forward_agents, etc.)
type PlanEntitlementModel struct {
	ID           uint   `gorm:"primarykey"`
	PlanID       uint   `gorm:"not null;index:idx_plan_id;uniqueIndex:uk_plan_resource,priority:1"`
	ResourceType string `gorm:"not null;size:50;uniqueIndex:uk_plan_resource,priority:2;index:idx_resource,priority:1"`
	ResourceID   uint   `gorm:"not null;uniqueIndex:uk_plan_resource,priority:3;index:idx_resource,priority:2"`
	CreatedAt    time.Time

	// Note: No foreign key constraints.
	// All relationships are managed by application business logic.
}

// TableName specifies the table name for GORM
func (PlanEntitlementModel) TableName() string {
	return constants.TablePlanEntitlements
}
