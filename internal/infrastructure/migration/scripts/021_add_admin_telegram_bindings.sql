-- +goose Up
-- +goose StatementBegin
CREATE TABLE admin_telegram_bindings (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    sid VARCHAR(50) NOT NULL COMMENT 'Stripe-style ID: atg_bind_xxxxxxxx',
    user_id BIGINT UNSIGNED NOT NULL COMMENT 'Reference to users.id (must be admin role)',
    telegram_user_id BIGINT NOT NULL COMMENT 'Telegram user ID',
    telegram_username VARCHAR(100) DEFAULT '' COMMENT 'Telegram @username',

    -- Notification preferences
    notify_node_offline BOOLEAN NOT NULL DEFAULT TRUE COMMENT 'Notify when node goes offline',
    notify_agent_offline BOOLEAN NOT NULL DEFAULT TRUE COMMENT 'Notify when forward agent goes offline',
    notify_new_user BOOLEAN NOT NULL DEFAULT TRUE COMMENT 'Notify on new user registration',
    notify_payment_success BOOLEAN NOT NULL DEFAULT TRUE COMMENT 'Notify on successful payments',
    notify_daily_summary BOOLEAN NOT NULL DEFAULT TRUE COMMENT 'Receive daily business summary',
    notify_weekly_summary BOOLEAN NOT NULL DEFAULT TRUE COMMENT 'Receive weekly business summary',

    -- Thresholds
    offline_threshold_minutes INT NOT NULL DEFAULT 5 COMMENT 'Minutes before considering offline (3-30)',

    -- Deduplication timestamps
    last_node_offline_notify_at TIMESTAMP NULL DEFAULT NULL,
    last_agent_offline_notify_at TIMESTAMP NULL DEFAULT NULL,
    last_daily_summary_at TIMESTAMP NULL DEFAULT NULL,
    last_weekly_summary_at TIMESTAMP NULL DEFAULT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_admin_telegram_bindings_sid (sid),
    UNIQUE INDEX idx_admin_telegram_bindings_user_id (user_id),
    UNIQUE INDEX idx_admin_telegram_bindings_telegram_user_id (telegram_user_id),
    INDEX idx_admin_telegram_bindings_notify_offline (notify_node_offline, notify_agent_offline)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS admin_telegram_bindings;
-- +goose StatementEnd
