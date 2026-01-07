-- +goose Up
-- Fix unique index to only apply to non-deleted records.
-- The old index (agent_id, listen_port) blocks new rules from reusing ports of deleted rules.
-- Solution: Use a virtual column that returns NULL for deleted records (NULL is ignored in unique indexes).

-- Step 1: Drop the old unique index
DROP INDEX idx_listen_port_agent ON forward_rules;

-- Step 2: Add virtual column that returns listen_port only for active records
ALTER TABLE forward_rules ADD COLUMN active_listen_port SMALLINT UNSIGNED
    GENERATED ALWAYS AS (IF(deleted_at IS NULL, listen_port, NULL)) VIRTUAL;

-- Step 3: Create new unique index on (agent_id, active_listen_port)
-- NULL values are not considered duplicates in MySQL unique indexes
CREATE UNIQUE INDEX idx_listen_port_agent ON forward_rules (agent_id, active_listen_port);

-- +goose Down
-- Restore original index structure
DROP INDEX idx_listen_port_agent ON forward_rules;
ALTER TABLE forward_rules DROP COLUMN active_listen_port;
CREATE UNIQUE INDEX idx_listen_port_agent ON forward_rules (agent_id, listen_port);
