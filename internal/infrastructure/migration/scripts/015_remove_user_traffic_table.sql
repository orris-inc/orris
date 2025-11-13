-- +goose Up
-- Migration: Remove user_traffic table
-- Created: 2025-11-12
-- Description: Remove user_traffic table as it's redundant.
--              Traffic tracking is now purely subscription-based via node_traffic table.
--              The UID field in XrayR traffic reports now represents subscription_id instead of user_id.
--
-- Background:
--   1. user_traffic table was redundant because users can have multiple subscriptions
--   2. XrayR node reports traffic using UID which should be subscription_id, not user_id
--   3. node_traffic table already contains subscription_id and can satisfy all traffic tracking needs
--
-- Changes:
--   - Drop user_traffic table entirely
--   - Traffic is now tracked only in node_traffic table with subscription_id
--   - User's total traffic can be calculated by aggregating subscriptions via JOIN with subscriptions table
--
-- Impact:
--   - BREAKING: Any code querying user_traffic table will fail
--   - Data migration: Historical user_traffic data is not migrated (archived if needed)
--   - Traffic reports from XrayR now directly use subscription_id as UID

-- Drop user_traffic table
DROP TABLE IF EXISTS user_traffic;

-- +goose Down
-- Rollback: Not reversible without data backup
-- Description: This migration is intentionally not reversible.
--              Recreating user_traffic table would require:
--              1. Restoring table schema
--              2. Restoring historical data from backup
--              3. Re-implementing traffic aggregation logic
--
-- If rollback is needed:
-- 1. Restore database from backup taken before this migration
-- 2. Revert code changes to use UserTrafficRepository
-- 3. Update XrayR DTO to use user_id instead of subscription_id

-- Rollback not implemented - restore from backup if needed
SELECT 'Migration 015 is not reversible - restore from backup if rollback needed' AS warning;
