package migration

import (
	"fmt"
	"path/filepath"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/shared/logger"
)

// Manager handles database migrations with different strategies
type Manager struct {
	strategy Strategy
	logger   logger.Interface
}

// NewManager creates a new migration manager
func NewManager(environment string, log logger.Interface) *Manager {
	var strategy Strategy

	scriptsPath, _ := filepath.Abs("./internal/infrastructure/migration/scripts")
	strategy = NewGooseStrategy(scriptsPath, log)

	return &Manager{
		strategy: strategy,
		logger:   log.With("component", "migration.manager"),
	}
}

func NewManagerWithStrategy(strategy Strategy, log logger.Interface) *Manager {
	return &Manager{
		strategy: strategy,
		logger:   log.With("component", "migration.manager"),
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
	case "goose":
		return "Goose - Version-controlled SQL migration scripts"
	case "golang_migrate":
		return "golang-migrate - Version-controlled SQL migration scripts"
	default:
		return "Unknown migration strategy"
	}
}

// MigrateWithGoose is a convenience function for goose migration
func MigrateWithGoose(db *gorm.DB, scriptsPath string, log logger.Interface, models ...interface{}) error {
	manager := NewManagerWithStrategy(NewGooseStrategy(scriptsPath, log), log)
	return manager.Migrate(db, models...)
}

// MigrateWithGolangMigrate is a convenience function for golang-migrate
func MigrateWithGolangMigrate(db *gorm.DB, scriptsPath string, log logger.Interface, models ...interface{}) error {
	manager := NewManagerWithStrategy(NewGolangMigrateStrategy(scriptsPath, log), log)
	return manager.Migrate(db, models...)
}
