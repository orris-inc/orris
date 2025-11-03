package migrations

import (
	"gorm.io/gorm"
	"orris/internal/infrastructure/persistence/models"
)

func MigratePaymentTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.PaymentModel{},
	)
}
