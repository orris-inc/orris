package migrations

import (
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// MigrateSubscriptionTrafficOverrides adds traffic_limit_override and traffic_used_adjustment
// columns to the subscriptions table for admin override support.
func MigrateSubscriptionTrafficOverrides(db *gorm.DB) error {
	return db.AutoMigrate(&models.SubscriptionModel{})
}
