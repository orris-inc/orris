package telegram

import "context"

// TelegramBindingRepository defines the repository interface for telegram bindings
type TelegramBindingRepository interface {
	Create(ctx context.Context, binding *TelegramBinding) error
	GetByID(ctx context.Context, id uint) (*TelegramBinding, error)
	GetBySID(ctx context.Context, sid string) (*TelegramBinding, error)
	GetByUserID(ctx context.Context, userID uint) (*TelegramBinding, error)
	GetByTelegramUserID(ctx context.Context, telegramUserID int64) (*TelegramBinding, error)
	Update(ctx context.Context, binding *TelegramBinding) error
	Delete(ctx context.Context, id uint) error

	// FindBindingsForExpiringNotification finds bindings that need expiring notifications
	// Returns bindings where:
	// - notifyExpiring = true
	// - lastExpiringNotifyAt is null OR older than 24 hours
	FindBindingsForExpiringNotification(ctx context.Context) ([]*TelegramBinding, error)

	// FindBindingsForTrafficNotification finds bindings that need traffic notifications
	// Returns bindings where:
	// - notifyTraffic = true
	// - lastTrafficNotifyAt is null OR older than 24 hours
	FindBindingsForTrafficNotification(ctx context.Context) ([]*TelegramBinding, error)

	// GetUserIDsWithBinding returns user IDs that have telegram binding
	GetUserIDsWithBinding(ctx context.Context) ([]uint, error)
}
