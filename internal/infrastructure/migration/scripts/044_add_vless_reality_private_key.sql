-- +goose Up
-- +goose StatementBegin
ALTER TABLE node_vless_configs ADD COLUMN private_key VARCHAR(255) DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE node_vless_configs DROP COLUMN private_key;
-- +goose StatementEnd
