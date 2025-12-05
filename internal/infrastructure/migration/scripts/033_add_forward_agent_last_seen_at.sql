-- +goose Up

-- Add last_seen_at field to forward_agents table
-- This field tracks the last time the agent reported status to the server
ALTER TABLE forward_agents ADD COLUMN last_seen_at DATETIME NULL DEFAULT NULL
    COMMENT 'Last time the agent reported status to the server';

CREATE INDEX idx_forward_agent_last_seen_at ON forward_agents(last_seen_at);

-- +goose Down

DROP INDEX idx_forward_agent_last_seen_at ON forward_agents;
ALTER TABLE forward_agents DROP COLUMN last_seen_at;
