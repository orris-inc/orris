-- +goose Up
-- +goose StatementBegin
DROP INDEX idx_node_group_node ON node_group_nodes;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_node_group_plan ON node_group_plans;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE INDEX idx_node_group_node ON node_group_nodes(node_group_id, node_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_node_group_plan ON node_group_plans(node_group_id, subscription_plan_id);
-- +goose StatementEnd
