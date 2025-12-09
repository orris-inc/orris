-- +goose Up

-- Add api_token column to nodes table for storing the plaintext token
-- This allows retrieving the current token without regenerating
ALTER TABLE nodes
    ADD COLUMN api_token VARCHAR(255) DEFAULT NULL COMMENT 'Plaintext API token for retrieval' AFTER token_hash;

-- +goose Down

ALTER TABLE nodes
    DROP COLUMN api_token;
