-- +goose Up
-- Add SID column to announcements table for external-facing Stripe-style IDs
ALTER TABLE announcements ADD COLUMN sid VARCHAR(50) NOT NULL DEFAULT '' AFTER id;

-- Generate SIDs for existing announcements
UPDATE announcements SET sid = CONCAT('ann_', SUBSTRING(MD5(CONCAT(id, RAND())), 1, 12)) WHERE sid = '';

-- Add unique index for SID lookups
ALTER TABLE announcements ADD UNIQUE INDEX idx_announcements_sid (sid);

-- +goose Down
ALTER TABLE announcements DROP INDEX idx_announcements_sid;
ALTER TABLE announcements DROP COLUMN sid;
