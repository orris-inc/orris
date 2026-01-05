-- +goose Up
-- Migration: Add sort_order to forward_agents
-- Created: 2025-01-05
-- Description: Add sort_order column for custom ordering of forward agents

-- +goose StatementBegin
ALTER TABLE forward_agents ADD COLUMN sort_order INT NOT NULL DEFAULT 0;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX idx_forward_agents_sort_order ON forward_agents(sort_order);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_forward_agents_sort_order ON forward_agents;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE forward_agents DROP COLUMN sort_order;
-- +goose StatementEnd
