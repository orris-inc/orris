-- +goose Up
-- +goose StatementBegin
ALTER TABLE forward_rules ADD COLUMN bind_ip VARCHAR(45) DEFAULT '' AFTER target_node_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE forward_rules DROP COLUMN bind_ip;
-- +goose StatementEnd
