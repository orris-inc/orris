-- +goose Up

-- First, permanently delete any soft-deleted records
DELETE FROM forward_agents WHERE deleted_at IS NOT NULL;
DELETE FROM nodes WHERE deleted_at IS NOT NULL;

-- Remove deleted_at columns (indexes are automatically dropped with the column)
ALTER TABLE forward_agents DROP COLUMN deleted_at;
ALTER TABLE nodes DROP COLUMN deleted_at;

-- +goose Down

-- Re-add deleted_at columns with index
ALTER TABLE forward_agents ADD COLUMN deleted_at TIMESTAMP NULL, ADD INDEX idx_forward_agents_deleted_at (deleted_at);
ALTER TABLE nodes ADD COLUMN deleted_at TIMESTAMP NULL, ADD INDEX idx_nodes_deleted_at (deleted_at);
