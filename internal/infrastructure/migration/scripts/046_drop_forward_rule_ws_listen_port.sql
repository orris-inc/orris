-- +goose Up
-- Drop deprecated ws_listen_port column (exit rule type has been removed)
-- WsListenPort is now reported by agents in status updates and stored in Redis cache
ALTER TABLE forward_rules DROP COLUMN ws_listen_port;

-- +goose Down
-- Restore ws_listen_port column for rollback
ALTER TABLE forward_rules ADD COLUMN ws_listen_port SMALLINT UNSIGNED NULL DEFAULT NULL AFTER exit_agent_id;
