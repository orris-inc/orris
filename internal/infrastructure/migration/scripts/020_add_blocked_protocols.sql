-- +goose Up
-- MySQL doesn't support ADD/DROP COLUMN IF EXISTS, so use a procedure
-- +goose StatementBegin
DROP PROCEDURE IF EXISTS migrate_blocked_protocols;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE PROCEDURE migrate_blocked_protocols()
BEGIN
    -- Add blocked_protocols to forward_agents if not exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = DATABASE()
        AND table_name = 'forward_agents'
        AND column_name = 'blocked_protocols'
    ) THEN
        ALTER TABLE forward_agents ADD COLUMN blocked_protocols JSON DEFAULT NULL;
    END IF;

    -- Remove blocked_protocols from forward_rules if exists
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = DATABASE()
        AND table_name = 'forward_rules'
        AND column_name = 'blocked_protocols'
    ) THEN
        ALTER TABLE forward_rules DROP COLUMN blocked_protocols;
    END IF;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CALL migrate_blocked_protocols();
-- +goose StatementEnd

-- +goose StatementBegin
DROP PROCEDURE IF EXISTS migrate_blocked_protocols;
-- +goose StatementEnd

-- +goose Down
-- Restore blocked_protocols to forward_rules table
ALTER TABLE forward_rules ADD COLUMN blocked_protocols JSON DEFAULT NULL;

-- Remove blocked_protocols from forward_agents table
ALTER TABLE forward_agents DROP COLUMN blocked_protocols;
