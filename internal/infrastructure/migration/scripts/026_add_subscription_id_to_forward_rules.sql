-- +goose Up
-- Migration: Add subscription_id to forward_rules
-- Created: 2026-01-07
-- Description: Add subscription_id column to forward_rules table for quota management

-- Add subscription_id column (nullable, admin rules don't bind to subscription)
ALTER TABLE forward_rules ADD COLUMN subscription_id BIGINT UNSIGNED NULL AFTER user_id;

-- Add index to support query by subscription
CREATE INDEX idx_forward_rules_subscription_id ON forward_rules(subscription_id);

-- +goose Down
DROP INDEX idx_forward_rules_subscription_id ON forward_rules;
ALTER TABLE forward_rules DROP COLUMN subscription_id;
