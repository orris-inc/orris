-- +goose Up

-- Add target_node_id field to forward_rules table
-- This field is mutually exclusive with target_address/target_port
ALTER TABLE forward_rules ADD COLUMN target_node_id BIGINT UNSIGNED NULL DEFAULT NULL
    COMMENT 'Target Node ID for dynamic address resolution (mutually exclusive with target_address/target_port)';

CREATE INDEX idx_forward_target_node_id ON forward_rules(target_node_id);

-- +goose Down

DROP INDEX idx_forward_target_node_id ON forward_rules;
ALTER TABLE forward_rules DROP COLUMN target_node_id;
