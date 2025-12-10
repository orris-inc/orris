-- +goose Up

-- Add short_id column to nodes table for Stripe-style prefixed IDs (node_xxx)
ALTER TABLE nodes
    ADD COLUMN short_id VARCHAR(20) UNIQUE COMMENT 'Short ID for external API (node_xxx format)' AFTER id;

-- Generate short_id for existing nodes using random 12-character alphanumeric strings
-- This uses MySQL's random functions to generate unique IDs
UPDATE nodes SET short_id = CONCAT(
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1)
) WHERE short_id IS NULL;

-- Make short_id NOT NULL after populating
ALTER TABLE nodes MODIFY COLUMN short_id VARCHAR(20) NOT NULL;

-- +goose Down

ALTER TABLE nodes
    DROP COLUMN short_id;
