-- +goose Up
-- +goose StatementBegin
ALTER TABLE admin_telegram_bindings
    ADD COLUMN daily_summary_hour INT NOT NULL DEFAULT 9,
    ADD COLUMN weekly_summary_hour INT NOT NULL DEFAULT 9,
    ADD COLUMN weekly_summary_weekday INT NOT NULL DEFAULT 1,
    ADD COLUMN offline_check_interval_minutes INT NOT NULL DEFAULT 5;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE admin_telegram_bindings
    DROP COLUMN daily_summary_hour,
    DROP COLUMN weekly_summary_hour,
    DROP COLUMN weekly_summary_weekday,
    DROP COLUMN offline_check_interval_minutes;
-- +goose StatementEnd
