package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// ForwardChainModel represents the database persistence model for forward chains.
type ForwardChainModel struct {
	ID            uint   `gorm:"primarykey"`
	Name          string `gorm:"not null;size:100;uniqueIndex:idx_chain_name"`
	Protocol      string `gorm:"not null;size:10;index:idx_chain_protocol"` // tcp, udp, both
	Status        string `gorm:"not null;default:disabled;size:20;index:idx_chain_status"`
	TargetAddress string `gorm:"not null;size:255"`
	TargetPort    uint16 `gorm:"not null"`
	Remark        string `gorm:"size:500"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`

	// Relations
	Nodes []ForwardChainNodeModel `gorm:"foreignKey:ChainID"`
}

// TableName specifies the table name for GORM.
func (ForwardChainModel) TableName() string {
	return constants.TableForwardChains
}

// BeforeCreate hook for GORM.
func (m *ForwardChainModel) BeforeCreate(tx *gorm.DB) error {
	if m.Status == "" {
		m.Status = "disabled"
	}
	if m.Protocol == "" {
		m.Protocol = "tcp"
	}
	return nil
}

// ForwardChainNodeModel represents a node in the forward chain.
type ForwardChainNodeModel struct {
	ID         uint   `gorm:"primarykey"`
	ChainID    uint   `gorm:"not null;index:idx_chain_node_chain_id"`
	AgentID    uint   `gorm:"not null;index:idx_chain_node_agent_id"`
	ListenPort uint16 `gorm:"not null"`
	Sequence   int    `gorm:"not null"` // order in the chain
	CreatedAt  time.Time
}

// TableName specifies the table name for GORM.
func (ForwardChainNodeModel) TableName() string {
	return constants.TableForwardChainNodes
}

// ForwardChainRuleModel represents the association between chain and rules.
type ForwardChainRuleModel struct {
	ID        uint `gorm:"primarykey"`
	ChainID   uint `gorm:"not null;index:idx_chain_rule_chain_id"`
	RuleID    uint `gorm:"not null;index:idx_chain_rule_rule_id"`
	CreatedAt time.Time
}

// TableName specifies the table name for GORM.
func (ForwardChainRuleModel) TableName() string {
	return constants.TableForwardChainRules
}
