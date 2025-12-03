package migration

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pressly/goose/v3"
	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/logger"
)

// Strategy defines the interface for different migration strategies
type Strategy interface {
	// Migrate executes the migration strategy
	Migrate(db *gorm.DB, models ...interface{}) error
	// GetName returns the strategy name
	GetName() string
}

// GolangMigrateStrategy implements migration using golang-migrate
type GolangMigrateStrategy struct {
	scriptsPath string
	logger      logger.Interface
}

// NewGolangMigrateStrategy creates a new golang-migrate strategy
func NewGolangMigrateStrategy(scriptsPath string) Strategy {
	return &GolangMigrateStrategy{
		scriptsPath: scriptsPath,
		logger:      logger.NewLogger().With("component", "migration.golang-migrate"),
	}
}

// Migrate executes golang-migrate migration
func (s *GolangMigrateStrategy) Migrate(db *gorm.DB, models ...interface{}) error {
	s.logger.Infow("starting golang-migrate migration",
		"scripts_path", s.scriptsPath)

	// Get underlying SQL database
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Create migrate instance
	m, err := s.createMigrateInstance(sqlDB)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Get current version
	currentVersion, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		s.logger.Errorw("failed to get current migration version", "error", err)
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	s.logger.Infow("current migration status",
		"version", currentVersion,
		"dirty", dirty)

	// Check if database is dirty
	if dirty {
		s.logger.Warnw("database is in dirty state, please fix manually")
		return fmt.Errorf("database is in dirty state at version %d", currentVersion)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		s.logger.Errorw("migration failed", "error", err)
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Get final version
	finalVersion, _, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		s.logger.Errorw("failed to get final migration version", "error", err)
		return fmt.Errorf("failed to get final migration version: %w", err)
	}

	s.logger.Infow("migration completed successfully",
		"from_version", currentVersion,
		"to_version", finalVersion)

	return nil
}

// GetName returns the strategy name
func (s *GolangMigrateStrategy) GetName() string {
	return "golang_migrate"
}

// createMigrateInstance creates a new migrate instance
func (s *GolangMigrateStrategy) createMigrateInstance(sqlDB *sql.DB) (*migrate.Migrate, error) {
	// Create MySQL driver instance
	driver, err := mysql.WithInstance(sqlDB, &mysql.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create MySQL driver: %w", err)
	}

	// Create migrate instance with file source
	sourceURL := fmt.Sprintf("file://%s", s.scriptsPath)
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "mysql", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, nil
}

// MigrateDown executes down migrations to a specific version
func (s *GolangMigrateStrategy) MigrateDown(db *gorm.DB, steps int) error {
	s.logger.Infow("starting down migration", "steps", steps)

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	m, err := s.createMigrateInstance(sqlDB)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Execute down migrations
	if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
		s.logger.Errorw("down migration failed", "error", err)
		return fmt.Errorf("failed to run down migrations: %w", err)
	}

	s.logger.Infow("down migration completed successfully")
	return nil
}

// GetVersion returns the current migration version
func (s *GolangMigrateStrategy) GetVersion(db *gorm.DB) (uint, bool, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	m, err := s.createMigrateInstance(sqlDB)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	return m.Version()
}

// Force sets the database migration version and clears dirty flag
func (s *GolangMigrateStrategy) Force(db *gorm.DB, version int) error {
	s.logger.Infow("forcing migration version", "version", version)

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	m, err := s.createMigrateInstance(sqlDB)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Force(version); err != nil {
		s.logger.Errorw("force migration failed", "error", err)
		return fmt.Errorf("failed to force version: %w", err)
	}

	s.logger.Infow("force migration completed successfully", "version", version)
	return nil
}

type GooseStrategy struct {
	scriptsPath string
	logger      logger.Interface
}

func NewGooseStrategy(scriptsPath string) Strategy {
	return &GooseStrategy{
		scriptsPath: scriptsPath,
		logger:      logger.NewLogger().With("component", "migration.goose"),
	}
}

func (s *GooseStrategy) Migrate(db *gorm.DB, models ...interface{}) error {
	s.logger.Infow("starting goose migration",
		"scripts_path", s.scriptsPath)

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	currentVersion, err := goose.GetDBVersion(sqlDB)
	if err != nil {
		s.logger.Errorw("failed to get current version", "error", err)
		return fmt.Errorf("failed to get current version: %w", err)
	}

	s.logger.Infow("current migration status",
		"version", currentVersion)

	if err := goose.Up(sqlDB, s.scriptsPath); err != nil {
		s.logger.Errorw("migration failed", "error", err)
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	finalVersion, err := goose.GetDBVersion(sqlDB)
	if err != nil {
		s.logger.Errorw("failed to get final version", "error", err)
		return fmt.Errorf("failed to get final version: %w", err)
	}

	s.logger.Infow("migration completed successfully",
		"from_version", currentVersion,
		"to_version", finalVersion)

	return nil
}

func (s *GooseStrategy) GetName() string {
	return "goose"
}

func (s *GooseStrategy) MigrateDown(db *gorm.DB, steps int) error {
	s.logger.Infow("starting down migration", "steps", steps)

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	for i := 0; i < steps; i++ {
		if err := goose.Down(sqlDB, s.scriptsPath); err != nil {
			s.logger.Errorw("down migration failed", "error", err)
			return fmt.Errorf("failed to run down migration: %w", err)
		}
	}

	s.logger.Infow("down migration completed successfully")
	return nil
}

func (s *GooseStrategy) GetVersion(db *gorm.DB) (int64, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return 0, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := goose.SetDialect("mysql"); err != nil {
		return 0, fmt.Errorf("failed to set goose dialect: %w", err)
	}

	version, err := goose.GetDBVersion(sqlDB)
	if err != nil {
		return 0, fmt.Errorf("failed to get version: %w", err)
	}

	return version, nil
}

func (s *GooseStrategy) Status(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Status(sqlDB, s.scriptsPath); err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	return nil
}

func (s *GooseStrategy) Create(name string) error {
	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Create(nil, s.scriptsPath, name, "sql"); err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	s.logger.Infow("migration created successfully", "name", name)
	return nil
}
