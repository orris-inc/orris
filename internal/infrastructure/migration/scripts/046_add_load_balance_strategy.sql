-- +goose Up
-- +goose StatementBegin
ALTER TABLE forward_rules ADD COLUMN load_balance_strategy VARCHAR(32) DEFAULT 'failover';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE forward_rules DROP COLUMN load_balance_strategy;
-- +goose StatementEnd
