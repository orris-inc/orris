-- +goose Up
-- +goose StatementBegin

-- Delete external forward rules without target_node_id
-- External rules require target_node_id to derive protocol information
DELETE FROM forward_rules
WHERE rule_type = 'external'
  AND target_node_id IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Cannot restore deleted data
-- No-op for down migration

-- +goose StatementEnd
