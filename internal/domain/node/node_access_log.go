package node

import (
	"fmt"
	"time"
)

// NodeAccessLog represents an access log entry for a node
type NodeAccessLog struct {
	id                uint
	nodeID            uint
	userID            uint
	subscriptionID    uint
	subscriptionToken string
	clientIP          string
	userAgent         string
	connectTime       time.Time
	disconnectTime    *time.Time
	duration          int64
	upload            uint64
	download          uint64
	createdAt         time.Time
}

// NewNodeAccessLog creates a new node access log entry
func NewNodeAccessLog(
	nodeID, userID, subscriptionID uint,
	subscriptionToken, clientIP, userAgent string,
	connectTime time.Time,
) (*NodeAccessLog, error) {
	if nodeID == 0 {
		return nil, fmt.Errorf("node ID is required")
	}
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if subscriptionID == 0 {
		return nil, fmt.Errorf("subscription ID is required")
	}
	if subscriptionToken == "" {
		return nil, fmt.Errorf("subscription token is required")
	}
	if clientIP == "" {
		return nil, fmt.Errorf("client IP is required")
	}

	return &NodeAccessLog{
		nodeID:            nodeID,
		userID:            userID,
		subscriptionID:    subscriptionID,
		subscriptionToken: subscriptionToken,
		clientIP:          clientIP,
		userAgent:         userAgent,
		connectTime:       connectTime,
		disconnectTime:    nil,
		duration:          0,
		upload:            0,
		download:          0,
		createdAt:         time.Now(),
	}, nil
}

// ReconstructNodeAccessLog reconstructs a node access log from persistence
func ReconstructNodeAccessLog(
	id, nodeID, userID, subscriptionID uint,
	subscriptionToken, clientIP, userAgent string,
	connectTime time.Time,
	disconnectTime *time.Time,
	duration int64,
	upload, download uint64,
	createdAt time.Time,
) (*NodeAccessLog, error) {
	if id == 0 {
		return nil, fmt.Errorf("node access log ID cannot be zero")
	}
	if nodeID == 0 {
		return nil, fmt.Errorf("node ID is required")
	}
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if subscriptionID == 0 {
		return nil, fmt.Errorf("subscription ID is required")
	}

	return &NodeAccessLog{
		id:                id,
		nodeID:            nodeID,
		userID:            userID,
		subscriptionID:    subscriptionID,
		subscriptionToken: subscriptionToken,
		clientIP:          clientIP,
		userAgent:         userAgent,
		connectTime:       connectTime,
		disconnectTime:    disconnectTime,
		duration:          duration,
		upload:            upload,
		download:          download,
		createdAt:         createdAt,
	}, nil
}

// ID returns the access log ID
func (nal *NodeAccessLog) ID() uint {
	return nal.id
}

// NodeID returns the node ID
func (nal *NodeAccessLog) NodeID() uint {
	return nal.nodeID
}

// UserID returns the user ID
func (nal *NodeAccessLog) UserID() uint {
	return nal.userID
}

// SubscriptionID returns the subscription ID
func (nal *NodeAccessLog) SubscriptionID() uint {
	return nal.subscriptionID
}

// SubscriptionToken returns the subscription token
func (nal *NodeAccessLog) SubscriptionToken() string {
	return nal.subscriptionToken
}

// ClientIP returns the client IP address
func (nal *NodeAccessLog) ClientIP() string {
	return nal.clientIP
}

// UserAgent returns the user agent string
func (nal *NodeAccessLog) UserAgent() string {
	return nal.userAgent
}

// ConnectTime returns when the connection was established
func (nal *NodeAccessLog) ConnectTime() time.Time {
	return nal.connectTime
}

// DisconnectTime returns when the connection was terminated
func (nal *NodeAccessLog) DisconnectTime() *time.Time {
	return nal.disconnectTime
}

// Duration returns the connection duration in seconds
func (nal *NodeAccessLog) Duration() int64 {
	return nal.duration
}

// Upload returns the upload traffic in bytes
func (nal *NodeAccessLog) Upload() uint64 {
	return nal.upload
}

// Download returns the download traffic in bytes
func (nal *NodeAccessLog) Download() uint64 {
	return nal.download
}

// CreatedAt returns when the access log was created
func (nal *NodeAccessLog) CreatedAt() time.Time {
	return nal.createdAt
}

// SetID sets the access log ID (only for persistence layer use)
func (nal *NodeAccessLog) SetID(id uint) error {
	if nal.id != 0 {
		return fmt.Errorf("node access log ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("node access log ID cannot be zero")
	}
	nal.id = id
	return nil
}

// RecordDisconnect records the disconnect time and calculates duration
func (nal *NodeAccessLog) RecordDisconnect(disconnectTime time.Time, upload, download uint64) error {
	if disconnectTime.Before(nal.connectTime) {
		return fmt.Errorf("disconnect time cannot be before connect time")
	}

	nal.disconnectTime = &disconnectTime
	nal.duration = int64(disconnectTime.Sub(nal.connectTime).Seconds())
	nal.upload = upload
	nal.download = download

	return nil
}

// IsActive checks if the connection is still active
func (nal *NodeAccessLog) IsActive() bool {
	return nal.disconnectTime == nil
}

// TotalTraffic returns the total traffic (upload + download)
func (nal *NodeAccessLog) TotalTraffic() uint64 {
	return nal.upload + nal.download
}

// GetMaskedToken returns a masked version of the subscription token
func (nal *NodeAccessLog) GetMaskedToken() string {
	if len(nal.subscriptionToken) <= 8 {
		return "****"
	}
	return nal.subscriptionToken[:4] + "****" + nal.subscriptionToken[len(nal.subscriptionToken)-4:]
}

// Validate performs domain-level validation
func (nal *NodeAccessLog) Validate() error {
	if nal.nodeID == 0 {
		return fmt.Errorf("node ID is required")
	}
	if nal.userID == 0 {
		return fmt.Errorf("user ID is required")
	}
	if nal.subscriptionID == 0 {
		return fmt.Errorf("subscription ID is required")
	}
	if nal.subscriptionToken == "" {
		return fmt.Errorf("subscription token is required")
	}
	if nal.clientIP == "" {
		return fmt.Errorf("client IP is required")
	}
	if nal.disconnectTime != nil && nal.disconnectTime.Before(nal.connectTime) {
		return fmt.Errorf("disconnect time cannot be before connect time")
	}
	return nil
}
