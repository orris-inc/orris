-- +goose Up
-- Migration: Add new fields to subscriptions table
-- Created: 2025-12-19
-- Description: Add uuid, link_token, subject_type, and subject_id fields

-- Add uuid column (internal unique identifier)
ALTER TABLE subscriptions
ADD COLUMN uuid VARCHAR(36) NOT NULL DEFAULT '' COMMENT 'unique identifier for internal use' AFTER sid;

-- Add link_token column (secure token for subscription link authentication, 256 bits)
ALTER TABLE subscriptions
ADD COLUMN link_token VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'secure token for subscription link authentication (256 bits, resettable)' AFTER uuid;

-- Add subject_type column (type of subject: user, user_group, etc.)
ALTER TABLE subscriptions
ADD COLUMN subject_type VARCHAR(20) NOT NULL DEFAULT 'user' COMMENT 'type of subject (user, user_group, etc.)' AFTER user_id;

-- Add subject_id column (ID of the subject)
ALTER TABLE subscriptions
ADD COLUMN subject_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'ID of the subject' AFTER subject_type;

-- Generate data for existing subscriptions
UPDATE subscriptions SET uuid = UUID() WHERE uuid = '';
UPDATE subscriptions SET link_token = REPLACE(TO_BASE64(RANDOM_BYTES(32)), '=', '') WHERE link_token = '';
UPDATE subscriptions SET subject_id = user_id WHERE subject_id = 0;

-- Add indexes
ALTER TABLE subscriptions
ADD UNIQUE INDEX idx_subscription_uuid (uuid),
ADD UNIQUE INDEX idx_subscription_link_token (link_token),
ADD INDEX idx_subscription_subject (subject_type, subject_id);

-- +goose Down
-- Rollback: Remove all added columns

ALTER TABLE subscriptions DROP INDEX idx_subscription_subject;
ALTER TABLE subscriptions DROP INDEX idx_subscription_link_token;
ALTER TABLE subscriptions DROP INDEX idx_subscription_uuid;
ALTER TABLE subscriptions DROP COLUMN subject_id;
ALTER TABLE subscriptions DROP COLUMN subject_type;
ALTER TABLE subscriptions DROP COLUMN link_token;
ALTER TABLE subscriptions DROP COLUMN uuid;
