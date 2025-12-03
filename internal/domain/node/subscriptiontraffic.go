package node

import (
	"fmt"
	"time"
)

// SubscriptionTraffic represents traffic statistics entity for a subscription
type SubscriptionTraffic struct {
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

// NewSubscriptionTraffic creates a new subscription traffic record
func NewSubscriptionTraffic(nodeID uint, subscriptionID, userID *uint, period time.Time) (*SubscriptionTraffic, error) {
	if nodeID == 0 {
		return nil, fmt.Errorf("node ID is required")
	}

	now := time.Now()
	return &SubscriptionTraffic{
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

// ReconstructSubscriptionTraffic reconstructs a subscription traffic entity from persistence
func ReconstructSubscriptionTraffic(
	id, nodeID uint,
	userID, subscriptionID *uint,
	upload, download, total uint64,
	period, createdAt, updatedAt time.Time,
) (*SubscriptionTraffic, error) {
	if id == 0 {
		return nil, fmt.Errorf("subscription traffic ID cannot be zero")
	}
	if nodeID == 0 {
		return nil, fmt.Errorf("node ID is required")
	}

	return &SubscriptionTraffic{
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
func (st *SubscriptionTraffic) ID() uint {
	return st.id
}

// NodeID returns the node ID
func (st *SubscriptionTraffic) NodeID() uint {
	return st.nodeID
}

// UserID returns the user ID
func (st *SubscriptionTraffic) UserID() *uint {
	return st.userID
}

// SubscriptionID returns the subscription ID
func (st *SubscriptionTraffic) SubscriptionID() *uint {
	return st.subscriptionID
}

// Upload returns the upload traffic in bytes
func (st *SubscriptionTraffic) Upload() uint64 {
	return st.upload
}

// Download returns the download traffic in bytes
func (st *SubscriptionTraffic) Download() uint64 {
	return st.download
}

// Total returns the total traffic in bytes
func (st *SubscriptionTraffic) Total() uint64 {
	return st.total
}

// Period returns the period timestamp
func (st *SubscriptionTraffic) Period() time.Time {
	return st.period
}

// CreatedAt returns when the traffic record was created
func (st *SubscriptionTraffic) CreatedAt() time.Time {
	return st.createdAt
}

// UpdatedAt returns when the traffic record was last updated
func (st *SubscriptionTraffic) UpdatedAt() time.Time {
	return st.updatedAt
}

// SetID sets the traffic record ID (only for persistence layer use)
func (st *SubscriptionTraffic) SetID(id uint) error {
	if st.id != 0 {
		return fmt.Errorf("subscription traffic ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("subscription traffic ID cannot be zero")
	}
	st.id = id
	return nil
}

// Accumulate adds upload and download traffic to the current record
func (st *SubscriptionTraffic) Accumulate(upload, download uint64) error {
	if upload == 0 && download == 0 {
		return nil
	}

	st.upload += upload
	st.download += download
	st.total = st.upload + st.download
	st.updatedAt = time.Now()

	return nil
}

// TotalTraffic returns the total traffic (upload + download)
func (st *SubscriptionTraffic) TotalTraffic() uint64 {
	return st.total
}

// UploadRatio calculates the ratio of upload traffic to total traffic
func (st *SubscriptionTraffic) UploadRatio() float64 {
	if st.total == 0 {
		return 0.0
	}
	return float64(st.upload) / float64(st.total)
}

// DownloadRatio calculates the ratio of download traffic to total traffic
func (st *SubscriptionTraffic) DownloadRatio() float64 {
	if st.total == 0 {
		return 0.0
	}
	return float64(st.download) / float64(st.total)
}

// IsEmpty checks if the traffic record has no data
func (st *SubscriptionTraffic) IsEmpty() bool {
	return st.upload == 0 && st.download == 0
}

// Reset resets all traffic counters to zero
func (st *SubscriptionTraffic) Reset() error {
	st.upload = 0
	st.download = 0
	st.total = 0
	st.updatedAt = time.Now()
	return nil
}

// Validate performs domain-level validation
func (st *SubscriptionTraffic) Validate() error {
	if st.nodeID == 0 {
		return fmt.Errorf("node ID is required")
	}
	if st.total != st.upload+st.download {
		return fmt.Errorf("total traffic must equal upload + download")
	}
	return nil
}
