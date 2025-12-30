-- +goose Up
ALTER TABLE forward_agents
    ADD COLUMN agent_version VARCHAR(50) DEFAULT NULL AFTER group_id,
    ADD COLUMN platform VARCHAR(20) DEFAULT NULL AFTER agent_version,
    ADD COLUMN arch VARCHAR(20) DEFAULT NULL AFTER platform;

ALTER TABLE nodes
    ADD COLUMN agent_version VARCHAR(50) DEFAULT NULL AFTER public_ipv6,
    ADD COLUMN platform VARCHAR(20) DEFAULT NULL AFTER agent_version,
    ADD COLUMN arch VARCHAR(20) DEFAULT NULL AFTER platform;

-- +goose Down
ALTER TABLE forward_agents
    DROP COLUMN agent_version,
    DROP COLUMN platform,
    DROP COLUMN arch;

ALTER TABLE nodes
    DROP COLUMN agent_version,
    DROP COLUMN platform,
    DROP COLUMN arch;
