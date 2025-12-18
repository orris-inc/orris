-- +goose Up
-- Migration: Remove unused columns from plans table
-- Created: 2025-12-18
-- Description: Remove custom_endpoint, storage_limit, and features columns that are not used in the codebase

-- Remove unused columns
ALTER TABLE plans DROP COLUMN custom_endpoint;
ALTER TABLE plans DROP COLUMN storage_limit;
ALTER TABLE plans DROP COLUMN features;

-- +goose Down
-- Rollback: Re-add the removed columns

ALTER TABLE plans ADD COLUMN custom_endpoint VARCHAR(200) AFTER limits;
ALTER TABLE plans ADD COLUMN storage_limit BIGINT UNSIGNED DEFAULT 1073741824 AFTER max_projects;
ALTER TABLE plans ADD COLUMN features JSON AFTER status;
