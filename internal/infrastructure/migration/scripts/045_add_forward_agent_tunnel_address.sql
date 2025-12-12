-- +goose Up
-- +goose StatementBegin
ALTER TABLE forward_agents ADD COLUMN tunnel_address VARCHAR(255) DEFAULT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE forward_agents DROP COLUMN tunnel_address;
-- +goose StatementEnd
