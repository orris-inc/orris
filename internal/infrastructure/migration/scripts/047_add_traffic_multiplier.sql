-- +goose Up
-- +goose StatementBegin
ALTER TABLE forward_rules
ADD COLUMN traffic_multiplier DECIMAL(10,4) NULL DEFAULT NULL
COMMENT 'Traffic multiplier for display. NULL means auto-calculate based on node count';

CREATE INDEX idx_forward_rules_traffic_multiplier ON forward_rules(traffic_multiplier);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_forward_rules_traffic_multiplier ON forward_rules;
ALTER TABLE forward_rules DROP COLUMN traffic_multiplier;
-- +goose StatementEnd
