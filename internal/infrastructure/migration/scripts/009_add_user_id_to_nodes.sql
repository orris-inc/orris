-- +goose Up
ALTER TABLE nodes ADD COLUMN user_id BIGINT UNSIGNED NULL COMMENT 'Owner user ID (NULL = admin created)';

-- +goose StatementBegin
CREATE INDEX idx_nodes_user_id ON nodes(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_nodes_user_id ON nodes;
-- +goose StatementEnd

ALTER TABLE nodes DROP COLUMN user_id;
