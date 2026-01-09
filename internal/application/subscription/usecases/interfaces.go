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
