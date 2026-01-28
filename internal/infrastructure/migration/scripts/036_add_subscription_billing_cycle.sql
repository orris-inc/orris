-- +goose Up
ALTER TABLE subscriptions
ADD COLUMN billing_cycle VARCHAR(20) DEFAULT NULL;

CREATE INDEX idx_subscriptions_billing_cycle ON subscriptions(billing_cycle);

-- +goose Down
DROP INDEX idx_subscriptions_billing_cycle ON subscriptions;
ALTER TABLE subscriptions DROP COLUMN billing_cycle;
