-- +goose Up

-- Create forward_agents table for managing forwarding agents
CREATE TABLE forward_agents (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT 'Agent node name',
    token_hash VARCHAR(64) NOT NULL COMMENT 'SHA256 hash of the agent token',
    status VARCHAR(20) NOT NULL DEFAULT 'enabled' COMMENT 'Status: enabled, disabled',
    remark VARCHAR(500) DEFAULT '' COMMENT 'Optional remark',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,

    INDEX idx_forward_agent_name (name),
    INDEX idx_forward_agent_token_hash (token_hash),
    INDEX idx_forward_agent_status (status),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- +goose Down

DROP TABLE IF EXISTS forward_agents;
