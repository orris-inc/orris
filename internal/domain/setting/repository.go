package setting

import (
	"context"
)

// Repository defines the interface for system setting persistence
type Repository interface {
	// GetByKey retrieves a setting by category and key
	GetByKey(ctx context.Context, category, key string) (*SystemSetting, error)

	// GetByCategory retrieves all settings in a category
	GetByCategory(ctx context.Context, category string) ([]*SystemSetting, error)

	// GetAll retrieves all system settings
	GetAll(ctx context.Context) ([]*SystemSetting, error)

	// Upsert creates or updates a setting
	Upsert(ctx context.Context, setting *SystemSetting) error

	// Delete removes a setting by category and key
	Delete(ctx context.Context, category, key string) error
}
