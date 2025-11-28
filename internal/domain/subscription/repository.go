package subscription

import (
	"context"
	"time"

	vo "orris/internal/domain/subscription/value_objects"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, subscription *Subscription) error
	GetByID(ctx context.Context, id uint) (*Subscription, error)
	GetByUserID(ctx context.Context, userID uint) ([]*Subscription, error)
	GetActiveByUserID(ctx context.Context, userID uint) ([]*Subscription, error)
	GetActiveSubscriptionsByNodeID(ctx context.Context, nodeID uint) ([]*Subscription, error)
	Update(ctx context.Context, subscription *Subscription) error
	Delete(ctx context.Context, id uint) error

	FindExpiringSubscriptions(ctx context.Context, days int) ([]*Subscription, error)
	FindExpiredSubscriptions(ctx context.Context) ([]*Subscription, error)
	List(ctx context.Context, filter SubscriptionFilter) ([]*Subscription, int64, error)

	CountByPlanID(ctx context.Context, planID uint) (int64, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
}

type SubscriptionFilter struct {
	UserID   *uint
	PlanID   *uint
	Status   *string
	Page     int
	PageSize int
	SortBy   string
	SortDesc bool
}

type SubscriptionPlanRepository interface {
	Create(ctx context.Context, plan *SubscriptionPlan) error
	GetByID(ctx context.Context, id uint) (*SubscriptionPlan, error)
	GetBySlug(ctx context.Context, slug string) (*SubscriptionPlan, error)
	Update(ctx context.Context, plan *SubscriptionPlan) error
	Delete(ctx context.Context, id uint) error

	GetActivePublicPlans(ctx context.Context) ([]*SubscriptionPlan, error)
	GetAllActive(ctx context.Context) ([]*SubscriptionPlan, error)
	List(ctx context.Context, filter PlanFilter) ([]*SubscriptionPlan, int64, error)

	ExistsBySlug(ctx context.Context, slug string) (bool, error)
}

type PlanFilter struct {
	Status       *string
	IsPublic     *bool
	BillingCycle *string
	Page         int
	PageSize     int
	SortBy       string
}

type SubscriptionTokenRepository interface {
	Create(ctx context.Context, token *SubscriptionToken) error
	GetByID(ctx context.Context, id uint) (*SubscriptionToken, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*SubscriptionToken, error)
	Update(ctx context.Context, token *SubscriptionToken) error
	Delete(ctx context.Context, id uint) error

	GetBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*SubscriptionToken, error)
	GetActiveBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*SubscriptionToken, error)

	RevokeAllBySubscriptionID(ctx context.Context, subscriptionID uint) error
	DeleteExpiredTokens(ctx context.Context) error
}

type SubscriptionUsageRepository interface {
	GetCurrentUsage(ctx context.Context, subscriptionID uint) (*SubscriptionUsage, error)
	Upsert(ctx context.Context, usage *SubscriptionUsage) error
	GetUsageHistory(ctx context.Context, subscriptionID uint, from, to time.Time) ([]*SubscriptionUsage, error)
	ResetUsage(ctx context.Context, subscriptionID uint, period time.Time) error
}

// PlanPricingRepository handles plan pricing persistence
type PlanPricingRepository interface {
	Create(ctx context.Context, pricing *vo.PlanPricing) error
	GetByID(ctx context.Context, id uint) (*vo.PlanPricing, error)
	GetByPlanAndCycle(ctx context.Context, planID uint, cycle vo.BillingCycle) (*vo.PlanPricing, error)
	GetByPlanID(ctx context.Context, planID uint) ([]*vo.PlanPricing, error)
	GetActivePricings(ctx context.Context, planID uint) ([]*vo.PlanPricing, error)
	// GetActivePricingsByPlanIDs retrieves active pricings for multiple plans in a single query
	// Returns a map where key is planID and value is the list of active pricings for that plan
	GetActivePricingsByPlanIDs(ctx context.Context, planIDs []uint) (map[uint][]*vo.PlanPricing, error)
	Update(ctx context.Context, pricing *vo.PlanPricing) error
	Delete(ctx context.Context, id uint) error
}
