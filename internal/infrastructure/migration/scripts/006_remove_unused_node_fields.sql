-- +goose Up
-- Migration: Remove unused fields from nodes table
-- Created: 2025-11-11
-- Description: Remove country and encryption_password fields from nodes table
--              - country: Not needed, following "less is more" principle
--              - encryption_password: Password should be subscription UUID, not stored in nodes table

-- Remove country field (MySQL syntax)
ALTER TABLE nodes DROP COLUMN country;

-- Remove encryption_password field
-- Note: Password for Shadowsocks is the subscription UUID, not stored at node level
ALTER TABLE nodes DROP COLUMN encryption_password;

-- +goose Down
-- Rollback Migration: Add back removed fields
-- Description: Restore country and encryption_password fields

-- Add back country field
ALTER TABLE nodes ADD COLUMN country VARCHAR(50) AFTER status;

-- Add back encryption_password field
ALTER TABLE nodes ADD COLUMN encryption_password VARCHAR(255) NOT NULL DEFAULT '' AFTER encryption_method;
