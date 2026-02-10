package adapters

import (
	"context"

	settingUsecases "github.com/orris-inc/orris/internal/application/setting/usecases"
)

// SubscriptionSettingCategory is the category name for subscription settings.
const SubscriptionSettingCategory = "subscription"

// SubscriptionSettingKeys defines the keys for subscription settings.
const (
	// SubscriptionSettingShowInfoNodes controls whether to show info nodes in subscription.
	SubscriptionSettingShowInfoNodes = "show_info_nodes"
)

// SubscriptionSettingProviderAdapter adapts SettingProvider to SubscriptionSettingProvider interface.
type SubscriptionSettingProviderAdapter struct {
	provider *settingUsecases.SettingProvider
}

// NewSubscriptionSettingProviderAdapter creates a new SubscriptionSettingProviderAdapter.
func NewSubscriptionSettingProviderAdapter(provider *settingUsecases.SettingProvider) *SubscriptionSettingProviderAdapter {
	return &SubscriptionSettingProviderAdapter{
		provider: provider,
	}
}

// IsShowInfoNodesEnabled returns whether to show info nodes (expire/traffic) in subscription.
// Default is false (disabled).
func (a *SubscriptionSettingProviderAdapter) IsShowInfoNodesEnabled(ctx context.Context) bool {
	return a.provider.GetBool(ctx, SubscriptionSettingCategory, SubscriptionSettingShowInfoNodes, false)
}
