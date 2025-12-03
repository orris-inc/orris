package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// SubscriptionTrafficModel represents the database persistence model for subscription traffic statistics
// This is the anti-corruption layer between domain and database
type SubscriptionTrafficModel struct {
	ID             uint      `gorm:"primarykey"`
	NodeID         uint      `gorm:"not null;index:idx_node_period"`
	UserID         *uint     `gorm:"index:idx_user_period"`
	SubscriptionID *uint     `gorm:"index:idx_subscription"`
	Upload         uint64    `gorm:"not null;default:0"`                                   // bytes uploaded
	Download       uint64    `gorm:"not null;default:0"`                                   // bytes downloaded
	Total          uint64    `gorm:"not null;default:0"`                                   // total bytes (upload + download)
	Period         time.Time `gorm:"not null;index:idx_node_period;index:idx_user_period"` // time period for this statistic (hourly/daily)
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// Note: No foreign key constraints or associations.
	// All relationships are managed by application business logic.
}

// TableName specifies the table name for GORM
func (SubscriptionTrafficModel) TableName() string {
	return constants.TableSubscriptionTraffic
}

// BeforeCreate hook for GORM
func (t *SubscriptionTrafficModel) BeforeCreate(tx *gorm.DB) error {
	// Automatically calculate total if not set
	if t.Total == 0 {
		t.Total = t.Upload + t.Download
	}
	return nil
}

// BeforeUpdate hook for GORM
func (t *SubscriptionTrafficModel) BeforeUpdate(tx *gorm.DB) error {
	// Automatically update total
	t.Total = t.Upload + t.Download
	return nil
}
