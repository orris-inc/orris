-- +goose Up
-- Migration: Add missing indexes for node_group association tables
-- Created: 2025-12-18
-- Description: Add indexes to improve query performance for node group lookups

-- Index for querying which groups a node belongs to
-- The existing unique index (node_group_id, node_id) is optimized for group-based lookups
-- This index optimizes node-based lookups (e.g., "find all groups containing node X")
CREATE INDEX idx_node_group_nodes_node_id ON node_group_nodes(node_id);

-- Index for querying which groups are associated with a subscription plan
-- The existing unique index (node_group_id, subscription_plan_id) is optimized for group-based lookups
-- This index optimizes plan-based lookups (e.g., "find all groups for plan X")
CREATE INDEX idx_node_group_plans_plan_id ON node_group_plans(subscription_plan_id);

-- +goose Down
-- Rollback: Remove the indexes

DROP INDEX idx_node_group_nodes_node_id ON node_group_nodes;
DROP INDEX idx_node_group_plans_plan_id ON node_group_plans;
