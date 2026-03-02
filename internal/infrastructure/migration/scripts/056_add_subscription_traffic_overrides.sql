-- +goose Up
-- Add traffic override fields for admin subscription editing
ALTER TABLE subscriptions ADD COLUMN traffic_limit_override BIGINT UNSIGNED NULL COMMENT 'override plan traffic limit (nil=use plan default)';
ALTER TABLE subscriptions ADD COLUMN traffic_used_adjustment BIGINT NOT NULL DEFAULT 0 COMMENT 'adjustment to actual traffic usage';

-- Purge historical soft-deleted records from tables switched to hard delete
DELETE FROM node_anytls_configs WHERE deleted_at IS NOT NULL;
DELETE FROM node_hysteria2_configs WHERE deleted_at IS NOT NULL;
DELETE FROM node_shadowsocks_configs WHERE deleted_at IS NOT NULL;
DELETE FROM node_trojan_configs WHERE deleted_at IS NOT NULL;
DELETE FROM node_tuic_configs WHERE deleted_at IS NOT NULL;
DELETE FROM node_vless_configs WHERE deleted_at IS NOT NULL;
DELETE FROM node_vmess_configs WHERE deleted_at IS NOT NULL;
DELETE FROM plans WHERE deleted_at IS NOT NULL;
DELETE FROM plan_pricings WHERE deleted_at IS NOT NULL;
DELETE FROM forward_rules WHERE deleted_at IS NOT NULL;

-- +goose Down
ALTER TABLE subscriptions DROP COLUMN traffic_used_adjustment;
ALTER TABLE subscriptions DROP COLUMN traffic_limit_override;
