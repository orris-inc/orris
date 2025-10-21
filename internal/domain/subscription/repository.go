package subscription

import (
	"context"
	"time"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, subscription *Subscription) error
	GetByID(ctx context.Context, id uint) (*Subscription, error)
	GetByUserID(ctx context.Context, userID uint) ([]*Subscription, error)
	GetActiveByUserID(ctx context.Context, userID uint) (*Subscription, error)
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
	IncrementAPIRequests(ctx context.Context, subscriptionID uint, count uint64) error
	IncrementStorageUsed(ctx context.Context, subscriptionID uint, bytes uint64) error
	GetUsageHistory(ctx context.Context, subscriptionID uint, from, to time.Time) ([]*SubscriptionUsage, error)
	ResetUsage(ctx context.Context, subscriptionID uint, period time.Time) error
}

