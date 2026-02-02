package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// ForwardRuleModel represents the database persistence model for forward rules.
type ForwardRuleModel struct {
	ID                  uint           `gorm:"primarykey"`
	SID                 string         `gorm:"column:sid;not null;size:20;uniqueIndex:idx_forward_rule_sid"` // Stripe-style prefixed ID (fr_xxx)
	AgentID             uint           `gorm:"not null;index:idx_forward_agent_id;uniqueIndex:idx_listen_port_agent_server"`
	UserID              *uint          `gorm:"index:idx_forward_rules_user_id;index:idx_forward_rules_user_status"` // user ID for user-owned rules (nullable)
	SubscriptionID      *uint          `gorm:"column:subscription_id;index:idx_forward_rules_subscription_id"`      // subscription ID for subscription-bound rules (nullable)
	RuleType            string         `gorm:"not null;default:direct;size:20"`                                     // direct, chain, direct_chain, websocket
	ExitAgentID         *uint          `gorm:"index:idx_forward_exit_agent_id"`                                     // exit agent ID for chain/websocket forward (nullable)
	ExitAgents          datatypes.JSON `gorm:"type:json;default:null"`                                              // multiple exit agents with weights for load balancing (JSON array)
	LoadBalanceStrategy string         `gorm:"column:load_balance_strategy;not null;default:failover;size:32"`      // load balance strategy: failover, weighted
	ChainAgentIDs       datatypes.JSON `gorm:"type:json;default:null"`                                              // ordered array of intermediate agent IDs for chain forwarding
	ChainPortConfig     datatypes.JSON `gorm:"type:json;default:null"`                                              // map of agent_id -> listen_port for direct_chain type or hybrid chain direct hops
	TunnelHops          *int           `gorm:"column:tunnel_hops"`                                                  // number of hops using tunnel (nil=full tunnel, N=first N hops use tunnel)
	TunnelType          string         `gorm:"not null;default:ws;size:10"`                                         // tunnel type: ws or tls
	Name              string         `gorm:"not null;size:100;index:idx_forward_name"`
	ListenPort        uint16         `gorm:"not null;uniqueIndex:idx_listen_port_agent_server"`
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
	SortOrder         int            `gorm:"not null;default:0"`
	GroupIDs          datatypes.JSON `gorm:"column:group_ids"` // resource group IDs (JSON array)
	// External rule fields (used when RuleType = 'external')
	ServerAddress  *string `gorm:"column:server_address;size:255;uniqueIndex:idx_listen_port_agent_server"` // server address for external rules
	ExternalSource *string `gorm:"column:external_source;size:50"`                                          // external source identifier
	ExternalRuleID *string `gorm:"column:external_rule_id;size:100;index:idx_forward_rules_external"`       // external rule reference ID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
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
	if m.LoadBalanceStrategy == "" {
		m.LoadBalanceStrategy = "failover"
	}
	return nil
}
