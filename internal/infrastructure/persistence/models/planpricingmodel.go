package models

import (
	"time"

	"gorm.io/gorm"
)

// SubscriptionPlanPricingModel represents the subscription_plan_pricing table
type SubscriptionPlanPricingModel struct {
	ID           uint           `gorm:"primarykey"`
	PlanID       uint           `gorm:"not null;index:idx_plan_id;comment:Reference to subscription_plans table"`
	BillingCycle string         `gorm:"not null;size:20;index:idx_billing_cycle;comment:Billing cycle: weekly, monthly, quarterly, semi_annual, yearly, lifetime"`
	Price        uint64         `gorm:"not null;comment:Price in smallest currency unit (cents)"`
	Currency     string         `gorm:"not null;size:3;default:CNY;comment:Currency code: CNY, USD, EUR, GBP, JPY"`
	IsActive     bool           `gorm:"not null;default:true;index:idx_is_active;comment:Whether this pricing option is active"`
	CreatedAt    time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt    gorm.DeletedAt `gorm:"index:idx_deleted_at"`

	// Relationship
	Plan *SubscriptionPlanModel `gorm:"foreignKey:PlanID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
}

// TableName specifies the table name for GORM
func (SubscriptionPlanPricingModel) TableName() string {
	return "subscription_plan_pricing"
}

// BeforeCreate hook to set timestamps
func (m *SubscriptionPlanPricingModel) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (m *SubscriptionPlanPricingModel) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}
