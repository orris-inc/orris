-- +goose Up
-- Migration: Remove trial_days column from plans table
-- Created: 2025-12-20
-- Description: Remove the trial_days field as trial functionality is no longer needed

ALTER TABLE plans DROP COLUMN trial_days;

-- +goose Down
-- Restore trial_days column
ALTER TABLE plans ADD COLUMN trial_days INT DEFAULT 0 AFTER description;
