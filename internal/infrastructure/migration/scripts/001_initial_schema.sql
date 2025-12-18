-- +goose Up
-- Migration: Consolidated initial database schema
-- Created: 2025-12-18
-- Description: Complete database schema with all tables and final structures

-- ============================================================================
-- Section 1: User Authentication & Authorization Tables
-- ============================================================================

CREATE TABLE users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    avatar_url VARCHAR(500),
    email_verified BOOLEAN DEFAULT FALSE,
    locale VARCHAR(10) DEFAULT 'en',
    password_hash VARCHAR(255),
    email_verification_token VARCHAR(255),
    email_verification_expires_at TIMESTAMP NULL,
    password_reset_token VARCHAR(255),
    password_reset_expires_at TIMESTAMP NULL,
    last_password_change_at TIMESTAMP NULL,
    failed_login_attempts INT UNSIGNED DEFAULT 0,
    locked_until TIMESTAMP NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_users_email (email),
    INDEX idx_users_role (role),
    INDEX idx_users_status (status),
    INDEX idx_users_deleted_at (deleted_at),
    INDEX idx_email_verified (email_verified),
    INDEX idx_password_reset_token (password_reset_token),
    INDEX idx_email_verification_token (email_verification_token)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE oauth_accounts (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    provider_email VARCHAR(255),
    provider_username VARCHAR(255),
    provider_avatar_url VARCHAR(500),
    raw_user_info JSON,
    last_login_at TIMESTAMP NULL,
    login_count INT UNSIGNED DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_provider_account (provider, provider_user_id),
    INDEX idx_user_id (user_id),
    INDEX idx_provider (provider),
    INDEX idx_provider_email (provider, provider_email),
    INDEX idx_last_login (last_login_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE sessions (
    id VARCHAR(64) PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    device_name VARCHAR(255),
    device_type VARCHAR(50),
    ip_address VARCHAR(45),
    user_agent TEXT,
    token_hash VARCHAR(64) NOT NULL,
    refresh_token_hash VARCHAR(64),
    expires_at TIMESTAMP NOT NULL,
    last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_sessions (user_id, expires_at),
    INDEX idx_token_hash (token_hash),
    INDEX idx_expires_at (expires_at),
    INDEX idx_last_activity (last_activity_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE oauth_providers (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    auth_url VARCHAR(500) NOT NULL,
    token_url VARCHAR(500) NOT NULL,
    user_info_url VARCHAR(500) NOT NULL,
    default_scopes VARCHAR(500) NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    allow_signup BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO oauth_providers (name, display_name, auth_url, token_url, user_info_url, default_scopes) VALUES
('google', 'Google', 'https://accounts.google.com/o/oauth2/v2/auth', 'https://oauth2.googleapis.com/token', 'https://www.googleapis.com/oauth2/v2/userinfo', 'openid email profile'),
('github', 'GitHub', 'https://github.com/login/oauth/authorize', 'https://github.com/login/oauth/access_token', 'https://api.github.com/user', 'read:user user:email');

-- ============================================================================
-- Section 2: Subscription Management Tables
-- ============================================================================

CREATE TABLE plans (
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
    version INT NOT NULL DEFAULT 1,
    deleted_at TIMESTAMP NULL,
    INDEX idx_slug (slug),
    INDEX idx_status (status),
    INDEX idx_is_public (is_public),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE plan_pricings (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    plan_id BIGINT UNSIGNED NOT NULL,
    billing_cycle VARCHAR(20) NOT NULL,
    price BIGINT UNSIGNED NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'CNY',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,
    UNIQUE KEY uk_plan_billing_cycle (plan_id, billing_cycle),
    INDEX idx_plan_id (plan_id),
    INDEX idx_billing_cycle (billing_cycle),
    INDEX idx_is_active (is_active),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE entitlements (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    plan_id BIGINT UNSIGNED NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_plan_resource (plan_id, resource_type, resource_id),
    INDEX idx_plan_id (plan_id),
    INDEX idx_resource (resource_type, resource_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE subscriptions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL,
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
    UNIQUE INDEX idx_uuid (uuid),
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
    resource_type VARCHAR(50) NOT NULL COMMENT 'Resource type: node / forward_rule',
    resource_id BIGINT UNSIGNED NOT NULL COMMENT 'Resource ID',
    upload BIGINT UNSIGNED NOT NULL DEFAULT 0,
    download BIGINT UNSIGNED NOT NULL DEFAULT 0,
    total BIGINT UNSIGNED NOT NULL DEFAULT 0,
    period TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_subscription_period (subscription_id, period),
    INDEX idx_resource (resource_type, resource_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ============================================================================
-- Section 3: Notification Tables
-- ============================================================================

CREATE TABLE notifications (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content LONGTEXT NOT NULL,
    related_id BIGINT UNSIGNED NULL,
    read_status VARCHAR(20) NOT NULL DEFAULT 'unread',
    archived_at TIMESTAMP NULL,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_user_read (user_id, read_status),
    INDEX idx_created_at (created_at),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE announcements (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content LONGTEXT NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'system',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    creator_id BIGINT UNSIGNED NOT NULL,
    priority INT DEFAULT 3,
    scheduled_at TIMESTAMP NULL,
    expires_at TIMESTAMP NULL,
    view_count INT DEFAULT 0,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_status (status),
    INDEX idx_creator_id (creator_id),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE notification_templates (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    template_type VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content LONGTEXT NOT NULL,
    variables JSON,
    enabled BOOLEAN DEFAULT TRUE,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_template_type (template_type),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ============================================================================
-- Section 4: Node Management Tables
-- ============================================================================

CREATE TABLE nodes (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    short_id VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL UNIQUE,
    server_address VARCHAR(255) NOT NULL,
    agent_port SMALLINT UNSIGNED NOT NULL,
    subscription_port SMALLINT UNSIGNED DEFAULT NULL,
    protocol VARCHAR(20) NOT NULL DEFAULT 'shadowsocks',
    status VARCHAR(20) NOT NULL DEFAULT 'inactive',
    region VARCHAR(100),
    tags JSON,
    plan_ids JSON,
    sort_order INT NOT NULL DEFAULT 0,
    maintenance_reason VARCHAR(500),
    token_hash VARCHAR(255) NOT NULL,
    api_token VARCHAR(255) DEFAULT NULL,
    last_seen_at TIMESTAMP NULL DEFAULT NULL,
    public_ipv4 VARCHAR(15) NULL DEFAULT NULL,
    public_ipv6 VARCHAR(45) NULL DEFAULT NULL,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_status (status),
    INDEX idx_protocol (protocol),
    INDEX idx_agent_address (server_address, agent_port),
    UNIQUE INDEX idx_token_hash (token_hash),
    INDEX idx_nodes_last_seen_at (last_seen_at),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE shadowsocks_configs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    node_id BIGINT UNSIGNED NOT NULL,
    encryption_method VARCHAR(50) NOT NULL,
    plugin VARCHAR(100),
    plugin_opts JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE INDEX idx_shadowsocks_configs_node_id (node_id),
    INDEX idx_shadowsocks_configs_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE trojan_configs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    node_id BIGINT UNSIGNED NOT NULL,
    transport_protocol VARCHAR(10) NOT NULL DEFAULT 'tcp',
    host VARCHAR(255),
    path VARCHAR(255),
    sni VARCHAR(255),
    allow_insecure TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE INDEX idx_trojan_configs_node_id (node_id),
    INDEX idx_trojan_configs_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ============================================================================
-- Section 5: Forward Port Management Tables
-- ============================================================================

CREATE TABLE forward_agents (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    short_id VARCHAR(16) NOT NULL,
    name VARCHAR(100) NOT NULL,
    token_hash VARCHAR(64) NOT NULL,
    api_token VARCHAR(255) DEFAULT NULL,
    public_address VARCHAR(255) NULL DEFAULT NULL,
    tunnel_address VARCHAR(255) DEFAULT NULL,
    plan_ids JSON,
    status VARCHAR(20) NOT NULL DEFAULT 'enabled',
    remark VARCHAR(500) DEFAULT '',
    last_seen_at DATETIME NULL DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,
    UNIQUE INDEX idx_forward_agent_short_id (short_id),
    INDEX idx_forward_agent_name (name),
    INDEX idx_forward_agent_token_hash (token_hash),
    INDEX idx_forward_agent_status (status),
    INDEX idx_forward_agent_last_seen_at (last_seen_at),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE forward_rules (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    short_id VARCHAR(16) NOT NULL,
    agent_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NULL,
    name VARCHAR(100) NOT NULL,
    listen_port SMALLINT UNSIGNED NOT NULL,
    target_address VARCHAR(255) DEFAULT '',
    target_port SMALLINT UNSIGNED DEFAULT 0,
    target_node_id BIGINT UNSIGNED NULL DEFAULT NULL,
    bind_ip VARCHAR(45) DEFAULT '',
    protocol VARCHAR(10) NOT NULL DEFAULT 'tcp',
    rule_type VARCHAR(20) NOT NULL DEFAULT 'direct',
    exit_agent_id BIGINT UNSIGNED NULL DEFAULT NULL,
    chain_agent_ids JSON DEFAULT NULL,
    chain_port_config JSON DEFAULT NULL,
    ip_version VARCHAR(10) NOT NULL DEFAULT 'auto',
    plan_ids JSON,
    traffic_multiplier DECIMAL(10,4) NULL DEFAULT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'disabled',
    remark VARCHAR(500) DEFAULT '',
    upload_bytes BIGINT NOT NULL DEFAULT 0,
    download_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,
    UNIQUE INDEX idx_forward_rule_short_id (short_id),
    UNIQUE INDEX idx_listen_port_agent (listen_port, agent_id),
    INDEX idx_forward_agent_id (agent_id),
    INDEX idx_forward_exit_agent_id (exit_agent_id),
    INDEX idx_forward_target_node_id (target_node_id),
    INDEX idx_forward_name (name),
    INDEX idx_forward_protocol (protocol),
    INDEX idx_forward_status (status),
    INDEX idx_forward_rules_traffic_multiplier (traffic_multiplier),
    INDEX idx_forward_rules_user_id (user_id),
    INDEX idx_forward_rules_user_status (user_id, status),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- +goose Down
-- Rollback Migration: Drop all tables in reverse order
-- Description: Remove all application tables

DROP TABLE IF EXISTS forward_rules;
DROP TABLE IF EXISTS forward_agents;
DROP TABLE IF EXISTS trojan_configs;
DROP TABLE IF EXISTS shadowsocks_configs;
DROP TABLE IF EXISTS nodes;
DROP TABLE IF EXISTS notification_templates;
DROP TABLE IF EXISTS announcements;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS subscription_usages;
DROP TABLE IF EXISTS subscription_histories;
DROP TABLE IF EXISTS subscription_tokens;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS entitlements;
DROP TABLE IF EXISTS plan_pricings;
DROP TABLE IF EXISTS plans;
DROP TABLE IF EXISTS oauth_providers;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS oauth_accounts;
DROP TABLE IF EXISTS users;
