-- +goose Up
-- +goose StatementBegin

-- Add agent_id column to forward_rules table
ALTER TABLE forward_rules
    ADD COLUMN agent_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Forward agent ID that executes this rule' AFTER id;

-- Add next_agent_id column for chain forwarding
ALTER TABLE forward_rules
    ADD COLUMN next_agent_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Next agent ID for chain forward (0=direct forward to target)' AFTER agent_id;

-- Make target_address and target_port nullable for chain forward
ALTER TABLE forward_rules
    MODIFY COLUMN target_address VARCHAR(255) DEFAULT '' COMMENT 'Target address (required when next_agent_id=0)',
    MODIFY COLUMN target_port SMALLINT UNSIGNED DEFAULT 0 COMMENT 'Target port (required when next_agent_id=0)';

-- Drop old unique index on listen_port
DROP INDEX idx_listen_port ON forward_rules;

-- Create new unique index on (listen_port, agent_id) to allow same port on different agents
CREATE UNIQUE INDEX idx_listen_port_agent ON forward_rules (listen_port, agent_id);

-- Create index on agent_id for faster lookups
CREATE INDEX idx_forward_agent_id ON forward_rules (agent_id);

-- Create index on next_agent_id for chain forward lookups
CREATE INDEX idx_forward_next_agent_id ON forward_rules (next_agent_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove next_agent_id index
DROP INDEX idx_forward_next_agent_id ON forward_rules;

-- Remove agent_id index
DROP INDEX idx_forward_agent_id ON forward_rules;

-- Remove composite unique index
DROP INDEX idx_listen_port_agent ON forward_rules;

-- Restore original unique index on listen_port
CREATE UNIQUE INDEX idx_listen_port ON forward_rules (listen_port);

-- Restore target_address and target_port as NOT NULL
ALTER TABLE forward_rules
    MODIFY COLUMN target_address VARCHAR(255) NOT NULL COMMENT 'Target address (IP or domain)',
    MODIFY COLUMN target_port SMALLINT UNSIGNED NOT NULL COMMENT 'Target port';

-- Remove next_agent_id column
ALTER TABLE forward_rules DROP COLUMN next_agent_id;

-- Remove agent_id column
ALTER TABLE forward_rules DROP COLUMN agent_id;

-- +goose StatementEnd
