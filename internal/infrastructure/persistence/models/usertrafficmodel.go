package models

import (
	"time"

	"gorm.io/gorm"

	"orris/internal/shared/constants"
)

// UserTrafficModel represents the database persistence model for user traffic statistics
// This tracks user-level traffic across nodes for quota management
type UserTrafficModel struct {
	ID             uint      `gorm:"primarykey"`
	UserID         uint      `gorm:"not null;index:idx_user_node_period;index:idx_user_period"`
	NodeID         uint      `gorm:"not null;index:idx_user_node_period;index:idx_node_period"`
	SubscriptionID *uint     `gorm:"index:idx_subscription"`
	Upload         uint64    `gorm:"not null;default:0"` // bytes uploaded
	Download       uint64    `gorm:"not null;default:0"` // bytes downloaded
	Total          uint64    `gorm:"not null;default:0"` // total bytes (upload + download)
	Period         time.Time `gorm:"not null;index:idx_user_node_period;index:idx_user_period;index:idx_node_period"` // time period for this statistic
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// Foreign keys
	User *UserModel `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Node *NodeModel `gorm:"foreignKey:NodeID;constraint:OnDelete:CASCADE"`
	// Subscription foreign key will reference subscriptions table
}

// TableName specifies the table name for GORM
func (UserTrafficModel) TableName() string {
	return constants.TableUserTraffic
}

// BeforeCreate hook for GORM
func (t *UserTrafficModel) BeforeCreate(tx *gorm.DB) error {
	// Automatically calculate total if not set
	if t.Total == 0 {
		t.Total = t.Upload + t.Download
	}
	return nil
}

// BeforeUpdate hook for GORM
func (t *UserTrafficModel) BeforeUpdate(tx *gorm.DB) error {
	// Automatically update total
	t.Total = t.Upload + t.Download
	return nil
}
