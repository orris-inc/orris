-- +goose Up
-- Update unique index to include server_address for external rules
-- This allows external rules with different server addresses to use the same listen port

-- Step 1: Drop old unique index
DROP INDEX idx_listen_port_agent ON forward_rules;

-- Step 2: Create new unique index including server_address
-- For non-external rules: server_address is NULL, uniqueness based on (agent_id, listen_port)
-- For external rules: agent_id is 0, uniqueness based on (agent_id, listen_port, server_address)
CREATE UNIQUE INDEX idx_listen_port_agent_server ON forward_rules(agent_id, listen_port, server_address);

-- +goose Down
-- Restore old unique index

-- Step 1: Drop new unique index
DROP INDEX idx_listen_port_agent_server ON forward_rules;

-- Step 2: Recreate old unique index
CREATE UNIQUE INDEX idx_listen_port_agent ON forward_rules(agent_id, listen_port);
