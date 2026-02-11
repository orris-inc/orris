-- +goose Up
CREATE TABLE node_anytls_configs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    node_id BIGINT UNSIGNED NOT NULL,
    sni VARCHAR(255),
    allow_insecure TINYINT(1) NOT NULL DEFAULT 1,
    fingerprint VARCHAR(100),
    idle_session_check_interval VARCHAR(20),
    idle_session_timeout VARCHAR(20),
    min_idle_session INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE INDEX idx_node_anytls_configs_node_id (node_id),
    INDEX idx_node_anytls_configs_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- +goose Down
DROP TABLE IF EXISTS node_anytls_configs;
