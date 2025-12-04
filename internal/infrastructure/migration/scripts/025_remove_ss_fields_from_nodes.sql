-- +goose Up
-- Remove Shadowsocks-specific fields from nodes table
-- These fields are now stored in shadowsocks_configs table

-- This migration should run after 024_create_shadowsocks_configs.sql

ALTER TABLE nodes DROP COLUMN encryption_method;
ALTER TABLE nodes DROP COLUMN plugin;
ALTER TABLE nodes DROP COLUMN plugin_opts;
ALTER TABLE nodes DROP COLUMN custom_fields;

-- +goose Down
-- Restore the removed columns (data will be empty)
ALTER TABLE nodes ADD COLUMN encryption_method VARCHAR(50) NOT NULL DEFAULT 'aes-256-gcm';
ALTER TABLE nodes ADD COLUMN plugin VARCHAR(100);
ALTER TABLE nodes ADD COLUMN plugin_opts JSON;
ALTER TABLE nodes ADD COLUMN custom_fields JSON;
