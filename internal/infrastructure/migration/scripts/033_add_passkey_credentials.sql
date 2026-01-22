-- +goose Up
-- Migration: Add passkey credentials table for WebAuthn/Passkey authentication

CREATE TABLE passkey_credentials (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    sid VARCHAR(50) NOT NULL COMMENT 'Stripe-style ID: pk_xxxxxxxx',
    user_id BIGINT UNSIGNED NOT NULL COMMENT 'Reference to users.id',
    credential_id VARBINARY(1024) NOT NULL COMMENT 'WebAuthn credential ID (raw bytes)',
    public_key BLOB NOT NULL COMMENT 'COSE-encoded public key',
    attestation_type VARCHAR(50) NOT NULL DEFAULT 'none' COMMENT 'Attestation format: none, packed, etc.',
    aaguid VARBINARY(16) COMMENT 'Authenticator Attestation GUID',
    sign_count INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Signature counter for clone detection',
    backup_eligible BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'WebAuthn BE flag: credential can be backed up',
    backup_state BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'WebAuthn BS flag: credential is currently backed up',
    transports JSON COMMENT 'Transport hints: usb, internal, hybrid, ble, nfc',
    device_name VARCHAR(100) NOT NULL DEFAULT '' COMMENT 'User-friendly device name',
    last_used_at TIMESTAMP NULL DEFAULT NULL COMMENT 'Last successful authentication time',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_passkey_credentials_sid (sid),
    UNIQUE INDEX idx_passkey_credentials_credential_id (credential_id(255)),
    INDEX idx_passkey_credentials_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS passkey_credentials;
