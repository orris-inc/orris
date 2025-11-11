-- +goose Up
-- +goose StatementBegin
-- Create subscription_plan_pricing table to support multiple pricing tiers per plan
CREATE TABLE IF NOT EXISTS subscription_plan_pricing (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    plan_id BIGINT UNSIGNED NOT NULL COMMENT 'Reference to subscription_plans table',
    billing_cycle VARCHAR(20) NOT NULL COMMENT 'Billing cycle: weekly, monthly, quarterly, semi_annual, yearly, lifetime',
    price BIGINT UNSIGNED NOT NULL COMMENT 'Price in smallest currency unit (cents)',
    currency VARCHAR(3) NOT NULL DEFAULT 'CNY' COMMENT 'Currency code: CNY, USD, EUR, GBP, JPY',
    is_active BOOLEAN NOT NULL DEFAULT TRUE COMMENT 'Whether this pricing option is active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,

    -- Unique constraint: one pricing per (plan_id, billing_cycle) combination
    UNIQUE KEY uk_plan_billing_cycle (plan_id, billing_cycle),

    -- Indexes for query performance
    INDEX idx_plan_id (plan_id),
    INDEX idx_billing_cycle (billing_cycle),
    INDEX idx_is_active (is_active),
    INDEX idx_deleted_at (deleted_at),

    -- Foreign key constraint with cascade delete
    CONSTRAINT fk_pricing_plan_id
        FOREIGN KEY (plan_id)
        REFERENCES subscription_plans(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Subscription plan pricing options for different billing cycles';
-- +goose StatementEnd

-- +goose StatementBegin
-- Migrate existing data from subscription_plans to subscription_plan_pricing
INSERT INTO subscription_plan_pricing (plan_id, billing_cycle, price, currency, is_active, created_at, updated_at)
SELECT
    id,
    billing_cycle,
    price,
    currency,
    CASE WHEN status = 'active' THEN TRUE ELSE FALSE END,
    created_at,
    updated_at
FROM subscription_plans
WHERE deleted_at IS NULL
  AND billing_cycle IS NOT NULL
  AND price > 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop the subscription_plan_pricing table
DROP TABLE IF NOT EXISTS subscription_plan_pricing;

-- Remove the added index if it was created
-- ALTER TABLE subscription_plans DROP INDEX IF EXISTS idx_billing_cycle_compat;

-- +goose StatementEnd
