-- +goose Up
ALTER TABLE forward_rules ADD COLUMN route_config JSON NULL;

-- +goose Down
ALTER TABLE forward_rules DROP COLUMN route_config;
