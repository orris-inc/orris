package dto

import "time"

// NewUserInfo contains information about a new user for alert notification.
// Used internally for building Telegram messages, not for JSON serialization.
type NewUserInfo struct {
	SID       string
	Email     string
	Name      string
	Source    string    // e.g., "registration", "oauth"
	CreatedAt time.Time // Registration timestamp
}

// PaymentInfo contains information about a successful payment for alert notification.
// Used internally for building Telegram messages, not for JSON serialization.
type PaymentInfo struct {
	PaymentSID    string
	UserSID       string
	UserEmail     string
	PlanName      string
	Amount        float64
	Currency      string
	PaymentMethod string
	TransactionID string    // Transaction ID from payment gateway
	PaidAt        time.Time // Payment timestamp
}

// OfflineNodeInfo contains information about an offline node for alert notification.
// Used internally for building Telegram messages, not for JSON serialization.
type OfflineNodeInfo struct {
	ID               uint
	SID              string
	Name             string
	LastSeenAt       *time.Time
	OfflineMinutes   int64
	MuteNotification bool
}

// OfflineAgentInfo contains information about an offline agent for alert notification.
// Used internally for building Telegram messages, not for JSON serialization.
type OfflineAgentInfo struct {
	ID               uint
	SID              string
	Name             string
	LastSeenAt       *time.Time
	OfflineMinutes   int64
	MuteNotification bool
}

// DailySummaryData contains aggregated daily business data for summary notification.
// Used internally for building Telegram messages, not for JSON serialization.
type DailySummaryData struct {
	Date             string // Report date (business timezone)
	NewUsers         int64
	ActiveUsers      int64
	NewSubscriptions int64
	TotalRevenue     float64
	Currency         string

	// Node status
	TotalNodes   int64
	OnlineNodes  int64
	OfflineNodes int64

	// Agent status
	TotalAgents   int64
	OnlineAgents  int64
	OfflineAgents int64

	// Traffic stats
	TotalTrafficBytes uint64
}

// WeeklySummaryData contains aggregated weekly business data with comparison for summary notification.
// Used internally for building Telegram messages, not for JSON serialization.
type WeeklySummaryData struct {
	// Period info
	WeekStart string
	WeekEnd   string

	// Current week stats
	NewUsers         int64
	ActiveUsers      int64
	NewSubscriptions int64
	TotalRevenue     float64
	Currency         string

	// Previous week stats for comparison
	PrevNewUsers         int64
	PrevNewSubscriptions int64
	PrevTotalRevenue     float64

	// Change percentages
	UserChangePercent    float64
	SubChangePercent     float64
	RevenueChangePercent float64

	// Node status
	TotalNodes   int64
	OnlineNodes  int64
	OfflineNodes int64

	// Agent status
	TotalAgents   int64
	OnlineAgents  int64
	OfflineAgents int64

	// Traffic stats
	TotalTrafficBytes     uint64
	PrevTotalTrafficBytes uint64
	TrafficChangePercent  float64
}

// ExpiringAgentInfo contains information about an expiring agent for notification.
// Used internally for building Telegram messages, not for JSON serialization.
type ExpiringAgentInfo struct {
	ID            uint
	SID           string
	Name          string
	ExpiresAt     time.Time
	DaysRemaining int
	CostLabel     *string
}

// ExpiringNodeInfo contains information about an expiring node for notification.
// Used internally for building Telegram messages, not for JSON serialization.
type ExpiringNodeInfo struct {
	ID            uint
	SID           string
	Name          string
	ExpiresAt     time.Time
	DaysRemaining int
	CostLabel     *string
}
