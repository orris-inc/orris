package migrations

import (
	"orris/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

func MigratePaymentTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.PaymentModel{},
	)
}
