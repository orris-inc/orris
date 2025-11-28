-- +goose Up
-- Migration: Add version field to subscription_plans for optimistic locking
-- Created: 2025-01-13
-- Description: Add version column to enable optimistic locking and prevent concurrent update conflicts
-- Reason: Prevent data loss when multiple users/processes update the same plan simultaneously

-- +goose StatementBegin
-- Add version column with default value 1
ALTER TABLE subscription_plans
ADD COLUMN version INT NOT NULL DEFAULT 1
COMMENT 'Version number for optimistic locking'
AFTER updated_at;
-- +goose StatementEnd

-- +goose StatementBegin
-- Initialize version=1 for all existing records (safety measure)
UPDATE subscription_plans
SET version = 1
WHERE version = 0 OR version IS NULL;
-- +goose StatementEnd

-- +goose Down
-- Rollback Migration: Remove version field
-- Description: Remove the version column from subscription_plans table

-- +goose StatementBegin
ALTER TABLE subscription_plans DROP COLUMN version;
-- +goose StatementEnd
