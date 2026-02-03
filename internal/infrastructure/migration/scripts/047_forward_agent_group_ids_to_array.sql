-- +goose Up
-- Migration: Convert forward_agents.group_id to group_ids JSON array
-- Description: Support forward agents belonging to multiple resource groups

-- Step 1: Add new group_ids JSON column
ALTER TABLE forward_agents ADD COLUMN group_ids JSON DEFAULT NULL;

-- Step 2: Migrate existing data from group_id to group_ids
UPDATE forward_agents SET group_ids = JSON_ARRAY(group_id) WHERE group_id IS NOT NULL;

-- Step 3: Drop the old group_id column and its index
DROP INDEX idx_forward_agent_group_id ON forward_agents;
ALTER TABLE forward_agents DROP COLUMN group_id;

-- +goose Down
-- Rollback: Convert group_ids JSON array back to single group_id

-- Step 1: Add back group_id column
ALTER TABLE forward_agents ADD COLUMN group_id BIGINT UNSIGNED NULL;

-- Step 2: Migrate data back (take first element from JSON array)
UPDATE forward_agents SET group_id = JSON_EXTRACT(group_ids, '$[0]') WHERE group_ids IS NOT NULL AND JSON_LENGTH(group_ids) > 0;

-- Step 3: Drop group_ids column
ALTER TABLE forward_agents DROP COLUMN group_ids;

-- Step 4: Recreate index
CREATE INDEX idx_forward_agent_group_id ON forward_agents (group_id);
