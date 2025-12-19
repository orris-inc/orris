package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// SubscriptionUsageModel represents the database persistence model for subscription usage statistics
// This is the anti-corruption layer between domain and database
type SubscriptionUsageModel struct {
	ID             uint      `gorm:"primarykey"`
	SID            string    `gorm:"column:sid;uniqueIndex;not null;size:50;comment:Stripe-style ID: usage_xxx"`
	SubscriptionID *uint     `gorm:"index:idx_subscription"`
	ResourceType   string    `gorm:"column:resource_type;not null;default:'node';size:50;index:idx_resource,priority:1"`
	ResourceID     uint      `gorm:"column:resource_id;not null;default:0;index:idx_resource,priority:2"`
	Upload         uint64    `gorm:"not null;default:0"`                            // bytes uploaded
	Download       uint64    `gorm:"not null;default:0"`                            // bytes downloaded
	Total          uint64    `gorm:"not null;default:0"`                            // total bytes (upload + download)
	Period         time.Time `gorm:"not null;index:idx_resource_period,priority:2"` // time period for this statistic (hourly/daily)
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// Note: No foreign key constraints or associations.
	// All relationships are managed by application business logic.
}

// TableName specifies the table name for GORM
func (SubscriptionUsageModel) TableName() string {
	return constants.TableSubscriptionUsages
}

// BeforeCreate hook for GORM
func (t *SubscriptionUsageModel) BeforeCreate(tx *gorm.DB) error {
	// Automatically calculate total if not set
	if t.Total == 0 {
		t.Total = t.Upload + t.Download
	}
	return nil
}

// BeforeUpdate hook for GORM
func (t *SubscriptionUsageModel) BeforeUpdate(tx *gorm.DB) error {
	// Automatically update total
	t.Total = t.Upload + t.Download
	return nil
}
