-- +goose Up
CREATE TABLE subscription_usage_stats (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    sid VARCHAR(50) NOT NULL,
    subscription_id INT UNSIGNED,
    resource_type VARCHAR(50) NOT NULL DEFAULT 'node',
    resource_id INT UNSIGNED NOT NULL DEFAULT 0,
    upload BIGINT UNSIGNED NOT NULL DEFAULT 0,
    download BIGINT UNSIGNED NOT NULL DEFAULT 0,
    total BIGINT UNSIGNED NOT NULL DEFAULT 0,
    granularity VARCHAR(10) NOT NULL COMMENT 'daily or monthly',
    period DATE NOT NULL COMMENT 'date for daily, first day of month for monthly',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_sid (sid),
    INDEX idx_subscription_period (subscription_id, granularity, period),
    INDEX idx_resource_period (resource_type, resource_id, granularity, period),
    UNIQUE INDEX idx_unique_stat (subscription_id, resource_type, resource_id, granularity, period)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS subscription_usage_stats;
