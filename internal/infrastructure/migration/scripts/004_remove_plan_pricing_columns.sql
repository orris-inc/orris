-- +goose Up
-- Migration: Remove single pricing columns from plans table
-- Created: 2025-12-18
-- Description: Remove price, currency, and billing_cycle columns as pricing is now stored in plan_pricings table

-- Remove pricing columns from plans table
ALTER TABLE plans DROP COLUMN price;
ALTER TABLE plans DROP COLUMN currency;
ALTER TABLE plans DROP COLUMN billing_cycle;

-- +goose Down
-- Rollback: Re-add the removed columns

ALTER TABLE plans ADD COLUMN price BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER description;
ALTER TABLE plans ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'CNY' AFTER price;
ALTER TABLE plans ADD COLUMN billing_cycle VARCHAR(20) NOT NULL DEFAULT 'monthly' AFTER currency;
