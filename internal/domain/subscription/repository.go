package subscription

import (
	"context"

	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
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

type PlanRepository interface {
	Create(ctx context.Context, plan *Plan) error
	GetByID(ctx context.Context, id uint) (*Plan, error)
	GetBySID(ctx context.Context, sid string) (*Plan, error)
	GetBySlug(ctx context.Context, slug string) (*Plan, error)
	Update(ctx context.Context, plan *Plan) error
	Delete(ctx context.Context, id uint) error

	GetActivePublicPlans(ctx context.Context) ([]*Plan, error)
	GetAllActive(ctx context.Context) ([]*Plan, error)
	List(ctx context.Context, filter PlanFilter) ([]*Plan, int64, error)

	ExistsBySlug(ctx context.Context, slug string) (bool, error)
}

type PlanFilter struct {
	Status   *string
	IsPublic *bool
	PlanType *string // Optional: filter by plan type ("node" or "forward")
	Page     int
	PageSize int
	SortBy   string
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
	// DeleteByPlanID deletes all pricing records for a specific plan
	DeleteByPlanID(ctx context.Context, planID uint) error
}
