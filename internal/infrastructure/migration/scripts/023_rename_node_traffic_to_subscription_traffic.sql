-- +goose Up
-- Migration: Rename node_traffic to subscription_traffic
-- Created: 2025-12-03
-- Description: Rename node_traffic table to subscription_traffic for better clarity
--              The table actually stores subscription-level traffic data, not node-level

-- Rename the table
RENAME TABLE node_traffic TO subscription_traffic;

-- +goose Down
-- Rollback Migration: Restore original table name
-- Description: Rename subscription_traffic back to node_traffic

RENAME TABLE subscription_traffic TO node_traffic;
