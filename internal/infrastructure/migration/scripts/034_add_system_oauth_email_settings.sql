-- +goose Up
-- Migration: Add system, OAuth, and email settings

-- System settings
-- Note: api_base_url and timezone are READ-ONLY, can only be configured via environment variable
INSERT INTO system_settings (sid, category, setting_key, value_type, description) VALUES
    ('setting_sys_api_base_url', 'system', 'api_base_url', 'string', 'API server base URL (READ-ONLY, set via SERVER_BASE_URL env var)'),
    ('setting_sys_sub_base_url', 'system', 'subscription_base_url', 'string', 'Subscription link base URL (falls back to api_base_url if empty)'),
    ('setting_sys_frontend_url', 'system', 'frontend_url', 'string', 'Frontend callback URL for OAuth redirects'),
    ('setting_sys_timezone', 'system', 'timezone', 'string', 'Business timezone (READ-ONLY, set via SERVER_TIMEZONE env var)');

-- Google OAuth settings
INSERT INTO system_settings (sid, category, setting_key, value_type, description) VALUES
    ('setting_oauth_google_client_id', 'oauth_google', 'client_id', 'string', 'Google OAuth Client ID'),
    ('setting_oauth_google_client_secret', 'oauth_google', 'client_secret', 'string', 'Google OAuth Client Secret'),
    ('setting_oauth_google_redirect_url', 'oauth_google', 'redirect_url', 'string', 'Google OAuth redirect URL (auto-generated if empty)');

-- GitHub OAuth settings
INSERT INTO system_settings (sid, category, setting_key, value_type, description) VALUES
    ('setting_oauth_github_client_id', 'oauth_github', 'client_id', 'string', 'GitHub OAuth Client ID'),
    ('setting_oauth_github_client_secret', 'oauth_github', 'client_secret', 'string', 'GitHub OAuth Client Secret'),
    ('setting_oauth_github_redirect_url', 'oauth_github', 'redirect_url', 'string', 'GitHub OAuth redirect URL (auto-generated if empty)');

-- Email settings
INSERT INTO system_settings (sid, category, setting_key, value_type, description) VALUES
    ('setting_email_smtp_host', 'email', 'smtp_host', 'string', 'SMTP server hostname'),
    ('setting_email_smtp_port', 'email', 'smtp_port', 'int', 'SMTP server port'),
    ('setting_email_smtp_user', 'email', 'smtp_user', 'string', 'SMTP authentication username'),
    ('setting_email_smtp_password', 'email', 'smtp_password', 'string', 'SMTP authentication password'),
    ('setting_email_from_address', 'email', 'from_address', 'string', 'Email sender address'),
    ('setting_email_from_name', 'email', 'from_name', 'string', 'Email sender display name');

-- +goose Down
DELETE FROM system_settings WHERE category IN ('system', 'oauth_google', 'oauth_github', 'email');
