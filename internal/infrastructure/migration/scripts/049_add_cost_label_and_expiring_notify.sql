-- +goose Up
-- +goose StatementBegin
ALTER TABLE admin_telegram_bindings
    ADD COLUMN notify_resource_expiring BOOLEAN NOT NULL DEFAULT TRUE COMMENT 'Notify when resources are about to expire',
    ADD COLUMN resource_expiring_days INT NOT NULL DEFAULT 7 COMMENT 'Days before expiration to start notifying (1-30)',
    ADD COLUMN last_resource_expiring_notify_date DATE NULL COMMENT 'Date of last expiring notification for daily deduplication';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE nodes
    ADD COLUMN cost_label VARCHAR(50) NULL COMMENT 'Cost label for display',
    DROP COLUMN renewal_amount;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE forward_agents
    ADD COLUMN cost_label VARCHAR(50) NULL COMMENT 'Cost label for display',
    DROP COLUMN renewal_amount;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE forward_agents
    DROP COLUMN cost_label,
    ADD COLUMN renewal_amount DECIMAL(10,2) NULL COMMENT 'Renewal cost amount';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE nodes
    DROP COLUMN cost_label,
    ADD COLUMN renewal_amount DECIMAL(10,2) NULL COMMENT 'Renewal cost amount';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE admin_telegram_bindings
    DROP COLUMN notify_resource_expiring,
    DROP COLUMN resource_expiring_days,
    DROP COLUMN last_resource_expiring_notify_date;
-- +goose StatementEnd
