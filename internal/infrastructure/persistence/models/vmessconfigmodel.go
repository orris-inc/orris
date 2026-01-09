package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// VMessConfigModel represents the database persistence model for VMess protocol configuration
// Separated from NodeModel to follow protocol-specific table pattern
type VMessConfigModel struct {
	ID            uint   `gorm:"primarykey"`
	NodeID        uint   `gorm:"uniqueIndex;not null"`          // Logical foreign key to nodes table
	AlterID       int    `gorm:"not null;default:0"`            // Alter ID (usually 0 for modern clients)
	Security      string `gorm:"not null;size:32;default:auto"` // auto, aes-128-gcm, chacha20-poly1305, none, zero
	TransportType string `gorm:"not null;size:10;default:tcp"`  // tcp, ws, grpc, http, quic
	Host          string `gorm:"size:255"`                      // WebSocket/HTTP host header
	Path          string `gorm:"size:255"`                      // WebSocket/HTTP path
	ServiceName   string `gorm:"size:255"`                      // gRPC service name
	TLS           bool   `gorm:"not null;default:false"`        // Enable TLS
	SNI           string `gorm:"size:255"`                      // TLS Server Name Indication
	AllowInsecure bool   `gorm:"not null;default:true"`         // Allow insecure TLS connection
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (VMessConfigModel) TableName() string {
	return constants.TableNodeVMessConfigs
}

// BeforeCreate hook for GORM
func (v *VMessConfigModel) BeforeCreate(tx *gorm.DB) error {
	if v.Security == "" {
		v.Security = "auto"
	}
	if v.TransportType == "" {
		v.TransportType = "tcp"
	}
	return nil
}
