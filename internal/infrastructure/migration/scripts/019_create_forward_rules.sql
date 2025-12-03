-- +goose Up

-- Create forward_rules table for TCP/UDP port forwarding configuration
CREATE TABLE forward_rules (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    agent_id BIGINT UNSIGNED NOT NULL COMMENT 'Forward agent ID that executes this rule',
    next_agent_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Next agent ID for chain forward (0=direct forward to target)',
    name VARCHAR(100) NOT NULL COMMENT 'Rule name',
    listen_port SMALLINT UNSIGNED NOT NULL COMMENT 'Port to listen on',
    target_address VARCHAR(255) DEFAULT '' COMMENT 'Target address (required when next_agent_id=0)',
    target_port SMALLINT UNSIGNED DEFAULT 0 COMMENT 'Target port (required when next_agent_id=0)',
    protocol VARCHAR(10) NOT NULL DEFAULT 'tcp' COMMENT 'Protocol: tcp, udp, both',
    status VARCHAR(20) NOT NULL DEFAULT 'disabled' COMMENT 'Status: enabled, disabled',
    remark VARCHAR(500) DEFAULT '' COMMENT 'Optional remark',
    upload_bytes BIGINT NOT NULL DEFAULT 0 COMMENT 'Total upload bytes',
    download_bytes BIGINT NOT NULL DEFAULT 0 COMMENT 'Total download bytes',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,

    UNIQUE INDEX idx_listen_port_agent (listen_port, agent_id),
    INDEX idx_forward_agent_id (agent_id),
    INDEX idx_forward_next_agent_id (next_agent_id),
    INDEX idx_forward_name (name),
    INDEX idx_forward_protocol (protocol),
    INDEX idx_forward_status (status),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- +goose Down

DROP TABLE IF EXISTS forward_rules;
