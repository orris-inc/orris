package migration

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"orris/internal/shared/constants"
	"orris/internal/shared/logger"
)

// Manager handles database migrations with different strategies
type Manager struct {
	strategy Strategy
	logger   *zap.Logger
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
		logger:   logger.WithComponent("migration.manager"),
	}
}

// NewManagerWithStrategy creates a new migration manager with a specific strategy
func NewManagerWithStrategy(strategy Strategy) *Manager {
	return &Manager{
		strategy: strategy,
		logger:   logger.WithComponent("migration.manager"),
	}
}

// Migrate executes the configured migration strategy
func (m *Manager) Migrate(db *gorm.DB, models ...interface{}) error {
	m.logger.Info("starting database migration",
		zap.String("strategy", m.strategy.GetName()),
		zap.Int("models_count", len(models)))

	if err := m.strategy.Migrate(db, models...); err != nil {
		m.logger.Error("migration failed",
			zap.String("strategy", m.strategy.GetName()),
			zap.Error(err))
		return fmt.Errorf("migration failed with strategy %s: %w", m.strategy.GetName(), err)
	}

	m.logger.Info("database migration completed successfully",
		zap.String("strategy", m.strategy.GetName()))

	return nil
}

// GetStrategy returns the current migration strategy
func (m *Manager) GetStrategy() Strategy {
	return m.strategy
}

// SetStrategy sets a new migration strategy
func (m *Manager) SetStrategy(strategy Strategy) {
	m.logger.Info("changing migration strategy",
		zap.String("from", m.strategy.GetName()),
		zap.String("to", strategy.GetName()))
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