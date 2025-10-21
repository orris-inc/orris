-- +goose Up
-- Migration: Subscription management tables
-- Created: 2025-10-21
-- Description: Create tables for subscription, subscription plans, tokens, history, and usage tracking

CREATE TABLE subscription_plans (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL UNIQUE,
    description VARCHAR(500),
    price BIGINT UNSIGNED NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'CNY',
    billing_cycle VARCHAR(20) NOT NULL,
    trial_days INT DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    features JSON,
    limits JSON,
    custom_endpoint VARCHAR(200),
    api_rate_limit INT UNSIGNED DEFAULT 60,
    max_users INT UNSIGNED DEFAULT 1,
    max_projects INT UNSIGNED DEFAULT 1,
    storage_limit BIGINT UNSIGNED DEFAULT 1073741824,
    is_public BOOLEAN DEFAULT TRUE,
    sort_order INT DEFAULT 0,
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_slug (slug),
    INDEX idx_status (status),
    INDEX idx_is_public (is_public),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE subscriptions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    plan_id BIGINT UNSIGNED NOT NULL,
    status VARCHAR(20) NOT NULL,
    start_date DATETIME NOT NULL,
    end_date DATETIME NOT NULL,
    auto_renew BOOLEAN DEFAULT FALSE,
    current_period_start DATETIME NOT NULL,
    current_period_end DATETIME NOT NULL,
    cancelled_at DATETIME NULL,
    cancel_reason VARCHAR(500),
    metadata JSON,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_user_subscription (user_id),
    INDEX idx_plan_subscription (plan_id),
    INDEX idx_status (status),
    INDEX idx_end_date (end_date),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE subscription_tokens (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    subscription_id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(100) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    prefix VARCHAR(20) NOT NULL,
    scope VARCHAR(20) NOT NULL,
    expires_at DATETIME NULL,
    last_used_at DATETIME NULL,
    last_used_ip VARCHAR(45),
    usage_count BIGINT UNSIGNED DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    revoked_at DATETIME NULL,
    INDEX idx_subscription_token (subscription_id),
    INDEX idx_expires_at (expires_at),
    INDEX idx_active (is_active),
    INDEX idx_token_hash (token_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE subscription_histories (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    subscription_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    plan_id BIGINT UNSIGNED NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_status VARCHAR(20),
    new_status VARCHAR(20) NOT NULL,
    old_plan_id BIGINT UNSIGNED,
    new_plan_id BIGINT UNSIGNED,
    amount BIGINT UNSIGNED,
    currency VARCHAR(3),
    reason VARCHAR(500),
    performed_by BIGINT UNSIGNED,
    ip_address VARCHAR(45),
    user_agent VARCHAR(255),
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_subscription_history (subscription_id),
    INDEX idx_user_history (user_id),
    INDEX idx_action (action),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE subscription_usages (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    subscription_id BIGINT UNSIGNED NOT NULL,
    period_start DATETIME NOT NULL,
    period_end DATETIME NOT NULL,
    api_requests BIGINT UNSIGNED NOT NULL DEFAULT 0,
    api_data_out BIGINT UNSIGNED NOT NULL DEFAULT 0,
    api_data_in BIGINT UNSIGNED NOT NULL DEFAULT 0,
    storage_used BIGINT UNSIGNED NOT NULL DEFAULT 0,
    users_count INT UNSIGNED NOT NULL DEFAULT 0,
    projects_count INT UNSIGNED NOT NULL DEFAULT 0,
    webhook_calls BIGINT UNSIGNED NOT NULL DEFAULT 0,
    emails_sent BIGINT UNSIGNED NOT NULL DEFAULT 0,
    reports_generated INT UNSIGNED NOT NULL DEFAULT 0,
    last_reset_at DATETIME NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE KEY idx_subscription_period (subscription_id, period_start),
    INDEX idx_period_end (period_end),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- +goose Down
-- Rollback Migration: Drop subscription tables
-- Description: Remove all subscription-related tables

DROP TABLE IF EXISTS subscription_usages;
DROP TABLE IF EXISTS subscription_histories;
DROP TABLE IF EXISTS subscription_tokens;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS subscription_plans;
