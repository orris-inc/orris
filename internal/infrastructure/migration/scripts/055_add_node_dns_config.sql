-- +goose Up
-- Add dns_config column to nodes table for DNS-based unlocking configuration
ALTER TABLE nodes ADD COLUMN dns_config JSON NULL COMMENT 'DNS configuration for DNS-based unlocking';

-- +goose Down
ALTER TABLE nodes DROP COLUMN dns_config;
