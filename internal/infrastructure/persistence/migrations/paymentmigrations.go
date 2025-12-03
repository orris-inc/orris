package migrations

import (
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

func MigratePaymentTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.PaymentModel{},
	)
}
