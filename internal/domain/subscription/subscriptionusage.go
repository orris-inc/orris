package subscription

import (
	"fmt"
	"time"
)

// SubscriptionUsage represents usage statistics entity for a subscription
type SubscriptionUsage struct {
	id             uint
	resourceType   string
	resourceID     uint
	subscriptionID *uint
	upload         uint64
	download       uint64
	total          uint64
	period         time.Time
	createdAt      time.Time
	updatedAt      time.Time
}

// NewSubscriptionUsage creates a new subscription usage record
func NewSubscriptionUsage(resourceType string, resourceID uint, subscriptionID *uint, period time.Time) (*SubscriptionUsage, error) {
	if resourceType == "" {
		return nil, fmt.Errorf("resource type is required")
	}
	if resourceID == 0 {
		return nil, fmt.Errorf("resource ID is required")
	}

	now := time.Now()
	return &SubscriptionUsage{
		resourceType:   resourceType,
		resourceID:     resourceID,
		subscriptionID: subscriptionID,
		upload:         0,
		download:       0,
		total:          0,
		period:         period,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// ReconstructSubscriptionUsage reconstructs a subscription usage entity from persistence
func ReconstructSubscriptionUsage(
	id uint,
	resourceType string,
	resourceID uint,
	subscriptionID *uint,
	upload, download, total uint64,
	period, createdAt, updatedAt time.Time,
) (*SubscriptionUsage, error) {
	if id == 0 {
		return nil, fmt.Errorf("subscription usage ID cannot be zero")
	}
	if resourceType == "" {
		return nil, fmt.Errorf("resource type is required")
	}
	if resourceID == 0 {
		return nil, fmt.Errorf("resource ID is required")
	}

	return &SubscriptionUsage{
		id:             id,
		resourceType:   resourceType,
		resourceID:     resourceID,
		subscriptionID: subscriptionID,
		upload:         upload,
		download:       download,
		total:          total,
		period:         period,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}, nil
}

// ID returns the usage record ID
func (su *SubscriptionUsage) ID() uint {
	return su.id
}

// ResourceType returns the resource type ("node" or "forward_rule")
func (su *SubscriptionUsage) ResourceType() string {
	return su.resourceType
}

// ResourceID returns the resource ID (node_id or forward_rule_id)
func (su *SubscriptionUsage) ResourceID() uint {
	return su.resourceID
}

// SubscriptionID returns the subscription ID
func (su *SubscriptionUsage) SubscriptionID() *uint {
	return su.subscriptionID
}

// Upload returns the upload traffic in bytes
func (su *SubscriptionUsage) Upload() uint64 {
	return su.upload
}

// Download returns the download traffic in bytes
func (su *SubscriptionUsage) Download() uint64 {
	return su.download
}

// Total returns the total traffic in bytes
func (su *SubscriptionUsage) Total() uint64 {
	return su.total
}

// Period returns the period timestamp
func (su *SubscriptionUsage) Period() time.Time {
	return su.period
}

// CreatedAt returns when the usage record was created
func (su *SubscriptionUsage) CreatedAt() time.Time {
	return su.createdAt
}

// UpdatedAt returns when the usage record was last updated
func (su *SubscriptionUsage) UpdatedAt() time.Time {
	return su.updatedAt
}

// SetID sets the usage record ID (only for persistence layer use)
func (su *SubscriptionUsage) SetID(id uint) error {
	if su.id != 0 {
		return fmt.Errorf("subscription usage ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("subscription usage ID cannot be zero")
	}
	su.id = id
	return nil
}

// Accumulate adds upload and download traffic to the current record
func (su *SubscriptionUsage) Accumulate(upload, download uint64) error {
	if upload == 0 && download == 0 {
		return nil
	}

	su.upload += upload
	su.download += download
	su.total = su.upload + su.download
	su.updatedAt = time.Now()

	return nil
}

// TotalTraffic returns the total traffic (upload + download)
func (su *SubscriptionUsage) TotalTraffic() uint64 {
	return su.total
}

// UploadRatio calculates the ratio of upload traffic to total traffic
func (su *SubscriptionUsage) UploadRatio() float64 {
	if su.total == 0 {
		return 0.0
	}
	return float64(su.upload) / float64(su.total)
}

// DownloadRatio calculates the ratio of download traffic to total traffic
func (su *SubscriptionUsage) DownloadRatio() float64 {
	if su.total == 0 {
		return 0.0
	}
	return float64(su.download) / float64(su.total)
}

// IsEmpty checks if the usage record has no data
func (su *SubscriptionUsage) IsEmpty() bool {
	return su.upload == 0 && su.download == 0
}

// Reset resets all traffic counters to zero
func (su *SubscriptionUsage) Reset() error {
	su.upload = 0
	su.download = 0
	su.total = 0
	su.updatedAt = time.Now()
	return nil
}

// Validate performs domain-level validation
func (su *SubscriptionUsage) Validate() error {
	if su.resourceType == "" {
		return fmt.Errorf("resource type is required")
	}
	if su.resourceID == 0 {
		return fmt.Errorf("resource ID is required")
	}
	if su.total != su.upload+su.download {
		return fmt.Errorf("total traffic must equal upload + download")
	}
	return nil
}
