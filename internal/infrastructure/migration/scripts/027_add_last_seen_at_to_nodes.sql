-- +goose Up
-- Add last_seen_at column to nodes table for tracking node online status
-- This field is updated when agent reports status (throttled to every 2 minutes)

ALTER TABLE nodes ADD COLUMN last_seen_at TIMESTAMP NULL DEFAULT NULL;

CREATE INDEX idx_nodes_last_seen_at ON nodes(last_seen_at);

-- +goose Down
DROP INDEX idx_nodes_last_seen_at ON nodes;

ALTER TABLE nodes DROP COLUMN last_seen_at;
