-- +goose Up
-- +goose StatementBegin
DROP PROCEDURE IF EXISTS add_schedule_config_fields;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE PROCEDURE add_schedule_config_fields()
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'admin_telegram_bindings' AND column_name = 'daily_summary_hour') THEN
        ALTER TABLE admin_telegram_bindings ADD COLUMN daily_summary_hour INT NOT NULL DEFAULT 9;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'admin_telegram_bindings' AND column_name = 'weekly_summary_hour') THEN
        ALTER TABLE admin_telegram_bindings ADD COLUMN weekly_summary_hour INT NOT NULL DEFAULT 9;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'admin_telegram_bindings' AND column_name = 'weekly_summary_weekday') THEN
        ALTER TABLE admin_telegram_bindings ADD COLUMN weekly_summary_weekday INT NOT NULL DEFAULT 1;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'admin_telegram_bindings' AND column_name = 'offline_check_interval_minutes') THEN
        ALTER TABLE admin_telegram_bindings ADD COLUMN offline_check_interval_minutes INT NOT NULL DEFAULT 5;
    END IF;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CALL add_schedule_config_fields();
-- +goose StatementEnd

-- +goose StatementBegin
DROP PROCEDURE IF EXISTS add_schedule_config_fields;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE admin_telegram_bindings
    DROP COLUMN IF EXISTS daily_summary_hour,
    DROP COLUMN IF EXISTS weekly_summary_hour,
    DROP COLUMN IF EXISTS weekly_summary_weekday,
    DROP COLUMN IF EXISTS offline_check_interval_minutes;
-- +goose StatementEnd
