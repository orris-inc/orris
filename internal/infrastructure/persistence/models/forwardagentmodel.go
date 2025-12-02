package models

import (
	"time"

	"gorm.io/gorm"

	"orris/internal/shared/constants"
)

// ForwardAgentModel represents the database persistence model for forward agents.
type ForwardAgentModel struct {
	ID        uint   `gorm:"primarykey"`
	Name      string `gorm:"not null;size:100;index:idx_forward_agent_name"`
	TokenHash string `gorm:"not null;size:64;index:idx_forward_agent_token_hash"`
	Status    string `gorm:"not null;default:enabled;size:20;index:idx_forward_agent_status"`
	Remark    string `gorm:"size:500"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
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
