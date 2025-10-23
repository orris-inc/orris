package node

import (
	"fmt"
	"time"
)

// NodeTraffic represents traffic statistics entity for a node
type NodeTraffic struct {
	id             uint
	nodeID         uint
	userID         *uint
	subscriptionID *uint
	upload         uint64
	download       uint64
	total          uint64
	period         time.Time
	createdAt      time.Time
	updatedAt      time.Time
}

// NewNodeTraffic creates a new node traffic record
func NewNodeTraffic(nodeID uint, userID, subscriptionID *uint, period time.Time) (*NodeTraffic, error) {
	if nodeID == 0 {
		return nil, fmt.Errorf("node ID is required")
	}

	now := time.Now()
	return &NodeTraffic{
		nodeID:         nodeID,
		userID:         userID,
		subscriptionID: subscriptionID,
		upload:         0,
		download:       0,
		total:          0,
		period:         period,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// ReconstructNodeTraffic reconstructs a node traffic entity from persistence
func ReconstructNodeTraffic(
	id, nodeID uint,
	userID, subscriptionID *uint,
	upload, download, total uint64,
	period, createdAt, updatedAt time.Time,
) (*NodeTraffic, error) {
	if id == 0 {
		return nil, fmt.Errorf("node traffic ID cannot be zero")
	}
	if nodeID == 0 {
		return nil, fmt.Errorf("node ID is required")
	}

	return &NodeTraffic{
		id:             id,
		nodeID:         nodeID,
		userID:         userID,
		subscriptionID: subscriptionID,
		upload:         upload,
		download:       download,
		total:          total,
		period:         period,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}, nil
}

// ID returns the traffic record ID
func (nt *NodeTraffic) ID() uint {
	return nt.id
}

// NodeID returns the node ID
func (nt *NodeTraffic) NodeID() uint {
	return nt.nodeID
}

// UserID returns the user ID
func (nt *NodeTraffic) UserID() *uint {
	return nt.userID
}

// SubscriptionID returns the subscription ID
func (nt *NodeTraffic) SubscriptionID() *uint {
	return nt.subscriptionID
}

// Upload returns the upload traffic in bytes
func (nt *NodeTraffic) Upload() uint64 {
	return nt.upload
}

// Download returns the download traffic in bytes
func (nt *NodeTraffic) Download() uint64 {
	return nt.download
}

// Total returns the total traffic in bytes
func (nt *NodeTraffic) Total() uint64 {
	return nt.total
}

// Period returns the period timestamp
func (nt *NodeTraffic) Period() time.Time {
	return nt.period
}

// CreatedAt returns when the traffic record was created
func (nt *NodeTraffic) CreatedAt() time.Time {
	return nt.createdAt
}

// UpdatedAt returns when the traffic record was last updated
func (nt *NodeTraffic) UpdatedAt() time.Time {
	return nt.updatedAt
}

// SetID sets the traffic record ID (only for persistence layer use)
func (nt *NodeTraffic) SetID(id uint) error {
	if nt.id != 0 {
		return fmt.Errorf("node traffic ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("node traffic ID cannot be zero")
	}
	nt.id = id
	return nil
}

// Accumulate adds upload and download traffic to the current record
func (nt *NodeTraffic) Accumulate(upload, download uint64) error {
	if upload == 0 && download == 0 {
		return nil
	}

	nt.upload += upload
	nt.download += download
	nt.total = nt.upload + nt.download
	nt.updatedAt = time.Now()

	return nil
}

// TotalTraffic returns the total traffic (upload + download)
func (nt *NodeTraffic) TotalTraffic() uint64 {
	return nt.total
}

// UploadRatio calculates the ratio of upload traffic to total traffic
func (nt *NodeTraffic) UploadRatio() float64 {
	if nt.total == 0 {
		return 0.0
	}
	return float64(nt.upload) / float64(nt.total)
}

// DownloadRatio calculates the ratio of download traffic to total traffic
func (nt *NodeTraffic) DownloadRatio() float64 {
	if nt.total == 0 {
		return 0.0
	}
	return float64(nt.download) / float64(nt.total)
}

// IsEmpty checks if the traffic record has no data
func (nt *NodeTraffic) IsEmpty() bool {
	return nt.upload == 0 && nt.download == 0
}

// Reset resets all traffic counters to zero
func (nt *NodeTraffic) Reset() error {
	nt.upload = 0
	nt.download = 0
	nt.total = 0
	nt.updatedAt = time.Now()
	return nil
}

// Validate performs domain-level validation
func (nt *NodeTraffic) Validate() error {
	if nt.nodeID == 0 {
		return fmt.Errorf("node ID is required")
	}
	if nt.total != nt.upload+nt.download {
		return fmt.Errorf("total traffic must equal upload + download")
	}
	return nil
}
