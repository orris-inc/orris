-- +goose Up
-- Migration: Remove storage-related fields from subscription system
-- Created: 2025-01-13
-- Description: Remove StorageLimit and StorageUsed fields as subscription system only needs traffic and time limits
-- Reason: Simplify subscription model - only track traffic limits and time periods, not storage

-- Remove storage_limit from subscription_plans table
-- Original: storage_limit BIGINT UNSIGNED DEFAULT 1073741824 (defined in 002_subscription_tables.sql line 22)
ALTER TABLE subscription_plans DROP COLUMN storage_limit;

-- Remove storage_used from subscription_usages table
-- Original: storage_used BIGINT UNSIGNED NOT NULL DEFAULT 0 (defined in 002_subscription_tables.sql line 112)
ALTER TABLE subscription_usages DROP COLUMN storage_used;

-- +goose Down
-- Rollback Migration: Add back storage fields
-- Description: Restore storage_limit and storage_used fields to their original state

-- Add back storage_limit to subscription_plans (after max_projects, before is_public)
ALTER TABLE subscription_plans
ADD COLUMN storage_limit BIGINT UNSIGNED DEFAULT 1073741824
AFTER max_projects;

-- Add back storage_used to subscription_usages (after api_data_in, before users_count)
ALTER TABLE subscription_usages
ADD COLUMN storage_used BIGINT UNSIGNED NOT NULL DEFAULT 0
AFTER api_data_in;
