-- +goose Up
ALTER TABLE sessions ADD COLUMN remember_me BOOLEAN NOT NULL DEFAULT TRUE;

-- +goose Down
ALTER TABLE sessions DROP COLUMN remember_me;
