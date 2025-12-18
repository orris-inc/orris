package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/constants"
)

// PlanModel represents the database persistence model for subscription plans
// This is the anti-corruption layer between domain and database
type PlanModel struct {
	ID           uint   `gorm:"primarykey"`
	Name         string `gorm:"not null;size:100"`
	Slug         string `gorm:"uniqueIndex;not null;size:50"`
	PlanType     string `gorm:"not null;size:20;default:node"`
	Description  string `gorm:"size:500"`
	Price        uint64 `gorm:"not null"`
	Currency     string `gorm:"not null;size:3"`
	BillingCycle string `gorm:"not null;size:20"`
	TrialDays    int    `gorm:"default:0"`
	Status       string `gorm:"not null;size:20;default:active"`
	Features     datatypes.JSON
	Limits       datatypes.JSON
	APIRateLimit uint `gorm:"default:60"`
	MaxUsers     uint `gorm:"default:1"`
	MaxProjects  uint `gorm:"default:1"`
	IsPublic     bool `gorm:"default:true"`
	SortOrder    int  `gorm:"default:0"`
	Metadata     datatypes.JSON
	Version      int `gorm:"not null;default:1"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (PlanModel) TableName() string {
	return constants.TablePlans
}

// BeforeCreate hook for GORM
func (p *PlanModel) BeforeCreate(tx *gorm.DB) error {
	if p.Status == "" {
		p.Status = string(subscription.PlanStatusActive)
	}
	if p.Currency == "" {
		p.Currency = constants.DefaultCurrency
	}
	return nil
}
