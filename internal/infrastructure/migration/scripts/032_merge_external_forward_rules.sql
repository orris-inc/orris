-- +goose Up
-- Merge ExternalForwardRule into ForwardRule with rule_type='external'

-- Step 1: Add new columns for external rule fields
ALTER TABLE forward_rules
    ADD COLUMN server_address VARCHAR(255) DEFAULT NULL COMMENT 'server address for external rules',
    ADD COLUMN external_source VARCHAR(50) DEFAULT NULL COMMENT 'external source identifier',
    ADD COLUMN external_rule_id VARCHAR(100) DEFAULT NULL COMMENT 'external rule reference ID';

-- Step 2: Create index for external rules
CREATE INDEX idx_forward_rules_external ON forward_rules(external_source, external_rule_id);

-- Step 3: Migrate data from external_forward_rules to forward_rules
-- Convert efr_xxx SID to fr_xxx format
INSERT INTO forward_rules (
    sid, agent_id, user_id, subscription_id, rule_type,
    name, listen_port, target_node_id, server_address,
    external_source, external_rule_id,
    protocol, ip_version, status, sort_order, remark, group_ids,
    created_at, updated_at, deleted_at
)
SELECT
    CONCAT('fr_', SUBSTRING(sid, 5)) AS sid,
    0 AS agent_id,
    user_id,
    subscription_id,
    'external' AS rule_type,
    name,
    listen_port,
    node_id AS target_node_id,
    server_address,
    external_source,
    external_rule_id,
    'tcp' AS protocol,
    'auto' AS ip_version,
    status,
    sort_order,
    remark,
    group_ids,
    created_at,
    updated_at,
    deleted_at
FROM external_forward_rules;

-- Step 4: Drop old table
DROP TABLE IF EXISTS external_forward_rules;

-- +goose Down
-- Recreate external_forward_rules table and migrate data back

-- Step 1: Recreate external_forward_rules table
CREATE TABLE external_forward_rules (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    sid VARCHAR(50) NOT NULL,
    subscription_id BIGINT UNSIGNED DEFAULT NULL,
    user_id BIGINT UNSIGNED DEFAULT NULL,
    node_id BIGINT UNSIGNED DEFAULT NULL,
    name VARCHAR(100) NOT NULL,
    server_address VARCHAR(255) NOT NULL,
    listen_port SMALLINT UNSIGNED NOT NULL,
    external_source VARCHAR(50) NOT NULL,
    external_rule_id VARCHAR(100) DEFAULT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'enabled',
    sort_order INT NOT NULL DEFAULT 0,
    remark VARCHAR(500) DEFAULT NULL,
    group_ids JSON DEFAULT NULL,
    created_at DATETIME(3) DEFAULT NULL,
    updated_at DATETIME(3) DEFAULT NULL,
    deleted_at DATETIME(3) DEFAULT NULL,
    UNIQUE INDEX idx_external_forward_rule_sid (sid),
    INDEX idx_external_forward_rules_subscription_id (subscription_id),
    INDEX idx_external_forward_rules_user_id (user_id),
    INDEX idx_external_forward_rules_node_id (node_id),
    INDEX idx_external_forward_rules_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Step 2: Migrate external rules back to external_forward_rules
INSERT INTO external_forward_rules (
    sid, subscription_id, user_id, node_id, name, server_address, listen_port,
    external_source, external_rule_id, status, sort_order, remark, group_ids,
    created_at, updated_at, deleted_at
)
SELECT
    CONCAT('efr_', SUBSTRING(sid, 4)) AS sid,
    subscription_id,
    user_id,
    target_node_id AS node_id,
    name,
    server_address,
    listen_port,
    external_source,
    external_rule_id,
    status,
    sort_order,
    remark,
    group_ids,
    created_at,
    updated_at,
    deleted_at
FROM forward_rules
WHERE rule_type = 'external';

-- Step 3: Delete external rules from forward_rules
DELETE FROM forward_rules WHERE rule_type = 'external';

-- Step 4: Drop new columns and index
DROP INDEX idx_forward_rules_external ON forward_rules;
ALTER TABLE forward_rules
    DROP COLUMN server_address,
    DROP COLUMN external_source,
    DROP COLUMN external_rule_id;
