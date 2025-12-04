-- +goose Up
-- +goose StatementBegin
DROP INDEX idx_user_period ON subscription_traffic;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE subscription_traffic DROP COLUMN user_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE subscription_traffic ADD COLUMN user_id BIGINT UNSIGNED DEFAULT NULL AFTER node_id;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_user_period ON subscription_traffic (user_id, period);
-- +goose StatementEnd
