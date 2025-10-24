package migrations

import (
	"gorm.io/gorm"
	"orris/internal/infrastructure/persistence"
)

func MigrateTicketTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&persistence.TicketModel{},
		&persistence.CommentModel{},
	)
}
