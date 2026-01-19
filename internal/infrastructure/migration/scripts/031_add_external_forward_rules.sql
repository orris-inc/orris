-- +goose Up
-- +goose StatementBegin
CREATE TABLE external_forward_rules (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    sid VARCHAR(50) NOT NULL UNIQUE,
    subscription_id BIGINT UNSIGNED NULL,
    user_id BIGINT UNSIGNED NULL,
    node_id BIGINT UNSIGNED NULL,

    -- Rule metadata
    name VARCHAR(100) NOT NULL,
    server_address VARCHAR(255) NOT NULL,
    listen_port SMALLINT UNSIGNED NOT NULL,

    -- External reference
    external_source VARCHAR(50) NOT NULL,
    external_rule_id VARCHAR(100),

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'enabled',
    sort_order INT NOT NULL DEFAULT 0,
    remark VARCHAR(500) DEFAULT '',

    -- Resource group association (for subscription distribution)
    group_ids JSON,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,

    INDEX idx_external_forward_rules_subscription_id (subscription_id),
    INDEX idx_external_forward_rules_user_id (user_id),
    INDEX idx_external_forward_rules_node_id (node_id),
    INDEX idx_external_forward_rules_external (external_source, external_rule_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS external_forward_rules;
-- +goose StatementEnd
