package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SubscriptionPlanModel represents the database persistence model for subscription plans
// This is the anti-corruption layer between domain and database
type SubscriptionPlanModel struct {
	ID             uint   `gorm:"primarykey"`
	Name           string `gorm:"not null;size:100"`
	Slug           string `gorm:"uniqueIndex;not null;size:50"`
	Description    string `gorm:"size:500"`
	Price          uint64 `gorm:"not null"`
	Currency       string `gorm:"not null;size:3;default:CNY"`
	BillingCycle   string `gorm:"not null;size:20"`
	TrialDays      int    `gorm:"default:0"`
	Status         string `gorm:"not null;size:20;default:active"`
	Features       datatypes.JSON
	Limits         datatypes.JSON
	CustomEndpoint string `gorm:"size:200"`
	APIRateLimit   uint   `gorm:"default:60"`
	MaxUsers       uint   `gorm:"default:1"`
	MaxProjects    uint   `gorm:"default:1"`
	StorageLimit   uint64 `gorm:"default:1073741824"` // 1GB in bytes
	IsPublic       bool   `gorm:"default:true"`
	SortOrder      int    `gorm:"default:0"`
	Metadata       datatypes.JSON
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (SubscriptionPlanModel) TableName() string {
	return "subscription_plans"
}

// BeforeCreate hook for GORM
func (sp *SubscriptionPlanModel) BeforeCreate(tx *gorm.DB) error {
	if sp.Status == "" {
		sp.Status = "active"
	}
	if sp.Currency == "" {
		sp.Currency = "CNY"
	}
	return nil
}
