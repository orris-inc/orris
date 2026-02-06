-- +goose Up
ALTER TABLE telegram_bindings ADD COLUMN language VARCHAR(5) NOT NULL DEFAULT 'zh';
ALTER TABLE admin_telegram_bindings ADD COLUMN language VARCHAR(5) NOT NULL DEFAULT 'zh';

-- +goose Down
ALTER TABLE telegram_bindings DROP COLUMN language;
ALTER TABLE admin_telegram_bindings DROP COLUMN language;
