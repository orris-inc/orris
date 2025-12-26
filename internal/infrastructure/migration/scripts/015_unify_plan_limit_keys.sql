-- +goose Up
-- Migration: Unify plan limit keys
-- Created: 2025-12-26
-- Description: Rename forward_rule_limit to rule_limit and forward_traffic_limit to traffic_limit in plans.limits JSON

UPDATE plans
SET limits = JSON_SET(
    JSON_REMOVE(limits, '$.forward_rule_limit'),
    '$.rule_limit', JSON_EXTRACT(limits, '$.forward_rule_limit')
)
WHERE JSON_CONTAINS_PATH(limits, 'one', '$.forward_rule_limit');

UPDATE plans
SET limits = JSON_SET(
    JSON_REMOVE(limits, '$.forward_traffic_limit'),
    '$.traffic_limit', JSON_EXTRACT(limits, '$.forward_traffic_limit')
)
WHERE JSON_CONTAINS_PATH(limits, 'one', '$.forward_traffic_limit');

-- +goose Down
UPDATE plans
SET limits = JSON_SET(
    JSON_REMOVE(limits, '$.rule_limit'),
    '$.forward_rule_limit', JSON_EXTRACT(limits, '$.rule_limit')
)
WHERE JSON_CONTAINS_PATH(limits, 'one', '$.rule_limit');

UPDATE plans
SET limits = JSON_SET(
    JSON_REMOVE(limits, '$.traffic_limit'),
    '$.forward_traffic_limit', JSON_EXTRACT(limits, '$.traffic_limit')
)
WHERE JSON_CONTAINS_PATH(limits, 'one', '$.traffic_limit');
