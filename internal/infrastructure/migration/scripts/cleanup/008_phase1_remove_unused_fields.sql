-- +goose Up
-- Migration: Phase 1 - Remove completely unused fields and tables
-- Created: 2025-11-12
-- Description: Remove subscription_histories table and unused subscription_usages fields
--              This is a zero-risk migration as these fields/tables have no code references

-- ========================================
-- Part 1: Remove subscription_histories table (never implemented)
-- ========================================
DROP TABLE IF EXISTS subscription_histories;

-- ========================================
-- Part 2: Clean up subscription_usages table
-- Remove 7 fields that only have domain layer getters/setters but no actual business usage
-- ========================================

-- Remove API tracking fields (not used in proxy node business)
ALTER TABLE subscription_usages DROP COLUMN api_requests;
ALTER TABLE subscription_usages DROP COLUMN api_data_out;
ALTER TABLE subscription_usages DROP COLUMN api_data_in;

-- Remove webhook and email tracking (no webhook/email features implemented)
ALTER TABLE subscription_usages DROP COLUMN webhook_calls;
ALTER TABLE subscription_usages DROP COLUMN emails_sent;

-- Remove report generation tracking (no report feature)
ALTER TABLE subscription_usages DROP COLUMN reports_generated;

-- Remove project count (no "project" concept in this application)
ALTER TABLE subscription_usages DROP COLUMN projects_count;

-- ========================================
-- Part 3: Remove custom_endpoint from subscription_plans
-- This field was reserved for future features but never implemented
-- ========================================
ALTER TABLE subscription_plans DROP COLUMN custom_endpoint;

-- +goose Down
-- Rollback Migration: Restore removed fields and tables
-- Note: Data will be lost and cannot be recovered

-- Restore subscription_histories table
CREATE TABLE subscription_histories (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    subscription_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    plan_id BIGINT UNSIGNED NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_status VARCHAR(20),
    new_status VARCHAR(20) NOT NULL,
    old_plan_id BIGINT UNSIGNED,
    new_plan_id BIGINT UNSIGNED,
    amount BIGINT UNSIGNED,
    currency VARCHAR(3),
    reason VARCHAR(500),
    performed_by BIGINT UNSIGNED,
    ip_address VARCHAR(45),
    user_agent VARCHAR(255),
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_subscription_history (subscription_id),
    INDEX idx_user_history (user_id),
    INDEX idx_action (action),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Restore subscription_usages fields
ALTER TABLE subscription_usages 
  ADD COLUMN api_requests BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN api_data_out BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN api_data_in BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN webhook_calls BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN emails_sent BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN reports_generated INT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN projects_count INT UNSIGNED NOT NULL DEFAULT 0;

-- Restore custom_endpoint
ALTER TABLE subscription_plans ADD COLUMN custom_endpoint VARCHAR(200);
