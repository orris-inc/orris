-- +goose Up
-- Migration: Optimize foreign key constraints
-- Created: 2025-11-12
-- Description: Optimize foreign key constraints for better flexibility with soft deletes
--              1. Change node_traffic.user_id from SET NULL to CASCADE
--              2. Remove foreign keys from node_group_plans (node association tables)
--              3. Remove foreign keys from node_group_nodes (node association tables)
--              4. Remove node_id foreign keys from traffic tables (soft delete compatibility)

-- Step 1: Drop existing foreign keys from node_traffic
ALTER TABLE node_traffic
DROP FOREIGN KEY node_traffic_ibfk_1,
DROP FOREIGN KEY node_traffic_ibfk_2;

-- Step 2: Re-add node_traffic.user_id foreign key with CASCADE
ALTER TABLE node_traffic
ADD CONSTRAINT fk_node_traffic_user
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Step 3: Drop foreign keys from node_group_nodes
ALTER TABLE node_group_nodes
DROP FOREIGN KEY node_group_nodes_ibfk_1,
DROP FOREIGN KEY node_group_nodes_ibfk_2;

-- Step 4: Drop foreign keys from node_group_plans
ALTER TABLE node_group_plans
DROP FOREIGN KEY node_group_plans_ibfk_1,
DROP FOREIGN KEY node_group_plans_ibfk_2;

-- Step 5: Drop node_id foreign key from user_traffic
ALTER TABLE user_traffic
DROP FOREIGN KEY user_traffic_ibfk_2;

-- +goose Down
-- Rollback Migration: Restore original foreign key constraints
-- Description: Restore the original foreign key constraints

-- Step 1: Restore node_traffic foreign keys
ALTER TABLE node_traffic
DROP FOREIGN KEY fk_node_traffic_user;

ALTER TABLE node_traffic
ADD CONSTRAINT node_traffic_ibfk_1
FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE,
ADD CONSTRAINT node_traffic_ibfk_2
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

-- Step 2: Restore node_group_nodes foreign keys
ALTER TABLE node_group_nodes
ADD CONSTRAINT node_group_nodes_ibfk_1
FOREIGN KEY (node_group_id) REFERENCES node_groups(id) ON DELETE CASCADE,
ADD CONSTRAINT node_group_nodes_ibfk_2
FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE;

-- Step 3: Restore node_group_plans foreign keys
ALTER TABLE node_group_plans
ADD CONSTRAINT node_group_plans_ibfk_1
FOREIGN KEY (node_group_id) REFERENCES node_groups(id) ON DELETE CASCADE,
ADD CONSTRAINT node_group_plans_ibfk_2
FOREIGN KEY (subscription_plan_id) REFERENCES subscription_plans(id) ON DELETE CASCADE;

-- Step 4: Restore user_traffic.node_id foreign key
ALTER TABLE user_traffic
ADD CONSTRAINT user_traffic_ibfk_2
FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE;
