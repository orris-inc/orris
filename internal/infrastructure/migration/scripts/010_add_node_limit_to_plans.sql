-- +goose Up
-- +goose StatementBegin
ALTER TABLE plans ADD COLUMN node_limit INT NULL COMMENT 'Maximum number of user nodes (NULL or 0 = unlimited)';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE plans DROP COLUMN node_limit;
-- +goose StatementEnd
