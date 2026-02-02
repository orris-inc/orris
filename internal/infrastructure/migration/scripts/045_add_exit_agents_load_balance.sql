-- +goose Up
-- +goose StatementBegin
ALTER TABLE forward_rules ADD COLUMN exit_agents JSON DEFAULT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE forward_rules DROP COLUMN exit_agents;
-- +goose StatementEnd
