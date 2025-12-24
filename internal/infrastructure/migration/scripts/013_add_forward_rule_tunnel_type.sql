-- +goose Up
-- Migration: Add tunnel_type to forward_rules
-- Created: 2025-12-24
-- Description: Add tunnel_type column for selecting WS or TLS tunnel

ALTER TABLE forward_rules ADD COLUMN tunnel_type VARCHAR(10) NOT NULL DEFAULT 'ws';

-- +goose Down
ALTER TABLE forward_rules DROP COLUMN tunnel_type;
