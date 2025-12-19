-- +goose Up
-- Migration: Redesign entitlement system for user-resource authorization
-- Created: 2025-12-19
-- Description: Transform entitlements from plan-resource to user-resource mapping,
--              and add subject tracking to subscriptions table

-- ============================================================================
-- Step 1: Create new entitlements table with redesigned structure
-- ============================================================================

CREATE TABLE entitlements_new (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    subject_type    VARCHAR(20) NOT NULL DEFAULT 'user',
    subject_id      BIGINT UNSIGNED NOT NULL,
    resource_type   VARCHAR(30) NOT NULL,
    resource_id     BIGINT UNSIGNED NOT NULL,
    source_type     VARCHAR(20) NOT NULL,
    source_id       BIGINT UNSIGNED NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    expires_at      TIMESTAMP NULL,
    metadata        JSON,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    version         INT NOT NULL DEFAULT 1,

    UNIQUE INDEX idx_unique_entitlement (subject_type, subject_id, resource_type, resource_id, source_type, source_id),
    INDEX idx_subject (subject_type, subject_id),
    INDEX idx_resource (resource_type, resource_id),
    INDEX idx_source (source_type, source_id),
    INDEX idx_status_expires (status, expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ============================================================================
-- Step 2: Modify subscriptions table to add subject tracking
-- ============================================================================

-- Add subject_type column with default value
ALTER TABLE subscriptions
ADD COLUMN subject_type VARCHAR(20) NOT NULL DEFAULT 'user' AFTER plan_id;

-- Add subject_id column (nullable initially for data migration)
ALTER TABLE subscriptions
ADD COLUMN subject_id BIGINT UNSIGNED NULL AFTER subject_type;

-- Migrate data: set subject_id to user_id for all existing subscriptions
UPDATE subscriptions
SET subject_id = user_id
WHERE subject_id IS NULL;

-- Make subject_id NOT NULL after data migration
ALTER TABLE subscriptions
MODIFY COLUMN subject_id BIGINT UNSIGNED NOT NULL;

-- Add index for subject lookups
ALTER TABLE subscriptions
ADD INDEX idx_subject (subject_type, subject_id);

-- ============================================================================
-- Step 3: Migrate entitlement data from plan-resource to user-resource
-- ============================================================================

-- Insert user-resource entitlements based on active subscriptions and plan entitlements
-- This creates entitlements for users based on their subscription's plan
INSERT INTO entitlements_new (
    subject_type,
    subject_id,
    resource_type,
    resource_id,
    source_type,
    source_id,
    status,
    expires_at,
    created_at,
    updated_at
)
SELECT DISTINCT
    'user' as subject_type,
    s.user_id as subject_id,
    e.resource_type,
    e.resource_id,
    'subscription' as source_type,
    s.id as source_id,
    CASE
        WHEN s.status = 'active' AND s.end_date > NOW() THEN 'active'
        WHEN s.status = 'expired' OR s.end_date <= NOW() THEN 'expired'
        WHEN s.status = 'cancelled' THEN 'revoked'
        ELSE 'inactive'
    END as status,
    s.end_date as expires_at,
    NOW() as created_at,
    NOW() as updated_at
FROM subscriptions s
INNER JOIN entitlements e ON s.plan_id = e.plan_id
WHERE s.deleted_at IS NULL;

-- ============================================================================
-- Step 4: Archive old entitlements table and activate new one
-- ============================================================================

-- Rename old entitlements table for backup
RENAME TABLE entitlements TO entitlements_legacy;

-- Activate new entitlements table
RENAME TABLE entitlements_new TO entitlements;

-- +goose Down
-- Rollback Migration: Restore original entitlement structure
-- Description: Revert entitlements to plan-resource mapping and remove subject tracking from subscriptions

-- ============================================================================
-- Step 1: Restore original entitlements table
-- ============================================================================

-- Recreate original entitlements table structure
CREATE TABLE entitlements_restored (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    plan_id BIGINT UNSIGNED NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_plan_resource (plan_id, resource_type, resource_id),
    INDEX idx_plan_id (plan_id),
    INDEX idx_resource (resource_type, resource_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Restore data from legacy table if it exists
INSERT INTO entitlements_restored (id, plan_id, resource_type, resource_id, created_at)
SELECT id, plan_id, resource_type, resource_id, created_at
FROM entitlements_legacy;

-- Replace current entitlements with restored version
DROP TABLE entitlements;
RENAME TABLE entitlements_restored TO entitlements;

-- Drop legacy backup table
DROP TABLE IF EXISTS entitlements_legacy;

-- ============================================================================
-- Step 2: Remove subject tracking from subscriptions table
-- ============================================================================

-- Remove subject index
ALTER TABLE subscriptions DROP INDEX idx_subject;

-- Remove subject columns
ALTER TABLE subscriptions DROP COLUMN subject_id;
ALTER TABLE subscriptions DROP COLUMN subject_type;
