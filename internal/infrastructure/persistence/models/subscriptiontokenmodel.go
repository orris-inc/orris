package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// SubscriptionTokenModel represents the database persistence model for subscription tokens
// This is the anti-corruption layer between domain and database
type SubscriptionTokenModel struct {
	ID             uint       `gorm:"primarykey"`
	SubscriptionID uint       `gorm:"not null;index:idx_subscription_token"`
	Name           string     `gorm:"not null;size:100"`
	TokenHash      string     `gorm:"uniqueIndex;not null;size:64"` // SHA256 hash
	Prefix         string     `gorm:"not null;size:20"`
	Scope          string     `gorm:"not null;size:20"`
	ExpiresAt      *time.Time `gorm:"index:idx_expires_at"`
	LastUsedAt     *time.Time
	LastUsedIP     *string `gorm:"size:45"` // IPv6 max length
	UsageCount     uint64  `gorm:"default:0"`
	IsActive       bool    `gorm:"default:true;index:idx_active"`
	CreatedAt      time.Time
	RevokedAt      *time.Time
}

// TableName specifies the table name for GORM
func (SubscriptionTokenModel) TableName() string {
	return constants.TableSubscriptionTokens
}

// BeforeCreate hook for GORM
func (st *SubscriptionTokenModel) BeforeCreate(tx *gorm.DB) error {
	if !st.IsActive {
		st.IsActive = true
	}
	if st.UsageCount == 0 {
		st.UsageCount = 0
	}
	return nil
}
