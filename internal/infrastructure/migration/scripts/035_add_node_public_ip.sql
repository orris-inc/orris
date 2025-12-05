-- +goose Up
-- Add public_ipv4 and public_ipv6 columns to nodes table for storing agent-reported public IP addresses
-- These fields are updated when agent reports status with public IP information
-- IPv4: max 15 chars (e.g., "255.255.255.255")
-- IPv6: max 45 chars (e.g., "2001:0db8:0000:0000:0000:0000:0000:0001")

ALTER TABLE nodes ADD COLUMN public_ipv4 VARCHAR(15) NULL DEFAULT NULL;
ALTER TABLE nodes ADD COLUMN public_ipv6 VARCHAR(45) NULL DEFAULT NULL;

-- +goose Down
ALTER TABLE nodes DROP COLUMN public_ipv6;
ALTER TABLE nodes DROP COLUMN public_ipv4;
