-- +goose Up
-- Migration: Add sort_order to forward_rules
-- Created: 2025-12-23
-- Description: Add sort_order column for custom ordering of forward rules

ALTER TABLE forward_rules ADD COLUMN sort_order INT NOT NULL DEFAULT 0;
CREATE INDEX idx_forward_rules_sort_order ON forward_rules(sort_order);

-- +goose Down
DROP INDEX idx_forward_rules_sort_order ON forward_rules;
ALTER TABLE forward_rules DROP COLUMN sort_order;
