-- +goose Up
-- Migration: Remove deprecated plan_ids columns from nodes, forward_agents, forward_rules
-- Description: Complete migration from planIDs to Resource Group system

-- Remove plan_ids column from nodes table
ALTER TABLE nodes DROP COLUMN plan_ids;

-- Remove plan_ids column from forward_agents table
ALTER TABLE forward_agents DROP COLUMN plan_ids;

-- Remove plan_ids column from forward_rules table
ALTER TABLE forward_rules DROP COLUMN plan_ids;

-- +goose Down
-- Rollback: Re-add plan_ids columns (empty JSON arrays)

ALTER TABLE nodes ADD COLUMN plan_ids JSON DEFAULT NULL;
ALTER TABLE forward_agents ADD COLUMN plan_ids JSON DEFAULT NULL;
ALTER TABLE forward_rules ADD COLUMN plan_ids JSON DEFAULT NULL;
