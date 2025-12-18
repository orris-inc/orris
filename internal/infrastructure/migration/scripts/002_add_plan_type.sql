-- +goose Up
-- Migration: Add plan_type column to plans table
-- Created: 2025-12-18
-- Description: Separates Node Plans from Forward Plans by adding a type discriminator

-- Add plan_type column with default value 'node'
ALTER TABLE plans ADD COLUMN plan_type VARCHAR(20) NOT NULL DEFAULT 'node' AFTER slug;

-- Add index for efficient filtering by plan type
ALTER TABLE plans ADD INDEX idx_plan_type (plan_type);

-- +goose Down
-- Rollback: Remove plan_type column and index

ALTER TABLE plans DROP INDEX idx_plan_type;
ALTER TABLE plans DROP COLUMN plan_type;
