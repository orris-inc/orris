-- +goose Up
-- +goose StatementBegin
ALTER TABLE forward_rules ADD COLUMN ip_version VARCHAR(10) NOT NULL DEFAULT 'auto';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE forward_rules DROP COLUMN ip_version;
-- +goose StatementEnd
