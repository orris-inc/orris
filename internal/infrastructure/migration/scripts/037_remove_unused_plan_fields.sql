-- +goose Up
-- Remove unused plan fields: api_rate_limit, max_users, max_projects
ALTER TABLE plans DROP COLUMN api_rate_limit;
ALTER TABLE plans DROP COLUMN max_users;
ALTER TABLE plans DROP COLUMN max_projects;

-- +goose Down
ALTER TABLE plans ADD COLUMN api_rate_limit INT UNSIGNED DEFAULT 60;
ALTER TABLE plans ADD COLUMN max_users INT UNSIGNED DEFAULT 1;
ALTER TABLE plans ADD COLUMN max_projects INT UNSIGNED DEFAULT 1;
