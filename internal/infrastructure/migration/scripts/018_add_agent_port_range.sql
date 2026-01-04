-- +goose Up
-- +goose StatementBegin
ALTER TABLE forward_agents
    ADD COLUMN allowed_port_range TEXT DEFAULT NULL AFTER arch;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE forward_agents
    DROP COLUMN allowed_port_range;
-- +goose StatementEnd
