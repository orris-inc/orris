-- +goose Up
-- Add short_id field to forward_rules for external API exposure (Stripe-style ID)

ALTER TABLE forward_rules
ADD COLUMN short_id VARCHAR(16) NOT NULL DEFAULT '' AFTER id;

-- Generate short_id for existing records using random base62 string
-- Using MD5 hash of id + created_at as seed for deterministic generation
UPDATE forward_rules
SET short_id = SUBSTRING(
    REPLACE(REPLACE(REPLACE(
        TO_BASE64(UNHEX(MD5(CONCAT(id, created_at, RAND())))),
        '+', 'A'), '/', 'B'), '=', ''),
    1, 12
)
WHERE short_id = '';

-- Add unique index
ALTER TABLE forward_rules
ADD UNIQUE INDEX idx_forward_rule_short_id (short_id);

-- +goose Down
ALTER TABLE forward_rules DROP INDEX idx_forward_rule_short_id;
ALTER TABLE forward_rules DROP COLUMN short_id;
