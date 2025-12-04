package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// NodeModel represents the database persistence model for nodes
// This is the anti-corruption layer between domain and database
// Note: Protocol-specific configs are now stored in separate tables:
// - shadowsocks_configs for Shadowsocks protocol (encryption_method, plugin, plugin_opts)
// - trojan_configs for Trojan protocol
type NodeModel struct {
	ID                uint    `gorm:"primarykey"`
	Name              string  `gorm:"uniqueIndex;not null;size:100"`
	ServerAddress     string  `gorm:"not null;size:255;index:idx_server"`
	ServerPort        uint16  `gorm:"not null;index:idx_server"`
	Protocol          string  `gorm:"not null;default:shadowsocks;size:20;index:idx_protocol"` // shadowsocks, trojan
	Status            string  `gorm:"not null;default:inactive;size:20;index:idx_status"`      // active, inactive, maintenance
	Region            *string `gorm:"size:100"`
	Tags              datatypes.JSON
	SortOrder         int        `gorm:"not null;default:0"`
	MaintenanceReason *string    `gorm:"size:500"`
	TokenHash         string     `gorm:"not null;uniqueIndex:idx_token_hash;size:255"` // hashed API token for node authentication
	LastSeenAt        *time.Time `gorm:"index:idx_nodes_last_seen_at"`                 // last time the node agent reported status
	Version           int        `gorm:"not null;default:1"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (NodeModel) TableName() string {
	return constants.TableNodes
}

// BeforeCreate hook for GORM
func (n *NodeModel) BeforeCreate(tx *gorm.DB) error {
	if n.Status == "" {
		n.Status = "inactive"
	}
	if n.Protocol == "" {
		n.Protocol = "shadowsocks"
	}
	if n.Version == 0 {
		n.Version = 1
	}
	return nil
}

// BeforeUpdate implements optimistic locking
func (n *NodeModel) BeforeUpdate(tx *gorm.DB) error {
	// Increment version for optimistic locking
	tx.Statement.SetColumn("version", n.Version+1)
	return nil
}
