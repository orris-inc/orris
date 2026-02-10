package setting

import (
	"context"

	sharedConfig "github.com/orris-inc/orris/internal/shared/config"
)

// ConfigValue represents a configuration value with its source information.
// This is a domain-level representation used by infrastructure services
// that need access to configuration with source tracking.
type ConfigValue struct {
	Value  any
	Source string // "database", "environment", or "default"
}

// SettingProvider defines the interface for providing hot-reloadable configuration.
// Infrastructure services depend on this interface instead of the concrete
// application-layer SettingProvider, following the dependency inversion principle.
type SettingProvider interface {
	// GetEmailConfig returns the merged email configuration.
	// Database values take precedence over environment variables.
	GetEmailConfig(ctx context.Context) sharedConfig.EmailConfig

	// GetAPIBaseURL returns the API base URL with source tracking.
	GetAPIBaseURL(ctx context.Context) ConfigValue

	// GetGoogleOAuthConfig returns the merged Google OAuth configuration.
	GetGoogleOAuthConfig(ctx context.Context) sharedConfig.GoogleOAuthConfig

	// GetGitHubOAuthConfig returns the merged GitHub OAuth configuration.
	GetGitHubOAuthConfig(ctx context.Context) sharedConfig.GitHubOAuthConfig

	// GetTelegramConfig returns the merged Telegram configuration.
	GetTelegramConfig(ctx context.Context) sharedConfig.TelegramConfig

	// IsTelegramEnabled checks if Telegram is enabled.
	IsTelegramEnabled(ctx context.Context) bool
}
