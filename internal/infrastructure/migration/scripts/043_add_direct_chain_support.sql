-- +goose Up
-- Add chain_port_config field for direct_chain-type forward rules
-- This field stores a map of agent_id -> listen_port for each agent in the chain
-- Format: JSON object with agent IDs as keys, e.g., {"2": 10001, "5": 10002}
ALTER TABLE forward_rules ADD COLUMN chain_port_config JSON DEFAULT NULL
    COMMENT 'Map of agent_id to listen port for direct_chain forwarding (JSON object)';

-- +goose Down
ALTER TABLE forward_rules DROP COLUMN chain_port_config;
