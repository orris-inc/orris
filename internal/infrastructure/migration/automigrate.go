package migration

import (
	"orris/internal/infrastructure/persistence/models"
)

func AutoMigrateModels() []interface{} {
	return []interface{}{
		&models.UserModel{},
		&models.SubscriptionModel{},
		&models.SubscriptionPlanModel{},
		&models.SubscriptionTokenModel{},
		&models.SubscriptionHistoryModel{},
		&models.SubscriptionUsageModel{},
	}
}
