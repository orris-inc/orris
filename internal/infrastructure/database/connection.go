package database

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/orris-inc/orris/internal/shared/config"
	appLogger "github.com/orris-inc/orris/internal/shared/logger"
)

var (
	db   *gorm.DB
	dbMu sync.RWMutex
)

// Init initializes the database connection with minimal configuration
func Init(cfg *config.DatabaseConfig) error {
	// Build DSN with essential parameters
	// Use loc=Local to parse time in server's local timezone
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=true&loc=Local",
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

	dbMu.Lock()
	db = database
	dbMu.Unlock()

	appLogger.Info("database connection established",
		"database", cfg.Database)

	return nil
}

// Get returns the database connection
func Get() *gorm.DB {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return db
}

// Close closes the database connection
func Close() error {
	dbMu.RLock()
	currentDB := db
	dbMu.RUnlock()

	if currentDB == nil {
		return nil
	}

	sqlDB, err := currentDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	appLogger.Info("database connection closed")
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
		appLogger.Error("database error", "details", msg)
	} else if strings.Contains(msg, "slow sql") || strings.Contains(msg, "SLOW SQL") {
		appLogger.Warn("slow query", "details", msg)
	} else {
		appLogger.Debug("database query", "details", msg)
	}
}
