package models

import (
	"time"

	"gorm.io/datatypes"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// EntitlementModel represents the database persistence model for entitlements
// This is the anti-corruption layer between domain and database
// It manages access rights for subjects (users) to resources (nodes, forward agents, features)
type EntitlementModel struct {
	ID           uint       `gorm:"primarykey"`
	SubjectType  string     `gorm:"not null;size:20;uniqueIndex:idx_unique_entitlement,priority:1;index:idx_subject,priority:1"`
	SubjectID    uint       `gorm:"not null;uniqueIndex:idx_unique_entitlement,priority:2;index:idx_subject,priority:2"`
	ResourceType string     `gorm:"not null;size:30;uniqueIndex:idx_unique_entitlement,priority:3;index:idx_resource,priority:1"`
	ResourceID   uint       `gorm:"not null;uniqueIndex:idx_unique_entitlement,priority:4;index:idx_resource,priority:2"`
	SourceType   string     `gorm:"not null;size:20;uniqueIndex:idx_unique_entitlement,priority:5;index:idx_source,priority:1"`
	SourceID     uint       `gorm:"not null;uniqueIndex:idx_unique_entitlement,priority:6;index:idx_source,priority:2"`
	Status       string     `gorm:"not null;size:20;default:active;index:idx_status_expires,priority:1"`
	ExpiresAt    *time.Time `gorm:"index:idx_status_expires,priority:2"`
	Metadata     datatypes.JSON
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Version      int `gorm:"not null;default:1"`
}

// TableName specifies the table name for GORM
func (EntitlementModel) TableName() string {
	return constants.TableEntitlements
}
