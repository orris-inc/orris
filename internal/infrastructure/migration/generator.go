package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"orris/internal/shared/logger"
)

// Generator handles creation of new migration files
type Generator struct {
	scriptsPath string
	logger      logger.Interface
}

// NewGenerator creates a new migration generator
func NewGenerator(scriptsPath string) *Generator {
	return &Generator{
		scriptsPath: scriptsPath,
		logger:      logger.NewLogger().With("component", "migration.generator"),
	}
}

// CreateMigration creates a new migration file pair (up and down)
func (g *Generator) CreateMigration(name string) error {
	g.logger.Infow("creating new migration", "name", name)

	// Generate timestamp
	timestamp := time.Now().Format("20060102150405")

	// Generate file names
	upFileName := fmt.Sprintf("%s_%s.up.sql", timestamp, name)
	downFileName := fmt.Sprintf("%s_%s.down.sql", timestamp, name)

	upFilePath := filepath.Join(g.scriptsPath, upFileName)
	downFilePath := filepath.Join(g.scriptsPath, downFileName)

	// Ensure scripts directory exists
	if err := os.MkdirAll(g.scriptsPath, 0755); err != nil {
		return fmt.Errorf("failed to create scripts directory: %w", err)
	}

	// Create up migration file
	upContent := g.generateUpMigrationTemplate(name)
	if err := g.writeFile(upFilePath, upContent); err != nil {
		return fmt.Errorf("failed to create up migration file: %w", err)
	}

	// Create down migration file
	downContent := g.generateDownMigrationTemplate(name)
	if err := g.writeFile(downFilePath, downContent); err != nil {
		return fmt.Errorf("failed to create down migration file: %w", err)
	}

	g.logger.Infow("migration files created successfully",
		"up_file", upFilePath,
		"down_file", downFilePath)

	return nil
}

// writeFile writes content to a file
func (g *Generator) writeFile(filePath, content string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// generateUpMigrationTemplate generates a template for up migration
func (g *Generator) generateUpMigrationTemplate(name string) string {
	return fmt.Sprintf(`-- Migration: %s
-- Created: %s
-- Description: Add description here

-- Add your SQL statements here
-- Example:
-- CREATE TABLE example_table (
--     id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
--     name VARCHAR(255) NOT NULL,
--     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
--     updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
-- );

`, name, time.Now().Format("2006-01-02 15:04:05"))
}

// generateDownMigrationTemplate generates a template for down migration
func (g *Generator) generateDownMigrationTemplate(name string) string {
	return fmt.Sprintf(`-- Rollback Migration: %s
-- Created: %s
-- Description: Add rollback description here

-- Add your rollback SQL statements here
-- Example:
-- DROP TABLE IF EXISTS example_table;

`, name, time.Now().Format("2006-01-02 15:04:05"))
}

// CreateUserTableMigration creates the initial user table migration
func (g *Generator) CreateUserTableMigration() error {
	g.logger.Infow("creating initial user table migration")

	// Use a fixed timestamp for the initial migration
	timestamp := "000001"
	name := "create_users_table"

	upFileName := fmt.Sprintf("%s_%s.up.sql", timestamp, name)
	downFileName := fmt.Sprintf("%s_%s.down.sql", timestamp, name)

	upFilePath := filepath.Join(g.scriptsPath, upFileName)
	downFilePath := filepath.Join(g.scriptsPath, downFileName)

	// Ensure scripts directory exists
	if err := os.MkdirAll(g.scriptsPath, 0755); err != nil {
		return fmt.Errorf("failed to create scripts directory: %w", err)
	}

	// Create up migration file for users table
	upContent := g.generateUserTableUpMigration()
	if err := g.writeFile(upFilePath, upContent); err != nil {
		return fmt.Errorf("failed to create user table up migration: %w", err)
	}

	// Create down migration file for users table
	downContent := g.generateUserTableDownMigration()
	if err := g.writeFile(downFilePath, downContent); err != nil {
		return fmt.Errorf("failed to create user table down migration: %w", err)
	}

	g.logger.Infow("user table migration created successfully",
		"up_file", upFilePath,
		"down_file", downFilePath)

	return nil
}

// generateUserTableUpMigration generates the up migration for users table
func (g *Generator) generateUserTableUpMigration() string {
	return `-- Migration: Create users table
-- Created: Initial migration
-- Description: Create the users table with all necessary fields

CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_users_email (email),
    INDEX idx_users_status (status),
    INDEX idx_users_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`
}

// generateUserTableDownMigration generates the down migration for users table
func (g *Generator) generateUserTableDownMigration() string {
	return `-- Rollback Migration: Create users table
-- Created: Initial migration rollback
-- Description: Drop the users table

DROP TABLE IF EXISTS users;
`
}
