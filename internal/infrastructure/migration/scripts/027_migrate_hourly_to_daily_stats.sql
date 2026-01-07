-- +goose Up
-- Migration: Migrate hourly usage data to daily stats
-- This migration aggregates subscription_usages (hourly) into subscription_usage_stats (daily)
-- to support the new architecture where hourly data is stored in Redis with 24h TTL

-- Insert aggregated daily stats from hourly data
-- Using CONVERT_TZ to properly group by business timezone (Asia/Shanghai) date
-- IGNORE skips records that already exist in daily stats (created by scheduler)
-- This ensures we only fill in gaps, not duplicate existing aggregated data
INSERT IGNORE INTO subscription_usage_stats (
    sid,
    subscription_id,
    resource_type,
    resource_id,
    upload,
    download,
    total,
    granularity,
    period,
    created_at,
    updated_at
)
SELECT
    CONCAT('usagestat_', LOWER(REPLACE(UUID(), '-', ''))) as sid,
    subscription_id,
    resource_type,
    resource_id,
    SUM(upload) as upload,
    SUM(download) as download,
    SUM(total) as total,
    'daily' as granularity,
    DATE(CONVERT_TZ(period, '+00:00', '+08:00')) as period,
    NOW() as created_at,
    NOW() as updated_at
FROM subscription_usages
GROUP BY subscription_id, resource_type, resource_id, DATE(CONVERT_TZ(period, '+00:00', '+08:00'));

-- Note: If subscription_usages table is very large (millions of rows), consider:
-- 1. Run during low-traffic period
-- 2. Or manually batch by date range:
--    WHERE period >= '2024-01-01' AND period < '2024-02-01'

-- +goose Down
-- Note: Down migration only removes records created by this migration
-- It cannot perfectly restore the original state since we're aggregating data
-- This is a best-effort rollback that removes daily stats that might have been created

-- We cannot reliably identify which daily stats were created by this migration
-- vs. those created by the normal daily aggregation scheduler.
-- Therefore, the down migration is intentionally left as a no-op to prevent data loss.
-- If rollback is needed, manual intervention is required.

SELECT 'WARNING: Down migration is a no-op. Manual cleanup may be required.' as message;
