package admin

import "context"

// AdminTelegramBindingRepository defines the repository interface for admin telegram bindings
type AdminTelegramBindingRepository interface {
	// Create creates a new admin telegram binding
	Create(ctx context.Context, binding *AdminTelegramBinding) error

	// GetByID retrieves an admin telegram binding by internal ID
	GetByID(ctx context.Context, id uint) (*AdminTelegramBinding, error)

	// GetBySID retrieves an admin telegram binding by Stripe-style ID
	GetBySID(ctx context.Context, sid string) (*AdminTelegramBinding, error)

	// GetByUserID retrieves an admin telegram binding by user ID
	GetByUserID(ctx context.Context, userID uint) (*AdminTelegramBinding, error)

	// GetByTelegramUserID retrieves an admin telegram binding by Telegram user ID
	GetByTelegramUserID(ctx context.Context, telegramUserID int64) (*AdminTelegramBinding, error)

	// Update updates an existing admin telegram binding
	Update(ctx context.Context, binding *AdminTelegramBinding) error

	// Delete deletes an admin telegram binding
	Delete(ctx context.Context, id uint) error

	// GetAll retrieves all admin telegram bindings
	GetAll(ctx context.Context) ([]*AdminTelegramBinding, error)

	// FindBindingsForNodeOfflineNotification finds bindings that want node offline notifications
	FindBindingsForNodeOfflineNotification(ctx context.Context) ([]*AdminTelegramBinding, error)

	// FindBindingsForNodeOnlineNotification finds bindings that want node online notifications
	FindBindingsForNodeOnlineNotification(ctx context.Context) ([]*AdminTelegramBinding, error)

	// FindBindingsForAgentOfflineNotification finds bindings that want agent offline notifications
	FindBindingsForAgentOfflineNotification(ctx context.Context) ([]*AdminTelegramBinding, error)

	// FindBindingsForAgentOnlineNotification finds bindings that want agent online notifications
	FindBindingsForAgentOnlineNotification(ctx context.Context) ([]*AdminTelegramBinding, error)

	// FindBindingsForNewUserNotification finds bindings that want new user notifications
	FindBindingsForNewUserNotification(ctx context.Context) ([]*AdminTelegramBinding, error)

	// FindBindingsForPaymentSuccessNotification finds bindings that want payment success notifications
	FindBindingsForPaymentSuccessNotification(ctx context.Context) ([]*AdminTelegramBinding, error)

	// FindBindingsForDailySummary finds bindings that want daily summary
	FindBindingsForDailySummary(ctx context.Context) ([]*AdminTelegramBinding, error)

	// FindBindingsForWeeklySummary finds bindings that want weekly summary
	FindBindingsForWeeklySummary(ctx context.Context) ([]*AdminTelegramBinding, error)
}
