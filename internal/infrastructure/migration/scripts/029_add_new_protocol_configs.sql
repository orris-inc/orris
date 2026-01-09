-- +goose Up

-- VLESS Configuration Table
CREATE TABLE node_vless_configs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    node_id BIGINT UNSIGNED NOT NULL,
    transport_type VARCHAR(10) NOT NULL DEFAULT 'tcp',
    host VARCHAR(255),
    path VARCHAR(255),
    service_name VARCHAR(255),
    flow VARCHAR(30),
    security VARCHAR(20) NOT NULL DEFAULT 'tls',
    sni VARCHAR(255),
    fingerprint VARCHAR(50),
    allow_insecure TINYINT(1) NOT NULL DEFAULT 0,
    public_key VARCHAR(100),
    short_id VARCHAR(20),
    spider_x VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE INDEX idx_node_vless_configs_node_id (node_id),
    INDEX idx_node_vless_configs_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- VMess Configuration Table
CREATE TABLE node_vmess_configs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    node_id BIGINT UNSIGNED NOT NULL,
    alter_id INT UNSIGNED NOT NULL DEFAULT 0,
    security VARCHAR(30) NOT NULL DEFAULT 'auto',
    transport_type VARCHAR(10) NOT NULL DEFAULT 'tcp',
    host VARCHAR(255),
    path VARCHAR(255),
    service_name VARCHAR(255),
    tls TINYINT(1) NOT NULL DEFAULT 0,
    sni VARCHAR(255),
    allow_insecure TINYINT(1) NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE INDEX idx_node_vmess_configs_node_id (node_id),
    INDEX idx_node_vmess_configs_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Hysteria2 Configuration Table
CREATE TABLE node_hysteria2_configs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    node_id BIGINT UNSIGNED NOT NULL,
    congestion_control VARCHAR(20) NOT NULL DEFAULT 'bbr',
    obfs VARCHAR(20),
    obfs_password VARCHAR(255),
    up_mbps INT UNSIGNED,
    down_mbps INT UNSIGNED,
    sni VARCHAR(255),
    allow_insecure TINYINT(1) NOT NULL DEFAULT 0,
    fingerprint VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE INDEX idx_node_hysteria2_configs_node_id (node_id),
    INDEX idx_node_hysteria2_configs_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- TUIC Configuration Table
CREATE TABLE node_tuic_configs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    node_id BIGINT UNSIGNED NOT NULL,
    congestion_control VARCHAR(20) NOT NULL DEFAULT 'bbr',
    udp_relay_mode VARCHAR(10) NOT NULL DEFAULT 'native',
    alpn VARCHAR(50),
    sni VARCHAR(255),
    allow_insecure TINYINT(1) NOT NULL DEFAULT 0,
    disable_sni TINYINT(1) NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE INDEX idx_node_tuic_configs_node_id (node_id),
    INDEX idx_node_tuic_configs_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- +goose Down
DROP TABLE IF EXISTS node_tuic_configs;
DROP TABLE IF EXISTS node_hysteria2_configs;
DROP TABLE IF EXISTS node_vmess_configs;
DROP TABLE IF EXISTS node_vless_configs;
