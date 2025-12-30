-- +goose Up
-- Add route_config column to nodes table for traffic splitting configuration
ALTER TABLE nodes ADD COLUMN route_config JSON NULL COMMENT 'Routing configuration for traffic splitting (sing-box compatible)';

-- +goose Down
ALTER TABLE nodes DROP COLUMN route_config;
