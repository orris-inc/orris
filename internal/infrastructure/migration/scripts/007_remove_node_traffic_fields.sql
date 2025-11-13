-- +goose Up
-- Migration: Remove traffic-related fields from nodes table
-- Created: 2025-11-12
-- Description: Remove traffic management fields from nodes table
--              Traffic tracking should be handled at subscription level, not node level
--              Following "separation of concerns" and "less is more" principles
--              Fields to remove:
--              - max_users: User limit should be at subscription plan level
--              - traffic_limit: Traffic limit should be at subscription level
--              - traffic_used: Traffic usage should be tracked via node_traffic table
--              - traffic_reset_at: Traffic reset should be managed at subscription level

-- Remove max_users field (MySQL syntax)
-- User limits should be enforced at subscription plan level
ALTER TABLE nodes DROP COLUMN max_users;

-- Remove traffic_limit field
-- Traffic limits are managed at subscription level
ALTER TABLE nodes DROP COLUMN traffic_limit;

-- Remove traffic_used field
-- Traffic usage is tracked via node_traffic and user_traffic tables
ALTER TABLE nodes DROP COLUMN traffic_used;

-- Remove traffic_reset_at field
-- Traffic reset cycles are managed at subscription level
ALTER TABLE nodes DROP COLUMN traffic_reset_at;

-- +goose Down
-- Rollback Migration: Restore traffic-related fields
-- Description: Add back traffic management fields to nodes table
--              Note: Data cannot be recovered, fields will be empty after rollback

-- Restore max_users field
-- Default 0 means unlimited users
ALTER TABLE nodes ADD COLUMN max_users INT UNSIGNED NOT NULL DEFAULT 0;

-- Restore traffic_limit field
-- Default 0 means unlimited traffic
ALTER TABLE nodes ADD COLUMN traffic_limit BIGINT UNSIGNED NOT NULL DEFAULT 0;

-- Restore traffic_used field
-- Starts at 0, should be recalculated from traffic tables
ALTER TABLE nodes ADD COLUMN traffic_used BIGINT UNSIGNED NOT NULL DEFAULT 0;

-- Restore traffic_reset_at field
-- NULL means no reset schedule
ALTER TABLE nodes ADD COLUMN traffic_reset_at TIMESTAMP NULL;
