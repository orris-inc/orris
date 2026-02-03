-- +goose Up
ALTER TABLE forward_agents ADD COLUMN expires_at DATETIME NULL;

-- +goose StatementBegin
ALTER TABLE forward_agents ADD COLUMN renewal_amount DECIMAL(10,2) NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE nodes ADD COLUMN expires_at DATETIME NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE nodes ADD COLUMN renewal_amount DECIMAL(10,2) NULL;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE nodes DROP COLUMN renewal_amount;

-- +goose StatementBegin
ALTER TABLE nodes DROP COLUMN expires_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE forward_agents DROP COLUMN renewal_amount;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE forward_agents DROP COLUMN expires_at;
-- +goose StatementEnd
