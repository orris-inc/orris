package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/subscription"
)

type TokenGenerator interface {
	Generate(prefix string) (plainToken string, hash string, err error)
	Hash(plainToken string) string
}

// SubscriptionChangeNotifier defines the interface for notifying subscription changes to node agents.
type SubscriptionChangeNotifier interface {
	// NotifySubscriptionActivation notifies nodes when a subscription becomes active.
	NotifySubscriptionActivation(ctx context.Context, sub *subscription.Subscription) error
	// NotifySubscriptionDeactivation notifies nodes when a subscription is deactivated/expired/cancelled.
	NotifySubscriptionDeactivation(ctx context.Context, sub *subscription.Subscription) error
	// NotifySubscriptionUpdate notifies nodes when a subscription is updated.
	NotifySubscriptionUpdate(ctx context.Context, sub *subscription.Subscription) error
}

// QuotaCacheManager defines the interface for managing subscription quota cache.
// This is used to invalidate or update cache when subscription status changes.
type QuotaCacheManager interface {
	// InvalidateQuota removes quota cache for a subscription, forcing reload on next access.
	InvalidateQuota(ctx context.Context, subscriptionID uint) error
	// SyncQuotaFromSubscription syncs quota from subscription to cache.
	SyncQuotaFromSubscription(ctx context.Context, sub *subscription.Subscription) error
	// SetSuspended updates only the suspended status in cache.
	SetSuspended(ctx context.Context, subscriptionID uint, suspended bool) error
}
