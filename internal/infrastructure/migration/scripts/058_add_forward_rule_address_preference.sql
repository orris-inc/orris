-- +goose Up
ALTER TABLE forward_rules ADD COLUMN address_preference VARCHAR(10) NOT NULL DEFAULT 'auto';

-- +goose Down
ALTER TABLE forward_rules DROP COLUMN address_preference;
