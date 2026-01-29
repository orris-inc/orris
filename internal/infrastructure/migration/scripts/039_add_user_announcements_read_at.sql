-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN announcements_read_at TIMESTAMP NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN announcements_read_at;
-- +goose StatementEnd
