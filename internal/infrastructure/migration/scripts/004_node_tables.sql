-- +goose Up
-- Migration: Node management tables
-- Created: 2025-11-05
-- Description: Create tables for node management, node groups, and traffic statistics

-- Create nodes table for proxy server configuration
CREATE TABLE nodes (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    server_address VARCHAR(255) NOT NULL,
    server_port SMALLINT UNSIGNED NOT NULL,
    encryption_method VARCHAR(50) NOT NULL,
    encryption_password VARCHAR(255) NOT NULL,
    plugin VARCHAR(255),
    plugin_opts JSON,
    protocol VARCHAR(20) NOT NULL DEFAULT 'shadowsocks',
    status VARCHAR(20) NOT NULL DEFAULT 'inactive',
    country VARCHAR(50),
    region VARCHAR(100),
    tags JSON,
    custom_fields JSON,
    max_users INT UNSIGNED NOT NULL DEFAULT 0,
    traffic_limit BIGINT UNSIGNED NOT NULL DEFAULT 0,
    traffic_used BIGINT UNSIGNED NOT NULL DEFAULT 0,
    traffic_reset_at TIMESTAMP NULL,
    sort_order INT NOT NULL DEFAULT 0,
    maintenance_reason VARCHAR(500),
    token_hash VARCHAR(255) NOT NULL,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_status (status),
    INDEX idx_protocol (protocol),
    INDEX idx_server (server_address, server_port),
    UNIQUE INDEX idx_token_hash (token_hash),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Create node_groups table for organizing nodes
CREATE TABLE node_groups (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description VARCHAR(500),
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    metadata JSON,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_is_public (is_public),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Create node_group_nodes table for many-to-many relationship between node groups and nodes
CREATE TABLE node_group_nodes (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    node_group_id BIGINT UNSIGNED NOT NULL,
    node_id BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_node_group_node (node_group_id, node_id),
    UNIQUE INDEX idx_node_group_node_unique (node_group_id, node_id),
    FOREIGN KEY (node_group_id) REFERENCES node_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Create node_group_plans table for many-to-many relationship between node groups and subscription plans
CREATE TABLE node_group_plans (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    node_group_id BIGINT UNSIGNED NOT NULL,
    subscription_plan_id BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_node_group_plan (node_group_id, subscription_plan_id),
    UNIQUE INDEX idx_node_group_plan_unique (node_group_id, subscription_plan_id),
    FOREIGN KEY (node_group_id) REFERENCES node_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (subscription_plan_id) REFERENCES subscription_plans(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Create node_traffic table for node-level traffic statistics
CREATE TABLE node_traffic (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    node_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED,
    subscription_id BIGINT UNSIGNED,
    upload BIGINT UNSIGNED NOT NULL DEFAULT 0,
    download BIGINT UNSIGNED NOT NULL DEFAULT 0,
    total BIGINT UNSIGNED NOT NULL DEFAULT 0,
    period TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_node_period (node_id, period),
    INDEX idx_user_period (user_id, period),
    INDEX idx_subscription (subscription_id),
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Create user_traffic table for user-level traffic statistics per node
CREATE TABLE user_traffic (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    node_id BIGINT UNSIGNED NOT NULL,
    subscription_id BIGINT UNSIGNED,
    upload BIGINT UNSIGNED NOT NULL DEFAULT 0,
    download BIGINT UNSIGNED NOT NULL DEFAULT 0,
    total BIGINT UNSIGNED NOT NULL DEFAULT 0,
    period TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_node_period (user_id, node_id, period),
    INDEX idx_user_period (user_id, period),
    INDEX idx_node_period (node_id, period),
    INDEX idx_subscription (subscription_id),
    UNIQUE INDEX idx_user_traffic_unique (user_id, node_id, period),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE,
    FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- +goose Down
-- Rollback Migration: Drop node-related tables
-- Description: Remove all node management and traffic tracking tables

DROP TABLE IF EXISTS user_traffic;
DROP TABLE IF EXISTS node_traffic;
DROP TABLE IF EXISTS node_group_plans;
DROP TABLE IF EXISTS node_group_nodes;
DROP TABLE IF EXISTS node_groups;
DROP TABLE IF EXISTS nodes;
