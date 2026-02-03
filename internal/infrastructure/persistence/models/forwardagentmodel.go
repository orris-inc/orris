package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// ForwardAgentModel represents the database persistence model for forward agents.
type ForwardAgentModel struct {
	ID               uint           `gorm:"primarykey"`
	SID              string         `gorm:"column:sid;not null;size:20;uniqueIndex:idx_forward_agent_sid"` // Stripe-style prefixed ID (fa_xxx)
	Name             string         `gorm:"not null;size:100;index:idx_forward_agent_name"`
	TokenHash        string         `gorm:"not null;size:64;index:idx_forward_agent_token_hash"`
	APIToken         string         `gorm:"column:api_token;size:255"` // stored token for retrieval
	PublicAddress    string         `gorm:"size:255"`                  // public address for agent access (nullable)
	TunnelAddress    string         `gorm:"size:255"`                  // tunnel address for entry to connect to exit (nullable, overrides public_address)
	Status           string         `gorm:"not null;default:enabled;size:20;index:idx_forward_agent_status"`
	Remark           string         `gorm:"size:500"`
	GroupIDs         datatypes.JSON `gorm:"column:group_ids"` // resource group IDs (JSON array)
	AgentVersion     string         `gorm:"size:50"`          // agent software version (e.g., "1.2.3")
	Platform         string         `gorm:"size:20"`          // OS platform (linux, darwin, windows)
	Arch             string         `gorm:"size:20"`          // CPU architecture (amd64, arm64, arm, 386)
	AllowedPortRange *string        `gorm:"column:allowed_port_range;type:text"`
	BlockedProtocols datatypes.JSON `gorm:"column:blocked_protocols;type:json"` // protocols blocked by this agent
	SortOrder        int            `gorm:"not null;default:0"`
	MuteNotification bool           `gorm:"not null;default:false"` // mute online/offline notifications
	LastSeenAt       *time.Time
	ExpiresAt        *time.Time `gorm:"column:expires_at"`                        // expiration time (null = never expires)
	RenewalAmount    *float64   `gorm:"column:renewal_amount;type:decimal(10,2)"` // renewal amount for display
	CreatedAt        time.Time
	UpdatedAt        time.Time
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
