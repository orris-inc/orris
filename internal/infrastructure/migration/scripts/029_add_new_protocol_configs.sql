-- +goose Up
-- +goose StatementBegin

-- VLESS Configuration Table
CREATE TABLE node_vless_configs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    node_id BIGINT UNSIGNED NOT NULL,
    -- Transport layer
    transport_type VARCHAR(10) NOT NULL DEFAULT 'tcp',  -- tcp, ws, grpc, h2
    host VARCHAR(255),                                   -- WS/H2 Host header
    path VARCHAR(255),                                   -- WS/H2 path
    service_name VARCHAR(255),                           -- gRPC service name
    -- Flow control
    flow VARCHAR(30),                                    -- xtls-rprx-vision or empty
    -- Security
    security VARCHAR(20) NOT NULL DEFAULT 'tls',         -- none, tls, reality
    sni VARCHAR(255),                                    -- TLS SNI
    fingerprint VARCHAR(50),                             -- TLS fingerprint
    allow_insecure TINYINT(1) NOT NULL DEFAULT 0,
    -- Reality specific
    public_key VARCHAR(100),                             -- Reality public key
    short_id VARCHAR(20),                                -- Reality short ID
    spider_x VARCHAR(500),                               -- Reality spider X
    -- Timestamps
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
    -- Core
    alter_id INT UNSIGNED NOT NULL DEFAULT 0,
    security VARCHAR(30) NOT NULL DEFAULT 'auto',        -- auto, aes-128-gcm, chacha20-poly1305, none, zero
    -- Transport
    transport_type VARCHAR(10) NOT NULL DEFAULT 'tcp',   -- tcp, ws, grpc, http, quic
    host VARCHAR(255),
    path VARCHAR(255),
    service_name VARCHAR(255),
    -- TLS
    tls TINYINT(1) NOT NULL DEFAULT 0,
    sni VARCHAR(255),
    allow_insecure TINYINT(1) NOT NULL DEFAULT 0,
    -- Timestamps
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
    -- Core
    congestion_control VARCHAR(20) NOT NULL DEFAULT 'bbr',  -- cubic, bbr, new_reno
    -- Obfuscation
    obfs VARCHAR(20),                                        -- salamander or empty
    obfs_password VARCHAR(255),
    -- Bandwidth limits (optional)
    up_mbps INT UNSIGNED,
    down_mbps INT UNSIGNED,
    -- TLS
    sni VARCHAR(255),
    allow_insecure TINYINT(1) NOT NULL DEFAULT 0,
    fingerprint VARCHAR(100),
    -- Timestamps
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
    -- Core
    congestion_control VARCHAR(20) NOT NULL DEFAULT 'bbr',  -- cubic, bbr, new_reno
    udp_relay_mode VARCHAR(10) NOT NULL DEFAULT 'native',   -- native, quic
    -- ALPN
    alpn VARCHAR(50),
    -- TLS
    sni VARCHAR(255),
    allow_insecure TINYINT(1) NOT NULL DEFAULT 0,
    disable_sni TINYINT(1) NOT NULL DEFAULT 0,
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    UNIQUE INDEX idx_node_tuic_configs_node_id (node_id),
    INDEX idx_node_tuic_configs_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS node_tuic_configs;
DROP TABLE IF EXISTS node_hysteria2_configs;
DROP TABLE IF EXISTS node_vmess_configs;
DROP TABLE IF EXISTS node_vless_configs;
-- +goose StatementEnd
