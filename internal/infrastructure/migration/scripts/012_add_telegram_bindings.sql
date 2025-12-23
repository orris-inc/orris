-- +goose Up
-- Migration: Add telegram bindings table for Telegram notification feature

CREATE TABLE telegram_bindings (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    sid VARCHAR(50) NOT NULL COMMENT 'Stripe-style ID: tg_bind_xxxxxxxx',
    user_id BIGINT UNSIGNED NOT NULL COMMENT 'Reference to users.id',
    telegram_user_id BIGINT NOT NULL COMMENT 'Telegram user ID',
    telegram_username VARCHAR(100) DEFAULT '' COMMENT 'Telegram @username',

    -- Notification preferences
    notify_expiring BOOLEAN NOT NULL DEFAULT TRUE COMMENT 'Enable subscription expiring reminder',
    notify_traffic BOOLEAN NOT NULL DEFAULT TRUE COMMENT 'Enable traffic usage reminder',
    expiring_days INT NOT NULL DEFAULT 3 COMMENT 'Days before expiry to send reminder (1-30)',
    traffic_threshold INT NOT NULL DEFAULT 80 COMMENT 'Traffic usage percentage threshold (50-99)',

    -- Time window deduplication (24 hours)
    last_expiring_notify_at TIMESTAMP NULL DEFAULT NULL COMMENT 'Last expiring notification sent time',
    last_traffic_notify_at TIMESTAMP NULL DEFAULT NULL COMMENT 'Last traffic notification sent time',

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_telegram_bindings_sid (sid),
    UNIQUE INDEX idx_telegram_bindings_user_id (user_id),
    UNIQUE INDEX idx_telegram_bindings_telegram_user_id (telegram_user_id),
    INDEX idx_telegram_bindings_notify_expiring (notify_expiring, last_expiring_notify_at),
    INDEX idx_telegram_bindings_notify_traffic (notify_traffic, last_traffic_notify_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- +goose Down
DROP TABLE IF EXISTS telegram_bindings;
