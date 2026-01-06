package subscription

import (
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/id"
)

// Granularity represents the time granularity for usage statistics
type Granularity string

const (
	// GranularityDaily represents daily aggregated statistics
	GranularityDaily Granularity = "daily"
	// GranularityMonthly represents monthly aggregated statistics
	GranularityMonthly Granularity = "monthly"
)

// String returns the string representation of Granularity
func (g Granularity) String() string {
	return string(g)
}

// IsValid checks if the granularity value is valid
func (g Granularity) IsValid() bool {
	return g == GranularityDaily || g == GranularityMonthly
}

// SubscriptionUsageStats represents aggregated usage statistics entity for a subscription
type SubscriptionUsageStats struct {
	id             uint
	sid            string // Stripe-style ID: usagestat_xxx
	resourceType   string
	resourceID     uint
	subscriptionID *uint
	upload         uint64
	download       uint64
	total          uint64
	granularity    Granularity
	period         time.Time // date for daily, first day of month for monthly
	createdAt      time.Time
	updatedAt      time.Time
}

// NewSubscriptionUsageStats creates a new subscription usage stats record
func NewSubscriptionUsageStats(
	resourceType string,
	resourceID uint,
	subscriptionID *uint,
	granularity Granularity,
	period time.Time,
) (*SubscriptionUsageStats, error) {
	if resourceType == "" {
		return nil, fmt.Errorf("resource type is required")
	}
	if resourceID == 0 {
		return nil, fmt.Errorf("resource ID is required")
	}
	if !granularity.IsValid() {
		return nil, fmt.Errorf("invalid granularity: %s", granularity)
	}

	// Generate Stripe-style SID
	sid, err := id.NewSubscriptionUsageStatsID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	now := biztime.NowUTC()
	return &SubscriptionUsageStats{
		sid:            sid,
		resourceType:   resourceType,
		resourceID:     resourceID,
		subscriptionID: subscriptionID,
		upload:         0,
		download:       0,
		total:          0,
		granularity:    granularity,
		period:         period,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// ReconstructSubscriptionUsageStats reconstructs a subscription usage stats entity from persistence
func ReconstructSubscriptionUsageStats(
	id uint,
	sid string,
	resourceType string,
	resourceID uint,
	subscriptionID *uint,
	upload, download, total uint64,
	granularity Granularity,
	period, createdAt, updatedAt time.Time,
) (*SubscriptionUsageStats, error) {
	if id == 0 {
		return nil, fmt.Errorf("subscription usage stats ID cannot be zero")
	}
	if sid == "" {
		return nil, fmt.Errorf("subscription usage stats SID is required")
	}
	if resourceType == "" {
		return nil, fmt.Errorf("resource type is required")
	}
	if resourceID == 0 {
		return nil, fmt.Errorf("resource ID is required")
	}
	if !granularity.IsValid() {
		return nil, fmt.Errorf("invalid granularity: %s", granularity)
	}

	return &SubscriptionUsageStats{
		id:             id,
		sid:            sid,
		resourceType:   resourceType,
		resourceID:     resourceID,
		subscriptionID: subscriptionID,
		upload:         upload,
		download:       download,
		total:          total,
		granularity:    granularity,
		period:         period,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}, nil
}

// ID returns the usage stats record ID
func (s *SubscriptionUsageStats) ID() uint {
	return s.id
}

// SID returns the Stripe-style ID
func (s *SubscriptionUsageStats) SID() string {
	return s.sid
}

// SetSID sets the Stripe-style ID (only for persistence layer use)
func (s *SubscriptionUsageStats) SetSID(sid string) {
	s.sid = sid
}

// ResourceType returns the resource type ("node" or "forward_rule")
func (s *SubscriptionUsageStats) ResourceType() string {
	return s.resourceType
}

// ResourceID returns the resource ID (node_id or forward_rule_id)
func (s *SubscriptionUsageStats) ResourceID() uint {
	return s.resourceID
}

// SubscriptionID returns the subscription ID
func (s *SubscriptionUsageStats) SubscriptionID() *uint {
	return s.subscriptionID
}

// Upload returns the upload traffic in bytes
func (s *SubscriptionUsageStats) Upload() uint64 {
	return s.upload
}

// Download returns the download traffic in bytes
func (s *SubscriptionUsageStats) Download() uint64 {
	return s.download
}

// Total returns the total traffic in bytes
func (s *SubscriptionUsageStats) Total() uint64 {
	return s.total
}

// Granularity returns the time granularity (daily or monthly)
func (s *SubscriptionUsageStats) Granularity() Granularity {
	return s.granularity
}

// Period returns the period date (date for daily, first day of month for monthly)
func (s *SubscriptionUsageStats) Period() time.Time {
	return s.period
}

// CreatedAt returns when the stats record was created
func (s *SubscriptionUsageStats) CreatedAt() time.Time {
	return s.createdAt
}

// UpdatedAt returns when the stats record was last updated
func (s *SubscriptionUsageStats) UpdatedAt() time.Time {
	return s.updatedAt
}

// SetID sets the usage stats record ID (only for persistence layer use)
func (s *SubscriptionUsageStats) SetID(id uint) error {
	if s.id != 0 {
		return fmt.Errorf("subscription usage stats ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("subscription usage stats ID cannot be zero")
	}
	s.id = id
	return nil
}

// SetUsage sets the upload, download and total traffic values
func (s *SubscriptionUsageStats) SetUsage(upload, download uint64) {
	s.upload = upload
	s.download = download
	s.total = upload + download
	s.updatedAt = biztime.NowUTC()
}

// Accumulate adds upload and download traffic to the current record
func (s *SubscriptionUsageStats) Accumulate(upload, download uint64) error {
	if upload == 0 && download == 0 {
		return nil
	}

	s.upload += upload
	s.download += download
	s.total = s.upload + s.download
	s.updatedAt = biztime.NowUTC()

	return nil
}

// TotalTraffic returns the total traffic (upload + download)
func (s *SubscriptionUsageStats) TotalTraffic() uint64 {
	return s.total
}

// IsEmpty checks if the usage stats record has no data
func (s *SubscriptionUsageStats) IsEmpty() bool {
	return s.upload == 0 && s.download == 0
}

// Validate performs domain-level validation
func (s *SubscriptionUsageStats) Validate() error {
	if s.resourceType == "" {
		return fmt.Errorf("resource type is required")
	}
	if s.resourceID == 0 {
		return fmt.Errorf("resource ID is required")
	}
	if !s.granularity.IsValid() {
		return fmt.Errorf("invalid granularity: %s", s.granularity)
	}
	if s.total != s.upload+s.download {
		return fmt.Errorf("total traffic must equal upload + download")
	}
	return nil
}
