-- +goose Up
-- Add chain_agent_ids field for chain-type forward rules
-- This field stores an ordered array of agent IDs representing the forwarding chain
-- Format: JSON array of unsigned integers, e.g., [2, 5, 8]
-- For chain type: client -> agent_id -> chain_agent_ids[0] -> chain_agent_ids[1] -> ... -> target
ALTER TABLE forward_rules ADD COLUMN chain_agent_ids JSON DEFAULT NULL
    COMMENT 'Ordered array of intermediate agent IDs for chain forwarding (JSON array of uint)';

-- +goose Down
ALTER TABLE forward_rules DROP COLUMN chain_agent_ids;
