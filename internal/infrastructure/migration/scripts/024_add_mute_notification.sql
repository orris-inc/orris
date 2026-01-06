-- +goose Up
-- +goose StatementBegin
ALTER TABLE forward_agents ADD COLUMN mute_notification BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE nodes ADD COLUMN mute_notification BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE nodes DROP COLUMN mute_notification;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE forward_agents DROP COLUMN mute_notification;
-- +goose StatementEnd
