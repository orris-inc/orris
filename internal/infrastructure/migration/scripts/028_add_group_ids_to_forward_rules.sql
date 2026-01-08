-- +goose Up
-- +goose StatementBegin
ALTER TABLE forward_rules ADD COLUMN group_ids JSON DEFAULT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE forward_rules DROP COLUMN group_ids;
-- +goose StatementEnd
