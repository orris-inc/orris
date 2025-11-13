package models

import (
	"time"

	"gorm.io/gorm"
)

// SubscriptionUsageModel represents the database persistence model for subscription usage tracking
// This is the anti-corruption layer between domain and database
type SubscriptionUsageModel struct {
	ID             uint      `gorm:"primarykey"`
	SubscriptionID uint      `gorm:"not null;uniqueIndex:idx_subscription_period"`
	PeriodStart    time.Time `gorm:"not null;uniqueIndex:idx_subscription_period"`
	PeriodEnd      time.Time `gorm:"not null;index:idx_period_end"`
	UsersCount     uint      `gorm:"not null;default:0"`
	LastResetAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (SubscriptionUsageModel) TableName() string {
	return "subscription_usages"
}

// BeforeCreate hook for GORM
func (su *SubscriptionUsageModel) BeforeCreate(tx *gorm.DB) error {
	return nil
}

// BeforeUpdate hook for GORM
func (su *SubscriptionUsageModel) BeforeUpdate(tx *gorm.DB) error {
	su.UpdatedAt = time.Now()
	return nil
}
