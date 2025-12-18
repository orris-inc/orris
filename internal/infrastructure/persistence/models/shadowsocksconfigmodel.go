package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// ShadowsocksConfigModel represents the database persistence model for Shadowsocks protocol configuration
// Separated from NodeModel to follow protocol-specific table pattern
type ShadowsocksConfigModel struct {
	ID               uint           `gorm:"primarykey"`
	NodeID           uint           `gorm:"uniqueIndex;not null"` // Logical foreign key to nodes table
	EncryptionMethod string         `gorm:"not null;size:50"`     // aes-256-gcm, aes-128-gcm, chacha20-ietf-poly1305
	Plugin           *string        `gorm:"size:100"`             // obfs-local, v2ray-plugin, etc.
	PluginOpts       datatypes.JSON `gorm:"type:json"`            // Plugin options as JSON
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (ShadowsocksConfigModel) TableName() string {
	return constants.TableNodeShadowsocksConfigs
}

// BeforeCreate hook for GORM
func (s *ShadowsocksConfigModel) BeforeCreate(tx *gorm.DB) error {
	if s.EncryptionMethod == "" {
		s.EncryptionMethod = "aes-256-gcm"
	}
	return nil
}
