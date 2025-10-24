package migration

import (
	"fmt"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"orris/internal/shared/constants"
	"orris/internal/shared/logger"
)

// Manager handles database migrations with different strategies
type Manager struct {
	strategy Strategy
	logger   logger.Interface
}

// NewManager creates a new migration manager
func NewManager(environment string) *Manager {
	var strategy Strategy

	// Choose strategy based on environment
	switch strings.ToLower(environment) {
	case constants.EnvDevelopment:
		strategy = NewGormAutoMigrateStrategy()
	case constants.EnvTest, constants.EnvProduction:
		// Get absolute path to migration scripts
		scriptsPath, _ := filepath.Abs("./internal/infrastructure/migration/scripts")
		strategy = NewGolangMigrateStrategy(scriptsPath)
	default:
		// Default to GORM AutoMigrate for unknown environments
		strategy = NewGormAutoMigrateStrategy()
	}

	return &Manager{
		strategy: strategy,
		logger:   logger.NewLogger().With("component", "migration.manager"),
	}
}

func NewManagerWithStrategy(strategy Strategy) *Manager {
	return &Manager{
		strategy: strategy,
		logger:   logger.NewLogger().With("component", "migration.manager"),
	}
}

// Migrate executes the configured migration strategy
func (m *Manager) Migrate(db *gorm.DB, models ...interface{}) error {
	m.logger.Infow("starting database migration",
		"strategy", m.strategy.GetName(),
		"models_count", len(models))

	if err := m.strategy.Migrate(db, models...); err != nil {
		m.logger.Errorw("migration failed",
			"strategy", m.strategy.GetName(),
			"error", err)
		return fmt.Errorf("migration failed with strategy %s: %w", m.strategy.GetName(), err)
	}

	m.logger.Infow("database migration completed successfully",
		"strategy", m.strategy.GetName())

	return nil
}

// GetStrategy returns the current migration strategy
func (m *Manager) GetStrategy() Strategy {
	return m.strategy
}

// SetStrategy sets a new migration strategy
func (m *Manager) SetStrategy(strategy Strategy) {
	m.logger.Infow("changing migration strategy",
		"from", m.strategy.GetName(),
		"to", strategy.GetName())
	m.strategy = strategy
}

// GetStrategyInfo returns information about the current strategy
func (m *Manager) GetStrategyInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":        m.strategy.GetName(),
		"description": getStrategyDescription(m.strategy.GetName()),
	}
}

// getStrategyDescription returns a description for the given strategy
func getStrategyDescription(strategyName string) string {
	switch strategyName {
	case "gorm_auto_migrate":
		return "GORM AutoMigrate - Automatic schema migration based on struct definitions"
	case "golang_migrate":
		return "golang-migrate - Version-controlled SQL migration scripts"
	default:
		return "Unknown migration strategy"
	}
}

// MigrateWithGormAutoMigrate is a convenience function for GORM AutoMigrate
func MigrateWithGormAutoMigrate(db *gorm.DB, models ...interface{}) error {
	manager := NewManagerWithStrategy(NewGormAutoMigrateStrategy())
	return manager.Migrate(db, models...)
}

// MigrateWithGolangMigrate is a convenience function for golang-migrate
func MigrateWithGolangMigrate(db *gorm.DB, scriptsPath string, models ...interface{}) error {
	manager := NewManagerWithStrategy(NewGolangMigrateStrategy(scriptsPath))
	return manager.Migrate(db, models...)
}
