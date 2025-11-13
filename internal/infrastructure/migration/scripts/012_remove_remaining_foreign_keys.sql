-- +goose Up
-- Migration: Remove remaining foreign key constraints
-- Created: 2025-11-12
-- Description: Remove the last two foreign key constraints that were missed in 011 migration:
--              1. user_traffic.subscription_id -> subscriptions.id
--              2. subscription_plan_pricing.plan_id -> subscription_plans.id

-- Step 1: Remove foreign key from user_traffic table (subscription_id)
SET @sql = IF((SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_traffic'
    AND CONSTRAINT_NAME = 'user_traffic_ibfk_3') > 0,
    'ALTER TABLE user_traffic DROP FOREIGN KEY user_traffic_ibfk_3',
    'SELECT "Constraint user_traffic_ibfk_3 does not exist, skipping" AS msg');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 2: Remove foreign key from subscription_plan_pricing table (plan_id)
SET @sql = IF((SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'subscription_plan_pricing'
    AND CONSTRAINT_NAME = 'fk_pricing_plan_id') > 0,
    'ALTER TABLE subscription_plan_pricing DROP FOREIGN KEY fk_pricing_plan_id',
    'SELECT "Constraint fk_pricing_plan_id does not exist, skipping" AS msg');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down
-- Rollback: Not reversible
-- Description: Foreign key constraints removal is intentionally not reversible.
--              This is a fundamental architectural change where data integrity
--              is managed by application logic rather than database constraints.

SELECT 'Foreign key removal is not reversible - this is an architectural change' AS notice;
