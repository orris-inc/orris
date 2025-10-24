package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SubscriptionModel represents the database persistence model for subscriptions
// This is the anti-corruption layer between domain and database
type SubscriptionModel struct {
	ID                 uint      `gorm:"primarykey"`
	UserID             uint      `gorm:"not null;index:idx_user_subscription"`
	PlanID             uint      `gorm:"not null;index:idx_plan_subscription"`
	Status             string    `gorm:"not null;size:20;index:idx_status"`
	StartDate          time.Time `gorm:"not null"`
	EndDate            time.Time `gorm:"not null;index:idx_end_date"`
	AutoRenew          bool      `gorm:"default:false"`
	CurrentPeriodStart time.Time `gorm:"not null"`
	CurrentPeriodEnd   time.Time `gorm:"not null"`
	CancelledAt        *time.Time
	CancelReason       *string `gorm:"size:500"`
	Metadata           datatypes.JSON
	Version            int `gorm:"not null;default:1"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (SubscriptionModel) TableName() string {
	return "subscriptions"
}

// BeforeCreate hook for GORM
func (s *SubscriptionModel) BeforeCreate(tx *gorm.DB) error {
	if s.Version == 0 {
		s.Version = 1
	}
	return nil
}

// BeforeUpdate implements optimistic locking
func (s *SubscriptionModel) BeforeUpdate(tx *gorm.DB) error {
	// Increment version for optimistic locking
	tx.Statement.SetColumn("version", s.Version+1)
	return nil
}
