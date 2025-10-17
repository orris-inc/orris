-- +goose Up
-- Migration: Add password authentication support
-- Created: 2025-10-17
-- Description: Add password authentication, email verification, and password reset functionality

ALTER TABLE users
ADD COLUMN password_hash VARCHAR(255) AFTER locale,
ADD COLUMN email_verification_token VARCHAR(255) AFTER password_hash,
ADD COLUMN email_verification_expires_at TIMESTAMP NULL AFTER email_verification_token,
ADD COLUMN password_reset_token VARCHAR(255) AFTER email_verification_expires_at,
ADD COLUMN password_reset_expires_at TIMESTAMP NULL AFTER password_reset_token,
ADD COLUMN last_password_change_at TIMESTAMP NULL AFTER password_reset_expires_at,
ADD COLUMN failed_login_attempts INT UNSIGNED DEFAULT 0 AFTER last_password_change_at,
ADD COLUMN locked_until TIMESTAMP NULL AFTER failed_login_attempts,
ADD INDEX idx_password_reset_token (password_reset_token),
ADD INDEX idx_email_verification_token (email_verification_token);

-- +goose Down
-- Rollback Migration: Remove password authentication support
-- Description: Remove password authentication related fields

ALTER TABLE users
DROP INDEX idx_email_verification_token,
DROP INDEX idx_password_reset_token,
DROP COLUMN locked_until,
DROP COLUMN failed_login_attempts,
DROP COLUMN last_password_change_at,
DROP COLUMN password_reset_expires_at,
DROP COLUMN password_reset_token,
DROP COLUMN email_verification_expires_at,
DROP COLUMN email_verification_token,
DROP COLUMN password_hash;
