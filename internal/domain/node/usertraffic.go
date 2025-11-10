package node

import (
	"fmt"
	"time"
)

// UserTraffic represents user-level traffic statistics entity
// This tracks traffic usage per user per node for quota management
type UserTraffic struct {
	id             uint
	userID         uint
	nodeID         uint
	subscriptionID *uint
	upload         uint64
	download       uint64
	total          uint64
	period         time.Time
	createdAt      time.Time
	updatedAt      time.Time
}

// NewUserTraffic creates a new user traffic record
func NewUserTraffic(userID, nodeID uint, subscriptionID *uint, period time.Time) (*UserTraffic, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if nodeID == 0 {
		return nil, fmt.Errorf("node ID is required")
	}

	now := time.Now()
	return &UserTraffic{
		userID:         userID,
		nodeID:         nodeID,
		subscriptionID: subscriptionID,
		upload:         0,
		download:       0,
		total:          0,
		period:         period,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// ReconstructUserTraffic reconstructs a user traffic entity from persistence
func ReconstructUserTraffic(
	id, userID, nodeID uint,
	subscriptionID *uint,
	upload, download, total uint64,
	period, createdAt, updatedAt time.Time,
) (*UserTraffic, error) {
	if id == 0 {
		return nil, fmt.Errorf("user traffic ID cannot be zero")
	}
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if nodeID == 0 {
		return nil, fmt.Errorf("node ID is required")
	}

	return &UserTraffic{
		id:             id,
		userID:         userID,
		nodeID:         nodeID,
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
func (ut *UserTraffic) ID() uint {
	return ut.id
}

// UserID returns the user ID
func (ut *UserTraffic) UserID() uint {
	return ut.userID
}

// NodeID returns the node ID
func (ut *UserTraffic) NodeID() uint {
	return ut.nodeID
}

// SubscriptionID returns the subscription ID
func (ut *UserTraffic) SubscriptionID() *uint {
	return ut.subscriptionID
}

// Upload returns the upload traffic in bytes
func (ut *UserTraffic) Upload() uint64 {
	return ut.upload
}

// Download returns the download traffic in bytes
func (ut *UserTraffic) Download() uint64 {
	return ut.download
}

// Total returns the total traffic in bytes
func (ut *UserTraffic) Total() uint64 {
	return ut.total
}

// Period returns the period timestamp
func (ut *UserTraffic) Period() time.Time {
	return ut.period
}

// CreatedAt returns when the traffic record was created
func (ut *UserTraffic) CreatedAt() time.Time {
	return ut.createdAt
}

// UpdatedAt returns when the traffic record was last updated
func (ut *UserTraffic) UpdatedAt() time.Time {
	return ut.updatedAt
}

// SetID sets the traffic record ID (only for persistence layer use)
func (ut *UserTraffic) SetID(id uint) error {
	if ut.id != 0 {
		return fmt.Errorf("user traffic ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("user traffic ID cannot be zero")
	}
	ut.id = id
	return nil
}

// Accumulate adds upload and download traffic to the current record
func (ut *UserTraffic) Accumulate(upload, download uint64) error {
	if upload == 0 && download == 0 {
		return nil
	}

	ut.upload += upload
	ut.download += download
	ut.total = ut.upload + ut.download
	ut.updatedAt = time.Now()

	return nil
}

// TotalTraffic returns the total traffic (upload + download)
func (ut *UserTraffic) TotalTraffic() uint64 {
	return ut.total
}

// UploadRatio calculates the ratio of upload traffic to total traffic
func (ut *UserTraffic) UploadRatio() float64 {
	if ut.total == 0 {
		return 0.0
	}
	return float64(ut.upload) / float64(ut.total)
}

// DownloadRatio calculates the ratio of download traffic to total traffic
func (ut *UserTraffic) DownloadRatio() float64 {
	if ut.total == 0 {
		return 0.0
	}
	return float64(ut.download) / float64(ut.total)
}

// IsEmpty checks if the traffic record has no data
func (ut *UserTraffic) IsEmpty() bool {
	return ut.upload == 0 && ut.download == 0
}

// Reset resets all traffic counters to zero
func (ut *UserTraffic) Reset() error {
	ut.upload = 0
	ut.download = 0
	ut.total = 0
	ut.updatedAt = time.Now()
	return nil
}

// Validate performs domain-level validation
func (ut *UserTraffic) Validate() error {
	if ut.userID == 0 {
		return fmt.Errorf("user ID is required")
	}
	if ut.nodeID == 0 {
		return fmt.Errorf("node ID is required")
	}
	if ut.total != ut.upload+ut.download {
		return fmt.Errorf("total traffic must equal upload + download")
	}
	return nil
}
