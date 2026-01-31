package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/constants"
)

// VLESSConfigModel represents the database persistence model for VLESS protocol configuration
// Separated from NodeModel to follow protocol-specific table pattern
type VLESSConfigModel struct {
	ID            uint   `gorm:"primarykey"`
	NodeID        uint   `gorm:"uniqueIndex;not null"`          // Logical foreign key to nodes table
	TransportType string `gorm:"not null;size:10;default:tcp"`  // tcp, ws, grpc, h2
	Flow          string `gorm:"size:32"`                       // xtls-rprx-vision or empty
	Security      string `gorm:"not null;size:16;default:none"` // none, tls, reality
	SNI           string `gorm:"size:255"`                      // TLS Server Name Indication
	Fingerprint   string `gorm:"size:64"`                       // TLS fingerprint (chrome, firefox, safari, etc.)
	AllowInsecure bool   `gorm:"not null;default:false"`        // Allow insecure TLS connection
	Host          string `gorm:"size:255"`                      // WebSocket/H2 host header
	Path          string `gorm:"size:255"`                      // WebSocket/H2 path
	ServiceName   string `gorm:"size:255"`                      // gRPC service name
	PrivateKey    string `gorm:"size:255"`                      // Reality private key (for server inbound)
	PublicKey     string `gorm:"size:255"`                      // Reality public key (for client outbound)
	ShortID       string `gorm:"size:32"`                       // Reality short ID
	SpiderX       string `gorm:"size:255"`                      // Reality spider X parameter
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (VLESSConfigModel) TableName() string {
	return constants.TableNodeVLESSConfigs
}

// BeforeCreate hook for GORM
func (v *VLESSConfigModel) BeforeCreate(tx *gorm.DB) error {
	if v.TransportType == "" {
		v.TransportType = "tcp"
	}
	if v.Security == "" {
		v.Security = "none"
	}
	return nil
}
