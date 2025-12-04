-- +goose Up
-- Create shadowsocks_configs table for Shadowsocks protocol-specific configuration
-- This separates protocol-specific fields from the nodes table for better maintainability

CREATE TABLE IF NOT EXISTS shadowsocks_configs (
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
);

-- Migrate existing Shadowsocks data from nodes table
-- Include nodes with empty protocol (legacy data defaults to shadowsocks)
INSERT INTO shadowsocks_configs (node_id, encryption_method, plugin, plugin_opts, created_at, updated_at)
SELECT id, encryption_method, plugin, plugin_opts, created_at, updated_at
FROM nodes
WHERE (protocol = 'shadowsocks' OR protocol = '' OR protocol IS NULL) AND deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS shadowsocks_configs;
