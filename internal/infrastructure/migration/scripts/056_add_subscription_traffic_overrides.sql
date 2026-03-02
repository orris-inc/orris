-- +goose Up
-- Add traffic override fields for admin subscription editing
ALTER TABLE subscriptions ADD COLUMN traffic_limit_override BIGINT UNSIGNED NULL COMMENT 'override plan traffic limit (nil=use plan default)';
ALTER TABLE subscriptions ADD COLUMN traffic_used_adjustment BIGINT NOT NULL DEFAULT 0 COMMENT 'adjustment to actual traffic usage';

-- +goose Down
ALTER TABLE subscriptions DROP COLUMN traffic_used_adjustment;
ALTER TABLE subscriptions DROP COLUMN traffic_limit_override;
