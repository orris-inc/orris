package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SubscriptionHistoryModel represents the database persistence model for subscription history
// This is the anti-corruption layer between domain and database
type SubscriptionHistoryModel struct {
	ID             uint           `gorm:"primarykey"`
	SubscriptionID uint           `gorm:"not null;index:idx_subscription_history"`
	UserID         uint           `gorm:"not null;index:idx_user_history"`
	PlanID         uint           `gorm:"not null"`
	Action         string         `gorm:"not null;size:50;index:idx_action"` // created, renewed, upgraded, downgraded, cancelled, expired
	OldStatus      *string        `gorm:"size:20"`
	NewStatus      string         `gorm:"not null;size:20"`
	OldPlanID      *uint
	NewPlanID      *uint
	Amount         *uint64
	Currency       *string `gorm:"size:3"`
	Reason         *string `gorm:"size:500"`
	PerformedBy    *uint   // User ID who performed the action (for admin actions)
	IPAddress      *string `gorm:"size:45"`
	UserAgent      *string `gorm:"size:255"`
	Metadata       datatypes.JSON
	CreatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (SubscriptionHistoryModel) TableName() string {
	return "subscription_histories"
}

// BeforeCreate hook for GORM
func (sh *SubscriptionHistoryModel) BeforeCreate(tx *gorm.DB) error {
	if sh.CreatedAt.IsZero() {
		sh.CreatedAt = time.Now()
	}
	return nil
}
