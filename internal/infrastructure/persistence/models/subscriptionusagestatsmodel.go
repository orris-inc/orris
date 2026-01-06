package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// SubscriptionUsageStatsModel represents the database persistence model for aggregated subscription usage statistics
// This is the anti-corruption layer between domain and database
type SubscriptionUsageStatsModel struct {
	ID             uint      `gorm:"primarykey"`
	SID            string    `gorm:"column:sid;uniqueIndex;not null;size:50;comment:Stripe-style ID: usagestat_xxx"`
	SubscriptionID *uint     `gorm:"index:idx_subscription_period,priority:1"`
	ResourceType   string    `gorm:"column:resource_type;not null;default:'node';size:50;index:idx_resource_period,priority:1"`
	ResourceID     uint      `gorm:"column:resource_id;not null;default:0;index:idx_resource_period,priority:2"`
	Upload         uint64    `gorm:"not null;default:0"` // bytes uploaded
	Download       uint64    `gorm:"not null;default:0"` // bytes downloaded
	Total          uint64    `gorm:"not null;default:0"` // total bytes (upload + download)
	Granularity    string    `gorm:"column:granularity;not null;size:10;index:idx_subscription_period,priority:2;index:idx_resource_period,priority:3;comment:daily or monthly"`
	Period         time.Time `gorm:"column:period;not null;type:date;index:idx_subscription_period,priority:3;index:idx_resource_period,priority:4;comment:date for daily, first day of month for monthly"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// Note: No foreign key constraints or associations.
	// All relationships are managed by application business logic.
}

// TableName specifies the table name for GORM
func (SubscriptionUsageStatsModel) TableName() string {
	return constants.TableSubscriptionUsageStats
}

// BeforeCreate hook for GORM
func (m *SubscriptionUsageStatsModel) BeforeCreate(tx *gorm.DB) error {
	// Automatically calculate total if not set
	if m.Total == 0 {
		m.Total = m.Upload + m.Download
	}
	return nil
}

// BeforeUpdate hook for GORM
func (m *SubscriptionUsageStatsModel) BeforeUpdate(tx *gorm.DB) error {
	// Automatically update total
	m.Total = m.Upload + m.Download
	return nil
}
