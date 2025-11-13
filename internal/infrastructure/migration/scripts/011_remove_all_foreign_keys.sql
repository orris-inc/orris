-- +goose Up
-- Migration: Remove all remaining foreign key constraints
-- Created: 2025-11-12
-- Description: Complete removal of all foreign key constraints from the database.
--              All data relationships will be managed entirely by application business logic.
--              This supports soft delete strategy and provides maximum flexibility.
--              Note: Some DROP statements may fail if the constraint doesn't exist - this is expected.

-- Step 1: Remove foreign key from node_traffic table (added in 010 migration)
SET @sql = IF((SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'node_traffic'
    AND CONSTRAINT_NAME = 'fk_node_traffic_user') > 0,
    'ALTER TABLE node_traffic DROP FOREIGN KEY fk_node_traffic_user',
    'SELECT "Constraint fk_node_traffic_user does not exist, skipping" AS msg');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 2: Remove foreign key from user_traffic table
SET @sql = IF((SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_traffic'
    AND CONSTRAINT_NAME = 'user_traffic_ibfk_1') > 0,
    'ALTER TABLE user_traffic DROP FOREIGN KEY user_traffic_ibfk_1',
    'SELECT "Constraint user_traffic_ibfk_1 does not exist, skipping" AS msg');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 3: Remove foreign key from subscription_plan_pricing table
SET @sql = IF((SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'subscription_plan_pricing'
    AND CONSTRAINT_NAME = 'subscription_plan_pricing_ibfk_1') > 0,
    'ALTER TABLE subscription_plan_pricing DROP FOREIGN KEY subscription_plan_pricing_ibfk_1',
    'SELECT "Constraint subscription_plan_pricing_ibfk_1 does not exist, skipping" AS msg');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Note: Other tables (subscriptions, notifications, etc.) were never defined with foreign keys
-- in the original migration scripts, so no cleanup is needed for them.

-- +goose Down
-- Rollback: Not reversible
-- Description: Foreign key constraints removal is intentionally not reversible.
--              This migration represents a fundamental architectural change where
--              data integrity is managed by application logic rather than database constraints.
--              Restoring foreign keys would require careful analysis of existing data integrity.

SELECT 'Foreign key removal is not reversible - this is an architectural change' AS notice;
