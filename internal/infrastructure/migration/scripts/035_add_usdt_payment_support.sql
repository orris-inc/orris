-- +goose Up
-- +goose StatementBegin
-- Create payments table with USDT support
CREATE TABLE payments (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    order_no VARCHAR(64) NOT NULL,
    subscription_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL,
    payment_method VARCHAR(20) NOT NULL,
    payment_status VARCHAR(20) NOT NULL,
    gateway_order_no VARCHAR(128) DEFAULT NULL,
    transaction_id VARCHAR(128) DEFAULT NULL,
    payment_url TEXT DEFAULT NULL,
    qr_code TEXT DEFAULT NULL,
    paid_at TIMESTAMP NULL DEFAULT NULL,
    expired_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- USDT-specific fields
    chain_type VARCHAR(10) DEFAULT NULL,
    usdt_amount_raw BIGINT UNSIGNED DEFAULT NULL COMMENT 'USDT amount in smallest unit (1 USDT = 1000000)',
    receiving_address VARCHAR(64) DEFAULT NULL,
    exchange_rate DECIMAL(20, 8) DEFAULT NULL COMMENT 'Exchange rate at time of payment (for display only)',
    tx_hash VARCHAR(128) DEFAULT NULL,
    block_number BIGINT UNSIGNED DEFAULT NULL,
    confirmed_at TIMESTAMP NULL DEFAULT NULL,
    metadata JSON DEFAULT NULL,
    version INT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_order_no (order_no),
    INDEX idx_subscription_id (subscription_id),
    INDEX idx_user_id (user_id),
    INDEX idx_payment_status (payment_status),
    INDEX idx_gateway_order_no (gateway_order_no),
    UNIQUE KEY uk_payments_chain_tx_hash (chain_type, tx_hash)
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Create suffix allocation table for unique USDT amounts (multi-wallet support)
CREATE TABLE usdt_amount_suffixes (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    chain_type VARCHAR(10) NOT NULL,
    receiving_address VARCHAR(64) NOT NULL,
    base_amount_raw BIGINT UNSIGNED NOT NULL COMMENT 'Base amount in smallest unit (1 USDT = 1000000)',
    suffix INT UNSIGNED NOT NULL,
    payment_id BIGINT UNSIGNED DEFAULT NULL,
    allocated_at TIMESTAMP NULL DEFAULT NULL,
    expires_at TIMESTAMP NULL DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_chain_address_base_suffix (chain_type, receiving_address, base_amount_raw, suffix)
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Insert USDT settings (multi-wallet address arrays)
INSERT INTO system_settings (sid, category, setting_key, value, value_type, description) VALUES
    ('setting_usdt_enabled', 'usdt', 'enabled', 'false', 'bool', 'Enable USDT payment'),
    ('setting_usdt_pol_addresses', 'usdt', 'pol_receiving_addresses', '[]', 'json', 'Polygon USDT receiving addresses'),
    ('setting_usdt_trc_addresses', 'usdt', 'trc_receiving_addresses', '[]', 'json', 'Tron TRC-20 USDT receiving addresses'),
    ('setting_usdt_polygonscan_key', 'usdt', 'polygonscan_api_key', '', 'string', 'PolygonScan API key'),
    ('setting_usdt_trongrid_key', 'usdt', 'trongrid_api_key', '', 'string', 'TronGrid API key'),
    ('setting_usdt_payment_ttl', 'usdt', 'payment_ttl_minutes', '10', 'int', 'Payment expiration time in minutes'),
    ('setting_usdt_pol_confirms', 'usdt', 'pol_confirmations', '12', 'int', 'Required Polygon block confirmations'),
    ('setting_usdt_trc_confirms', 'usdt', 'trc_confirmations', '19', 'int', 'Required Tron block confirmations');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM system_settings WHERE category = 'usdt';
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS usdt_amount_suffixes;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS payments;
-- +goose StatementEnd
