package migrations

import (
	"fmt"

	"orris/internal/infrastructure/persistence/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MigrateSubscriptionUUID adds UUID field to subscriptions table and generates UUIDs for existing records
func MigrateSubscriptionUUID(db *gorm.DB) error {
	// First, auto-migrate to add the UUID column
	if err := db.AutoMigrate(&models.SubscriptionModel{}); err != nil {
		return fmt.Errorf("failed to auto-migrate subscription model: %w", err)
	}

	// Then, generate UUIDs for existing subscriptions that don't have one
	var subscriptions []models.SubscriptionModel
	if err := db.Where("uuid IS NULL OR uuid = ''").Find(&subscriptions).Error; err != nil {
		return fmt.Errorf("failed to query subscriptions without UUID: %w", err)
	}

	// Update each subscription with a unique UUID
	for i := range subscriptions {
		subscriptions[i].UUID = uuid.New().String()
		if err := db.Model(&subscriptions[i]).Update("uuid", subscriptions[i].UUID).Error; err != nil {
			return fmt.Errorf("failed to update subscription %d with UUID: %w", subscriptions[i].ID, err)
		}
	}

	return nil
}
