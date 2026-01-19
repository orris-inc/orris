package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// ExternalForwardRuleModel represents the database persistence model for external forward rules.
type ExternalForwardRuleModel struct {
	ID             uint           `gorm:"primarykey"`
	SID            string         `gorm:"column:sid;not null;size:50;uniqueIndex:idx_external_forward_rule_sid"`
	SubscriptionID *uint          `gorm:"index:idx_external_forward_rules_subscription_id"`
	UserID         *uint          `gorm:"index:idx_external_forward_rules_user_id"`
	NodeID         *uint          `gorm:"column:node_id;index:idx_external_forward_rules_node_id"`
	Name           string         `gorm:"not null;size:100"`
	ServerAddress  string         `gorm:"not null;size:255"`
	ListenPort     uint16         `gorm:"not null"`
	ExternalSource string         `gorm:"not null;size:50"`
	ExternalRuleID string         `gorm:"size:100"`
	Status         string         `gorm:"not null;default:enabled;size:20"`
	SortOrder      int            `gorm:"not null;default:0"`
	Remark         string         `gorm:"size:500"`
	GroupIDs       datatypes.JSON `gorm:"column:group_ids"` // resource group IDs (JSON array)
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM.
func (ExternalForwardRuleModel) TableName() string {
	return constants.TableExternalForwardRules
}

// BeforeCreate hook for GORM.
func (m *ExternalForwardRuleModel) BeforeCreate(tx *gorm.DB) error {
	if m.Status == "" {
		m.Status = "enabled"
	}
	return nil
}
