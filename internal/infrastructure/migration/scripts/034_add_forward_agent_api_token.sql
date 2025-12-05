-- +goose Up

-- Add api_token column to forward_agents table for storing the plaintext token
-- This allows retrieving the current token without regenerating
ALTER TABLE forward_agents
    ADD COLUMN api_token VARCHAR(255) DEFAULT NULL COMMENT 'Plaintext API token for retrieval' AFTER token_hash;

-- +goose Down

ALTER TABLE forward_agents
    DROP COLUMN api_token;
