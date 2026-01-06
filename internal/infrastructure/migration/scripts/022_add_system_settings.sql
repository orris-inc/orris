-- +goose Up
-- Migration: Add system_settings table for centralized configuration management

CREATE TABLE system_settings (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    sid VARCHAR(50) NOT NULL COMMENT 'Stripe-style ID: setting_xxxxxxxx',
    category VARCHAR(50) NOT NULL COMMENT 'Setting category: telegram, email, etc.',
    setting_key VARCHAR(100) NOT NULL COMMENT 'Setting key within category',
    value TEXT DEFAULT NULL COMMENT 'Setting value (stored as string, parsed based on value_type)',
    value_type VARCHAR(20) NOT NULL DEFAULT 'string' COMMENT 'Value type: string, int, bool, json',
    description VARCHAR(500) DEFAULT '' COMMENT 'Human-readable description',
    updated_by BIGINT UNSIGNED DEFAULT NULL COMMENT 'Reference to users.id who last updated',
    version INT NOT NULL DEFAULT 1 COMMENT 'Optimistic locking version',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_system_settings_sid (sid),
    UNIQUE INDEX idx_system_settings_category_key (category, setting_key),
    INDEX idx_system_settings_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Insert Telegram initial settings
INSERT INTO system_settings (sid, category, setting_key, value_type, description) VALUES
    ('setting_tg_bot_token', 'telegram', 'bot_token', 'string', 'Telegram Bot API token'),
    ('setting_tg_webhook_url', 'telegram', 'webhook_url', 'string', 'Telegram webhook callback URL'),
    ('setting_tg_webhook_secret', 'telegram', 'webhook_secret', 'string', 'Telegram webhook secret for verification'),
    ('setting_tg_enabled', 'telegram', 'enabled', 'bool', 'Whether Telegram integration is enabled');

-- +goose Down
DROP TABLE IF EXISTS system_settings;
