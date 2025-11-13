package migrations

import (
	"orris/internal/infrastructure/persistence/models"

	"gorm.io/gorm"
)

func MigrateTicketTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.TicketModel{},
		&models.CommentModel{},
	)
}
