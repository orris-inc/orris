package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// ForwardAgentModel represents the database persistence model for forward agents.
type ForwardAgentModel struct {
	ID            uint   `gorm:"primarykey"`
	SID           string `gorm:"column:sid;not null;size:20;uniqueIndex:idx_forward_agent_sid"` // Stripe-style prefixed ID (fa_xxx)
	Name          string `gorm:"not null;size:100;index:idx_forward_agent_name"`
	TokenHash     string `gorm:"not null;size:64;index:idx_forward_agent_token_hash"`
	APIToken      string `gorm:"column:api_token;size:255"` // stored token for retrieval
	PublicAddress string `gorm:"size:255"`                  // public address for agent access (nullable)
	TunnelAddress string `gorm:"size:255"`                  // tunnel address for entry to connect to exit (nullable, overrides public_address)
	Status        string `gorm:"not null;default:enabled;size:20;index:idx_forward_agent_status"`
	Remark        string `gorm:"size:500"`
	GroupID       *uint  `gorm:"index:idx_forward_agent_group_id"` // resource group ID
	LastSeenAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM.
func (ForwardAgentModel) TableName() string {
	return constants.TableForwardAgents
}

// BeforeCreate hook for GORM.
func (m *ForwardAgentModel) BeforeCreate(tx *gorm.DB) error {
	if m.Status == "" {
		m.Status = "enabled"
	}
	return nil
}
