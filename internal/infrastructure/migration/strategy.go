package migration

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"orris/internal/shared/logger"
)

// Strategy defines the interface for different migration strategies
type Strategy interface {
	// Migrate executes the migration strategy
	Migrate(db *gorm.DB, models ...interface{}) error
	// GetName returns the strategy name
	GetName() string
}

// GormAutoMigrateStrategy implements migration using GORM AutoMigrate
type GormAutoMigrateStrategy struct {
	logger *zap.Logger
}

// NewGormAutoMigrateStrategy creates a new GORM AutoMigrate strategy
func NewGormAutoMigrateStrategy() Strategy {
	return &GormAutoMigrateStrategy{
		logger: logger.WithComponent("migration.gorm"),
	}
}

// Migrate executes GORM AutoMigrate
func (s *GormAutoMigrateStrategy) Migrate(db *gorm.DB, models ...interface{}) error {
	s.logger.Info("starting GORM AutoMigrate")

	if err := db.AutoMigrate(models...); err != nil {
		s.logger.Error("GORM AutoMigrate failed", zap.Error(err))
		return fmt.Errorf("failed to run GORM AutoMigrate: %w", err)
	}

	s.logger.Info("GORM AutoMigrate completed successfully")
	return nil
}

// GetName returns the strategy name
func (s *GormAutoMigrateStrategy) GetName() string {
	return "gorm_auto_migrate"
}

// GolangMigrateStrategy implements migration using golang-migrate
type GolangMigrateStrategy struct {
	scriptsPath string
	logger      *zap.Logger
}

// NewGolangMigrateStrategy creates a new golang-migrate strategy
func NewGolangMigrateStrategy(scriptsPath string) Strategy {
	return &GolangMigrateStrategy{
		scriptsPath: scriptsPath,
		logger:      logger.WithComponent("migration.golang-migrate"),
	}
}

// Migrate executes golang-migrate migration
func (s *GolangMigrateStrategy) Migrate(db *gorm.DB, models ...interface{}) error {
	s.logger.Info("starting golang-migrate migration",
		zap.String("scripts_path", s.scriptsPath))

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
		s.logger.Error("failed to get current migration version", zap.Error(err))
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	s.logger.Info("current migration status",
		zap.Uint("version", currentVersion),
		zap.Bool("dirty", dirty))

	// Check if database is dirty
	if dirty {
		s.logger.Warn("database is in dirty state, please fix manually")
		return fmt.Errorf("database is in dirty state at version %d", currentVersion)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		s.logger.Error("migration failed", zap.Error(err))
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Get final version
	finalVersion, _, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		s.logger.Error("failed to get final migration version", zap.Error(err))
		return fmt.Errorf("failed to get final migration version: %w", err)
	}

	s.logger.Info("migration completed successfully",
		zap.Uint("from_version", currentVersion),
		zap.Uint("to_version", finalVersion))

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
	s.logger.Info("starting down migration", zap.Int("steps", steps))

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
		s.logger.Error("down migration failed", zap.Error(err))
		return fmt.Errorf("failed to run down migrations: %w", err)
	}

	s.logger.Info("down migration completed successfully")
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
	s.logger.Info("forcing migration version", zap.Int("version", version))

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
		s.logger.Error("force migration failed", zap.Error(err))
		return fmt.Errorf("failed to force version: %w", err)
	}

	s.logger.Info("force migration completed successfully", zap.Int("version", version))
	return nil
}

type GooseStrategy struct {
	scriptsPath string
	logger      *zap.Logger
}

func NewGooseStrategy(scriptsPath string) Strategy {
	return &GooseStrategy{
		scriptsPath: scriptsPath,
		logger:      logger.WithComponent("migration.goose"),
	}
}

func (s *GooseStrategy) Migrate(db *gorm.DB, models ...interface{}) error {
	s.logger.Info("starting goose migration",
		zap.String("scripts_path", s.scriptsPath))

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	currentVersion, err := goose.GetDBVersion(sqlDB)
	if err != nil {
		s.logger.Error("failed to get current version", zap.Error(err))
		return fmt.Errorf("failed to get current version: %w", err)
	}

	s.logger.Info("current migration status",
		zap.Int64("version", currentVersion))

	if err := goose.Up(sqlDB, s.scriptsPath); err != nil {
		s.logger.Error("migration failed", zap.Error(err))
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	finalVersion, err := goose.GetDBVersion(sqlDB)
	if err != nil {
		s.logger.Error("failed to get final version", zap.Error(err))
		return fmt.Errorf("failed to get final version: %w", err)
	}

	s.logger.Info("migration completed successfully",
		zap.Int64("from_version", currentVersion),
		zap.Int64("to_version", finalVersion))

	return nil
}

func (s *GooseStrategy) GetName() string {
	return "goose"
}

func (s *GooseStrategy) MigrateDown(db *gorm.DB, steps int) error {
	s.logger.Info("starting down migration", zap.Int("steps", steps))

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	for i := 0; i < steps; i++ {
		if err := goose.Down(sqlDB, s.scriptsPath); err != nil {
			s.logger.Error("down migration failed", zap.Error(err))
			return fmt.Errorf("failed to run down migration: %w", err)
		}
	}

	s.logger.Info("down migration completed successfully")
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

	s.logger.Info("migration created successfully", zap.String("name", name))
	return nil
}