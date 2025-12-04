-- +goose Up
-- +goose StatementBegin
ALTER TABLE nodes CHANGE COLUMN server_port agent_port SMALLINT UNSIGNED NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE nodes ADD COLUMN subscription_port SMALLINT UNSIGNED DEFAULT NULL AFTER agent_port;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_server ON nodes;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_agent_address ON nodes (server_address, agent_port);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_agent_address ON nodes;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_server ON nodes (server_address, agent_port);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE nodes DROP COLUMN subscription_port;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE nodes CHANGE COLUMN agent_port server_port SMALLINT UNSIGNED NOT NULL;
-- +goose StatementEnd
