-- +goose Up
-- Migration: Phase 2 - Remove low-usage fields
-- Created: 2025-11-12
-- Description: Remove fields that have minimal usage or incorrect implementation
--              Medium risk - requires code changes in OAuth and notification handlers

-- ========================================
-- Part 1: Remove users.locale
-- Only set during OAuth login, no actual business usage
-- ========================================
ALTER TABLE users DROP COLUMN locale;

-- ========================================
-- Part 2: Remove announcements.view_count
-- Issues:
-- 1. Not concurrency-safe (race conditions)
-- 2. No analytics/statistics feature using this data
-- 3. Adds unnecessary DB write pressure
-- Recommendation: Migrate to Redis if view count is needed
-- ========================================
ALTER TABLE announcements DROP COLUMN view_count;

-- ========================================
-- Part 3: Remove notifications.archived_at
-- Redundant with GORM's deleted_at soft delete functionality
-- ========================================
ALTER TABLE notifications DROP COLUMN archived_at;

-- +goose Down
-- Rollback Migration: Restore removed fields
-- Note: Data will be lost and cannot be recovered

-- Restore users.locale
ALTER TABLE users ADD COLUMN locale VARCHAR(10) DEFAULT 'en' AFTER email_verified;

-- Restore announcements.view_count
ALTER TABLE announcements ADD COLUMN view_count INT DEFAULT 0 AFTER expires_at;

-- Restore notifications.archived_at
ALTER TABLE notifications ADD COLUMN archived_at TIMESTAMP NULL AFTER read_status;
