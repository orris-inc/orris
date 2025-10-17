package database

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"orris/internal/infrastructure/config"
	appLogger "orris/internal/shared/logger"
)

var db *gorm.DB

// Init initializes the database connection with minimal configuration
func Init(cfg *config.DatabaseConfig) error {
	// Build DSN with essential parameters
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=UTC",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	// Create custom logger to filter schema queries
	gormLogger := logger.New(
		&filteredLogger{},
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// Open database connection with essential optimizations
	database, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,
		SkipInitializeWithVersion: true, // Skip schema validation query
	}), &gorm.Config{
		Logger:      gormLogger,
		PrepareStmt: true, // Cache prepared statements
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	db = database

	appLogger.Info("database connection established",
		zap.String("database", cfg.Database))

	return nil
}

// Get returns the database connection
func Get() *gorm.DB {
	return db
}

// Close closes the database connection
func Close() error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	appLogger.Info("database connection closed")
	return nil
}

// Migrate runs database migrations for given models
func Migrate(models ...interface{}) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	appLogger.Info("running database migrations")
	
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	appLogger.Info("database migrations completed successfully")
	return nil
}

// filteredLogger filters out schema validation queries
type filteredLogger struct{}

func (l *filteredLogger) Printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	
	// Filter schema validation queries
	if strings.Contains(strings.ToLower(msg), "information_schema.schemata") ||
		strings.Contains(strings.ToLower(msg), "select version()") {
		return
	}
	
	// Log other messages
	if strings.Contains(msg, "[error]") || strings.Contains(msg, "ERROR") {
		appLogger.Error("database error", zap.String("details", msg))
	} else if strings.Contains(msg, "slow sql") || strings.Contains(msg, "SLOW SQL") {
		appLogger.Warn("slow query", zap.String("details", msg))
	} else {
		appLogger.Debug("database query", zap.String("details", msg))
	}
}