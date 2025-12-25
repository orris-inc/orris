-- +goose Up
-- Migration: Add tunnel_hops to forward_rules
-- Created: 2025-12-25
-- Description: Add tunnel_hops column for hybrid chain support (first N hops use tunnel, rest use direct)

ALTER TABLE forward_rules ADD COLUMN tunnel_hops INT NULL;

-- +goose Down
ALTER TABLE forward_rules DROP COLUMN tunnel_hops;
