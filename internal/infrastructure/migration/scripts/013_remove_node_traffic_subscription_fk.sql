-- +goose Up
-- Migration: Remove node_traffic.subscription_id foreign key
-- Created: 2025-11-12
-- Description: Remove the last remaining foreign key constraint from node_traffic table:
--              node_traffic.subscription_id -> subscriptions.id

-- Remove foreign key from node_traffic table (subscription_id)
SET @sql = IF((SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'node_traffic'
    AND CONSTRAINT_NAME = 'node_traffic_ibfk_3') > 0,
    'ALTER TABLE node_traffic DROP FOREIGN KEY node_traffic_ibfk_3',
    'SELECT "Constraint node_traffic_ibfk_3 does not exist, skipping" AS msg');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down
-- Rollback: Not reversible
-- Description: Foreign key constraints removal is intentionally not reversible.

SELECT 'Foreign key removal is not reversible - this is an architectural change' AS notice;
