-- +goose Up
-- Performance indexes for JSON column queries and usage statistics

-- Helper procedure to drop an index if it exists (works on older MySQL versions)
-- +goose StatementBegin
DROP PROCEDURE IF EXISTS drop_index_if_exists;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE PROCEDURE drop_index_if_exists(IN p_table VARCHAR(64), IN p_index VARCHAR(64))
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.STATISTICS
        WHERE table_schema = DATABASE()
          AND table_name = p_table
          AND index_name = p_index
    ) THEN
        SET @stmt = CONCAT('DROP INDEX `', p_index, '` ON `', p_table, '`');
        PREPARE pstmt FROM @stmt;
        EXECUTE pstmt;
        DEALLOCATE PREPARE pstmt;
    END IF;
END;
-- +goose StatementEnd

-- Drop existing indexes if any (idempotency for partial runs)
-- +goose StatementBegin
CALL drop_index_if_exists('forward_rules', 'idx_forward_rules_group_ids');
-- +goose StatementEnd
-- +goose StatementBegin
CALL drop_index_if_exists('nodes', 'idx_nodes_group_ids');
-- +goose StatementEnd
-- +goose StatementBegin
CALL drop_index_if_exists('forward_agents', 'idx_forward_agents_group_ids');
-- +goose StatementEnd
-- +goose StatementBegin
CALL drop_index_if_exists('subscription_usages', 'idx_subscription_usages_resource_sub_period');
-- +goose StatementEnd

-- Clean up helper procedure before creating multi-valued indexes
-- (stored procedures cannot coexist with CAST(.. AS .. ARRAY) statements in some MySQL versions)
-- +goose StatementBegin
DROP PROCEDURE IF EXISTS drop_index_if_exists;
-- +goose StatementEnd

-- Multi-Valued Index for forward_rules.group_ids JSON column
-- NOTE: CAST(.. AS .. ARRAY) cannot be used inside stored procedures (MySQL limitation)
CREATE INDEX idx_forward_rules_group_ids ON forward_rules ((CAST(group_ids->'$[*]' AS UNSIGNED ARRAY)));

-- Multi-Valued Index for nodes.group_ids JSON column
CREATE INDEX idx_nodes_group_ids ON nodes ((CAST(group_ids->'$[*]' AS UNSIGNED ARRAY)));

-- Multi-Valued Index for forward_agents.group_ids JSON column
CREATE INDEX idx_forward_agents_group_ids ON forward_agents ((CAST(group_ids->'$[*]' AS UNSIGNED ARRAY)));

-- Composite index for subscription_usages queries
CREATE INDEX idx_subscription_usages_resource_sub_period ON subscription_usages (resource_type, subscription_id, period);

-- +goose Down
-- +goose StatementBegin
DROP PROCEDURE IF EXISTS drop_index_if_exists;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE PROCEDURE drop_index_if_exists(IN p_table VARCHAR(64), IN p_index VARCHAR(64))
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.STATISTICS
        WHERE table_schema = DATABASE()
          AND table_name = p_table
          AND index_name = p_index
    ) THEN
        SET @stmt = CONCAT('DROP INDEX `', p_index, '` ON `', p_table, '`');
        PREPARE pstmt FROM @stmt;
        EXECUTE pstmt;
        DEALLOCATE PREPARE pstmt;
    END IF;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CALL drop_index_if_exists('forward_rules', 'idx_forward_rules_group_ids');
-- +goose StatementEnd
-- +goose StatementBegin
CALL drop_index_if_exists('nodes', 'idx_nodes_group_ids');
-- +goose StatementEnd
-- +goose StatementBegin
CALL drop_index_if_exists('forward_agents', 'idx_forward_agents_group_ids');
-- +goose StatementEnd
-- +goose StatementBegin
CALL drop_index_if_exists('subscription_usages', 'idx_subscription_usages_resource_sub_period');
-- +goose StatementEnd

-- +goose StatementBegin
DROP PROCEDURE IF EXISTS drop_index_if_exists;
-- +goose StatementEnd
