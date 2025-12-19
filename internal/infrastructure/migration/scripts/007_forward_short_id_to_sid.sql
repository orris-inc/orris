-- +goose Up
-- Migration: Rename short_id to sid and add prefix for forward_agents and forward_rules
-- Description: Standardize Stripe-style ID naming convention with prefix stored in database

-- Step 1: forward_agents - Rename column and update data with prefix
ALTER TABLE forward_agents CHANGE COLUMN short_id sid VARCHAR(20) NOT NULL;

-- Update existing data to add 'fa_' prefix (only if not already prefixed)
UPDATE forward_agents SET sid = CONCAT('fa_', sid) WHERE sid NOT LIKE 'fa\_%';

-- Rename index
ALTER TABLE forward_agents DROP INDEX idx_forward_agent_short_id;
CREATE UNIQUE INDEX idx_forward_agent_sid ON forward_agents (sid);

-- Step 2: forward_rules - Rename column and update data with prefix
ALTER TABLE forward_rules CHANGE COLUMN short_id sid VARCHAR(20) NOT NULL;

-- Update existing data to add 'fr_' prefix (only if not already prefixed)
UPDATE forward_rules SET sid = CONCAT('fr_', sid) WHERE sid NOT LIKE 'fr\_%';

-- Rename index
ALTER TABLE forward_rules DROP INDEX idx_forward_rule_short_id;
CREATE UNIQUE INDEX idx_forward_rule_sid ON forward_rules (sid);

-- +goose Down
-- Rollback: Rename sid back to short_id and remove prefix

-- Step 1: forward_agents - Remove prefix and rename column
UPDATE forward_agents SET sid = SUBSTRING(sid, 4) WHERE sid LIKE 'fa\_%';
ALTER TABLE forward_agents CHANGE COLUMN sid short_id VARCHAR(16) NOT NULL;

-- Restore index
ALTER TABLE forward_agents DROP INDEX idx_forward_agent_sid;
CREATE UNIQUE INDEX idx_forward_agent_short_id ON forward_agents (short_id);

-- Step 2: forward_rules - Remove prefix and rename column
UPDATE forward_rules SET sid = SUBSTRING(sid, 4) WHERE sid LIKE 'fr\_%';
ALTER TABLE forward_rules CHANGE COLUMN sid short_id VARCHAR(16) NOT NULL;

-- Restore index
ALTER TABLE forward_rules DROP INDEX idx_forward_rule_sid;
CREATE UNIQUE INDEX idx_forward_rule_short_id ON forward_rules (short_id);
