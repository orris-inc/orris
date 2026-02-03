package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/shared/constants"
)

// NodeModel represents the database persistence model for nodes
// This is the anti-corruption layer between domain and database
// Note: Protocol-specific configs are now stored in separate tables:
// - node_shadowsocks_configs for Shadowsocks protocol (encryption_method, plugin, plugin_opts)
// - node_trojan_configs for Trojan protocol
type NodeModel struct {
	ID                uint           `gorm:"primarykey"`
	SID               string         `gorm:"uniqueIndex;size:50;column:sid"` // Stripe-style prefixed ID (node_xxx)
	Name              string         `gorm:"uniqueIndex;not null;size:100"`
	ServerAddress     string         `gorm:"not null;size:255;index:idx_agent_address"`
	AgentPort         uint16         `gorm:"not null;index:idx_agent_address"`                                     // port for agent connections
	SubscriptionPort  *uint16        `gorm:"default:null"`                                                         // port for client subscriptions (if nil, use AgentPort)
	Protocol          string         `gorm:"not null;default:shadowsocks;size:20;index:idx_protocol"`              // shadowsocks, trojan
	Status            string         `gorm:"not null;default:inactive;size:20;index:idx_status"`                   // active, inactive, maintenance
	GroupIDs          datatypes.JSON `gorm:"column:group_ids"`                                                     // resource group IDs (JSON array)
	UserID            *uint          `gorm:"index:idx_nodes_user_id;comment:Owner user ID (NULL = admin created)"` // owner user ID
	Region            *string        `gorm:"size:100"`
	Tags              datatypes.JSON
	SortOrder         int            `gorm:"not null;default:0"`
	MuteNotification  bool           `gorm:"not null;default:false"` // mute online/offline notifications
	MaintenanceReason *string        `gorm:"size:500"`
	RouteConfig       datatypes.JSON `gorm:"column:route_config"`                          // routing configuration for traffic splitting (JSON)
	TokenHash         string         `gorm:"not null;uniqueIndex:idx_token_hash;size:255"` // hashed API token for node authentication
	APIToken          string         `gorm:"column:api_token;size:255"`                    // stored token for retrieval
	LastSeenAt        *time.Time     `gorm:"index:idx_nodes_last_seen_at"`                 // last time the node agent reported status
	PublicIPv4        *string        `gorm:"size:15"`                                      // public IPv4 address reported by agent
	PublicIPv6        *string        `gorm:"size:45"`                                      // public IPv6 address reported by agent
	AgentVersion      *string        `gorm:"size:50"`                                      // agent software version (e.g., "1.2.3")
	Platform          *string        `gorm:"size:20"`                                      // OS platform (linux, darwin, windows)
	Arch              *string        `gorm:"size:20"`                                      // CPU architecture (amd64, arm64, arm, 386)
	ExpiresAt         *time.Time     `gorm:"column:expires_at"`                            // expiration time (null = never expires)
	RenewalAmount     *float64       `gorm:"column:renewal_amount;type:decimal(10,2)"`     // renewal amount for display
	Version           int            `gorm:"not null;default:1"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TableName specifies the table name for GORM
func (NodeModel) TableName() string {
	return constants.TableNodes
}

// BeforeCreate hook for GORM
func (n *NodeModel) BeforeCreate(tx *gorm.DB) error {
	if n.Status == "" {
		n.Status = string(valueobjects.NodeStatusInactive)
	}
	if n.Protocol == "" {
		n.Protocol = "shadowsocks"
	}
	if n.Version == 0 {
		n.Version = 1
	}
	return nil
}
