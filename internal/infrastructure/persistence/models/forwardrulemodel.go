package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// ForwardRuleModel represents the database persistence model for forward rules.
type ForwardRuleModel struct {
	ID                uint           `gorm:"primarykey"`
	ShortID           string         `gorm:"not null;size:16;uniqueIndex:idx_forward_rule_short_id"` // external API identifier
	AgentID           uint           `gorm:"not null;index:idx_forward_agent_id;uniqueIndex:idx_listen_port_agent"`
	UserID            *uint          `gorm:"index:idx_forward_rules_user_id;index:idx_forward_rules_user_status"` // user ID for user-owned rules (nullable)
	PlanIDs           datatypes.JSON `gorm:"column:plan_ids;default:'[]'"`                                        // JSON array of subscription plan IDs
	RuleType          string         `gorm:"not null;default:direct;size:20"`                                     // direct, chain, direct_chain, websocket
	ExitAgentID       *uint          `gorm:"index:idx_forward_exit_agent_id"`                                     // exit agent ID for chain/websocket forward (nullable)
	ChainAgentIDs     datatypes.JSON `gorm:"type:json;default:null"`                                              // ordered array of intermediate agent IDs for chain forwarding
	ChainPortConfig   datatypes.JSON `gorm:"type:json;default:null"`                                              // map of agent_id -> listen_port for direct_chain type
	Name              string         `gorm:"not null;size:100;index:idx_forward_name"`
	ListenPort        uint16         `gorm:"not null;uniqueIndex:idx_listen_port_agent"`
	TargetAddress     string         `gorm:"size:255"`                                    // required when RuleType=direct (if TargetNodeID is not set)
	TargetPort        uint16         `gorm:"default:0"`                                   // required when RuleType=direct (if TargetNodeID is not set)
	TargetNodeID      *uint          `gorm:"index:idx_forward_target_node_id"`            // target node ID for dynamic address resolution (mutually exclusive with TargetAddress/TargetPort)
	BindIP            string         `gorm:"size:45"`                                     // bind IP address for outbound connections (optional)
	IPVersion         string         `gorm:"not null;default:auto;size:10"`               // auto, ipv4, ipv6
	Protocol          string         `gorm:"not null;size:10;index:idx_forward_protocol"` // tcp, udp, both
	Status            string         `gorm:"not null;default:disabled;size:20;index:idx_forward_status"`
	Remark            string         `gorm:"size:500"`
	UploadBytes       int64          `gorm:"not null;default:0"`
	DownloadBytes     int64          `gorm:"not null;default:0"`
	TrafficMultiplier *float64       `gorm:"column:traffic_multiplier;type:decimal(10,4)"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
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
	if m.RuleType == "" {
		m.RuleType = "direct"
	}
	if m.IPVersion == "" {
		m.IPVersion = "auto"
	}
	return nil
}
