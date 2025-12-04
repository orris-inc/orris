-- +goose Up

-- Add rule_type field to forward_rules table
-- RuleType: direct (forward to target directly), chain (forward via exit agent), websocket (forward via websocket)
ALTER TABLE forward_rules ADD COLUMN rule_type VARCHAR(20) NOT NULL DEFAULT 'direct'
    COMMENT 'Rule type: direct, chain, websocket';

-- Add exit_agent_id field for chain/websocket forwarding
ALTER TABLE forward_rules ADD COLUMN exit_agent_id BIGINT UNSIGNED NULL DEFAULT NULL
    COMMENT 'Exit agent ID for chain/websocket forward (nullable)';

-- Add ws_listen_port field for websocket forwarding
ALTER TABLE forward_rules ADD COLUMN ws_listen_port SMALLINT UNSIGNED NULL DEFAULT NULL
    COMMENT 'Websocket listen port (nullable, used for websocket type)';

-- Add index for exit_agent_id
CREATE INDEX idx_forward_exit_agent_id ON forward_rules(exit_agent_id);

-- Drop the old next_agent_id index
DROP INDEX idx_forward_next_agent_id ON forward_rules;

-- Remove next_agent_id field
ALTER TABLE forward_rules DROP COLUMN next_agent_id;

-- Add public_address field to forward_agents table
ALTER TABLE forward_agents ADD COLUMN public_address VARCHAR(255) NULL DEFAULT NULL
    COMMENT 'Public address for agent access (nullable)';

-- +goose Down

-- Restore next_agent_id field
ALTER TABLE forward_agents DROP COLUMN public_address;

ALTER TABLE forward_rules ADD COLUMN next_agent_id BIGINT UNSIGNED NOT NULL DEFAULT 0
    COMMENT 'Next agent ID for chain forward (0=direct forward to target)';

CREATE INDEX idx_forward_next_agent_id ON forward_rules(next_agent_id);

DROP INDEX idx_forward_exit_agent_id ON forward_rules;

ALTER TABLE forward_rules DROP COLUMN ws_listen_port;

ALTER TABLE forward_rules DROP COLUMN exit_agent_id;

ALTER TABLE forward_rules DROP COLUMN rule_type;
