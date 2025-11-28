-- +goose Up
-- Migration: Add UUID field to subscriptions table
-- Created: 2025-01-19
-- Description: Add UUID column for node authentication and unique identification
-- Reason: Subscriptions need a unique identifier for secure node authentication

-- +goose StatementBegin
-- Step 1: Add uuid column as nullable first to handle existing data
ALTER TABLE subscriptions
ADD COLUMN uuid VARCHAR(36) NULL
COMMENT 'unique identifier used for node authentication'
AFTER id;
-- +goose StatementEnd

-- +goose StatementBegin
-- Step 2: Generate UUIDs for existing records (if any)
UPDATE subscriptions
SET uuid = UUID()
WHERE uuid IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- Step 3: Make uuid NOT NULL and add unique constraint
ALTER TABLE subscriptions
MODIFY COLUMN uuid VARCHAR(36) NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- Step 4: Add unique index on uuid
CREATE UNIQUE INDEX idx_uuid ON subscriptions(uuid);
-- +goose StatementEnd

-- +goose Down
-- Rollback Migration: Remove UUID field from subscriptions table
-- Description: Remove the uuid column and its index

-- +goose StatementBegin
DROP INDEX idx_uuid ON subscriptions;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE subscriptions DROP COLUMN uuid;
-- +goose StatementEnd
