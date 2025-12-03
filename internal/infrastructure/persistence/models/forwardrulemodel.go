package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// ForwardRuleModel represents the database persistence model for forward rules.
type ForwardRuleModel struct {
	ID            uint   `gorm:"primarykey"`
	AgentID       uint   `gorm:"not null;index:idx_forward_agent_id;uniqueIndex:idx_listen_port_agent"`
	NextAgentID   uint   `gorm:"not null;default:0;index:idx_forward_next_agent_id"` // 0=direct forward, >0=chain forward
	Name          string `gorm:"not null;size:100;index:idx_forward_name"`
	ListenPort    uint16 `gorm:"not null;uniqueIndex:idx_listen_port_agent"`
	TargetAddress string `gorm:"size:255"`                                    // required when NextAgentID=0
	TargetPort    uint16 `gorm:"default:0"`                                   // required when NextAgentID=0
	Protocol      string `gorm:"not null;size:10;index:idx_forward_protocol"` // tcp, udp, both
	Status        string `gorm:"not null;default:disabled;size:20;index:idx_forward_status"`
	Remark        string `gorm:"size:500"`
	UploadBytes   int64  `gorm:"not null;default:0"`
	DownloadBytes int64  `gorm:"not null;default:0"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM.
func (ForwardRuleModel) TableName() string {
	return constants.TableForwardRules
}

// BeforeCreate hook for GORM.
func (m *ForwardRuleModel) BeforeCreate(tx *gorm.DB) error {
	if m.Status == "" {
		m.Status = "disabled"
	}
	if m.Protocol == "" {
		m.Protocol = "tcp"
	}
	return nil
}
