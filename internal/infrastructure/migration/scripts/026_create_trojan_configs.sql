-- +goose Up
-- Create trojan_configs table for Trojan protocol-specific configuration
-- This separates protocol-specific fields from the nodes table for better maintainability

CREATE TABLE IF NOT EXISTS trojan_configs (
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
);

-- +goose Down
DROP TABLE IF EXISTS trojan_configs;
