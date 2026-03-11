package usecases

import (
	"context"
	"time"
)

// CachedQuotaInfo represents the cached subscription quota information.
// This mirrors cache.CachedQuota to avoid import cycle.
type CachedQuotaInfo struct {
	Limit       int64     // Traffic limit in bytes
	PeriodStart time.Time // Billing period start
	PeriodEnd   time.Time // Billing period end
	PlanType    string    // node/forward/hybrid
	Suspended   bool      // Whether the subscription is suspended
	NotFound    bool      // Null marker: subscription confirmed not found/inactive in DB
}

// NodeSubscriptionQuotaCache defines the interface for subscription quota caching.
type NodeSubscriptionQuotaCache interface {
	// GetQuota retrieves subscription quota information from cache.
	// Returns nil if cache does not exist.
	GetQuota(ctx context.Context, subscriptionID uint) (*CachedQuotaInfo, error)

	// MarkSuspended marks the subscription as suspended in cache.
	MarkSuspended(ctx context.Context, subscriptionID uint) error
}

// NodeSubscriptionQuotaLoader defines the interface for lazy loading subscription quota.
// This is used when quota cache miss occurs to load quota from database.
type NodeSubscriptionQuotaLoader interface {
	// LoadQuotaByID loads subscription quota from database and caches it.
	// Returns the cached quota info, or nil if subscription/plan not found.
	LoadQuotaByID(ctx context.Context, subscriptionID uint) (*CachedQuotaInfo, error)
}
