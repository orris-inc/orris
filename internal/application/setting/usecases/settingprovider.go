package usecases

import (
	"context"
	"fmt"
	"sync"

	settingDTO "github.com/orris-inc/orris/internal/application/setting/dto"
	"github.com/orris-inc/orris/internal/domain/setting"
	sharedConfig "github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SettingChangeSubscriber defines the interface for setting change subscribers
type SettingChangeSubscriber interface {
	OnSettingChange(ctx context.Context, category string, changes map[string]any) error
}

// SettingProviderConfig holds all fallback configurations from environment
type SettingProviderConfig struct {
	TelegramConfig      sharedConfig.TelegramConfig
	GoogleOAuthConfig   sharedConfig.GoogleOAuthConfig
	GitHubOAuthConfig   sharedConfig.GitHubOAuthConfig
	EmailConfig         sharedConfig.EmailConfig
	APIBaseURL          string
	SubscriptionBaseURL string
	FrontendURL         string
	Timezone            string
}

// SettingProvider provides hot-reloadable configuration with database-first, env-fallback logic
type SettingProvider struct {
	settingRepo    setting.Repository
	telegramConfig sharedConfig.TelegramConfig
	apiBaseURL     string // Server API base URL for auto-generating webhook URLs
	logger         logger.Interface

	// Extended fields for System, OAuth, Email configurations
	subscriptionBaseURL string
	frontendURL         string
	timezone            string
	googleOAuthConfig   sharedConfig.GoogleOAuthConfig
	githubOAuthConfig   sharedConfig.GitHubOAuthConfig
	emailConfig         sharedConfig.EmailConfig

	subscribers []SettingChangeSubscriber
	mu          sync.RWMutex
}

// NewSettingProvider creates a new SettingProvider
func NewSettingProvider(
	settingRepo setting.Repository,
	cfg SettingProviderConfig,
	logger logger.Interface,
) *SettingProvider {
	return &SettingProvider{
		settingRepo:         settingRepo,
		telegramConfig:      cfg.TelegramConfig,
		apiBaseURL:          cfg.APIBaseURL,
		subscriptionBaseURL: cfg.SubscriptionBaseURL,
		frontendURL:         cfg.FrontendURL,
		timezone:            cfg.Timezone,
		googleOAuthConfig:   cfg.GoogleOAuthConfig,
		githubOAuthConfig:   cfg.GitHubOAuthConfig,
		emailConfig:         cfg.EmailConfig,
		logger:              logger,
		subscribers:         make([]SettingChangeSubscriber, 0),
	}
}

// Subscribe registers a subscriber for setting changes
func (p *SettingProvider) Subscribe(subscriber SettingChangeSubscriber) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.subscribers = append(p.subscribers, subscriber)
}

// Unsubscribe removes a subscriber from the list
func (p *SettingProvider) Unsubscribe(subscriber SettingChangeSubscriber) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, s := range p.subscribers {
		if s == subscriber {
			p.subscribers = append(p.subscribers[:i], p.subscribers[i+1:]...)
			break
		}
	}
}

// NotifyChange notifies all subscribers of configuration changes
func (p *SettingProvider) NotifyChange(ctx context.Context, category string, changes map[string]any) error {
	p.mu.RLock()
	subscribers := make([]SettingChangeSubscriber, len(p.subscribers))
	copy(subscribers, p.subscribers)
	p.mu.RUnlock()

	var errs []error
	for _, subscriber := range subscribers {
		if err := subscriber.OnSettingChange(ctx, category, changes); err != nil {
			p.logger.Errorw("subscriber failed to handle setting change",
				"category", category,
				"subscriber", fmt.Sprintf("%T", subscriber),
				"error", err,
			)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		// Return combined error info for caller awareness
		return fmt.Errorf("failed to notify %d/%d subscribers, first error: %w", len(errs), len(subscribers), errs[0])
	}

	return nil
}

// GetTelegramConfig returns the merged Telegram configuration
// Database values take precedence over environment variables
// If webhook_url is not explicitly configured but apiBaseURL is available,
// it will auto-generate the webhook URL as {apiBaseURL}/webhooks/telegram
func (p *SettingProvider) GetTelegramConfig(ctx context.Context) sharedConfig.TelegramConfig {
	// Start with environment variable config as defaults
	config := p.telegramConfig

	// Try to get database overrides
	settings, err := p.settingRepo.GetByCategory(ctx, "telegram")
	if err != nil {
		p.logger.Warnw("failed to get telegram settings from database, using env config",
			"error", err,
		)
		// Even on error, try to auto-generate webhook URL if applicable
		if config.WebhookURL == "" && p.apiBaseURL != "" {
			config.WebhookURL = fmt.Sprintf("%s/webhooks/telegram", p.apiBaseURL)
		}
		return config
	}

	// Override with database values if present
	for _, s := range settings {
		switch s.Key() {
		case "bot_token":
			if s.HasValue() {
				config.BotToken = s.GetStringValue()
			}
		case "webhook_url":
			if s.HasValue() {
				config.WebhookURL = s.GetStringValue()
			}
		case "webhook_secret":
			if s.HasValue() {
				config.WebhookSecret = s.GetStringValue()
			}
		}
	}

	// Auto-generate webhook URL if not explicitly configured and apiBaseURL is available
	if config.WebhookURL == "" && p.apiBaseURL != "" {
		config.WebhookURL = fmt.Sprintf("%s/webhooks/telegram", p.apiBaseURL)
	}

	return config
}

// GetString retrieves a string setting value
// Database values take precedence over default
func (p *SettingProvider) GetString(ctx context.Context, category, key, defaultValue string) string {
	s, err := p.settingRepo.GetByKey(ctx, category, key)
	if err != nil || s == nil || !s.HasValue() {
		return defaultValue
	}
	return s.GetStringValue()
}

// GetInt retrieves an int setting value
// Database values take precedence over default
func (p *SettingProvider) GetInt(ctx context.Context, category, key string, defaultValue int) int {
	s, err := p.settingRepo.GetByKey(ctx, category, key)
	if err != nil || s == nil || !s.HasValue() {
		return defaultValue
	}
	val, err := s.GetIntValue()
	if err != nil {
		return defaultValue
	}
	return val
}

// GetBool retrieves a bool setting value
// Database values take precedence over default
func (p *SettingProvider) GetBool(ctx context.Context, category, key string, defaultValue bool) bool {
	s, err := p.settingRepo.GetByKey(ctx, category, key)
	if err != nil || s == nil || !s.HasValue() {
		return defaultValue
	}
	val, err := s.GetBoolValue()
	if err != nil {
		return defaultValue
	}
	return val
}

// IsTelegramEnabled checks if Telegram is enabled
// Checks database first, falls back to env config
func (p *SettingProvider) IsTelegramEnabled(ctx context.Context) bool {
	// Check database setting first
	enabled := p.GetBool(ctx, "telegram", "enabled", true)
	if !enabled {
		return false
	}

	// Even if enabled in DB, check if we have valid config
	config := p.GetTelegramConfig(ctx)
	return config.IsConfigured()
}

// GetWebhookSecret returns the webhook secret for Telegram webhook verification
// Database values take precedence over environment variables
func (p *SettingProvider) GetWebhookSecret(ctx context.Context) string {
	config := p.GetTelegramConfig(ctx)
	return config.WebhookSecret
}

// GetAPIBaseURL returns the API base URL with source tracking
// Priority: Database > Environment > Default
func (p *SettingProvider) GetAPIBaseURL(ctx context.Context) settingDTO.SettingWithSource {
	// 1. Check database
	if s, err := p.settingRepo.GetByKey(ctx, "system", "api_base_url"); err == nil && s != nil && s.HasValue() {
		return settingDTO.SettingWithSource{
			Value:  s.GetStringValue(),
			Source: settingDTO.SourceDatabase,
		}
	}
	// 2. Fall back to environment variable
	if p.apiBaseURL != "" {
		return settingDTO.SettingWithSource{
			Value:  p.apiBaseURL,
			Source: settingDTO.SourceEnvironment,
		}
	}
	// 3. Default
	return settingDTO.SettingWithSource{
		Value:  "http://localhost:8080",
		Source: settingDTO.SourceDefault,
	}
}

// GetSubscriptionBaseURL returns the subscription base URL
// Priority: Database > Environment > APIBaseURL
func (p *SettingProvider) GetSubscriptionBaseURL(ctx context.Context) settingDTO.SettingWithSource {
	if s, err := p.settingRepo.GetByKey(ctx, "system", "subscription_base_url"); err == nil && s != nil && s.HasValue() {
		return settingDTO.SettingWithSource{
			Value:  s.GetStringValue(),
			Source: settingDTO.SourceDatabase,
		}
	}
	if p.subscriptionBaseURL != "" {
		return settingDTO.SettingWithSource{
			Value:  p.subscriptionBaseURL,
			Source: settingDTO.SourceEnvironment,
		}
	}
	// Fall back to API base URL
	return p.GetAPIBaseURL(ctx)
}

// GetFrontendURL returns the frontend callback URL
// Priority: Database > Environment > Default (empty)
func (p *SettingProvider) GetFrontendURL(ctx context.Context) settingDTO.SettingWithSource {
	if s, err := p.settingRepo.GetByKey(ctx, "system", "frontend_url"); err == nil && s != nil && s.HasValue() {
		return settingDTO.SettingWithSource{
			Value:  s.GetStringValue(),
			Source: settingDTO.SourceDatabase,
		}
	}
	if p.frontendURL != "" {
		return settingDTO.SettingWithSource{
			Value:  p.frontendURL,
			Source: settingDTO.SourceEnvironment,
		}
	}
	return settingDTO.SettingWithSource{
		Value:  "",
		Source: settingDTO.SourceDefault,
	}
}

// GetTimezone returns the timezone (read-only from environment)
func (p *SettingProvider) GetTimezone(_ context.Context) settingDTO.SettingWithSource {
	return settingDTO.SettingWithSource{
		Value:  p.timezone,
		Source: settingDTO.SourceEnvironment,
	}
}

// GetGoogleOAuthConfig returns the merged Google OAuth configuration
// Database values take precedence over environment variables
func (p *SettingProvider) GetGoogleOAuthConfig(ctx context.Context) sharedConfig.GoogleOAuthConfig {
	config := p.googleOAuthConfig

	settings, err := p.settingRepo.GetByCategory(ctx, "oauth_google")
	if err != nil {
		p.logger.Warnw("failed to get Google OAuth settings from database", "error", err)
		return config
	}

	for _, s := range settings {
		switch s.Key() {
		case "client_id":
			if s.HasValue() {
				config.ClientID = s.GetStringValue()
			}
		case "client_secret":
			if s.HasValue() {
				config.ClientSecret = s.GetStringValue()
			}
		case "redirect_url":
			if s.HasValue() {
				config.RedirectURL = s.GetStringValue()
			}
		}
	}

	return config
}

// GetGitHubOAuthConfig returns the merged GitHub OAuth configuration
// Database values take precedence over environment variables
func (p *SettingProvider) GetGitHubOAuthConfig(ctx context.Context) sharedConfig.GitHubOAuthConfig {
	config := p.githubOAuthConfig

	settings, err := p.settingRepo.GetByCategory(ctx, "oauth_github")
	if err != nil {
		p.logger.Warnw("failed to get GitHub OAuth settings from database", "error", err)
		return config
	}

	for _, s := range settings {
		switch s.Key() {
		case "client_id":
			if s.HasValue() {
				config.ClientID = s.GetStringValue()
			}
		case "client_secret":
			if s.HasValue() {
				config.ClientSecret = s.GetStringValue()
			}
		case "redirect_url":
			if s.HasValue() {
				config.RedirectURL = s.GetStringValue()
			}
		}
	}

	return config
}

// GetEmailConfig returns the merged Email configuration
// Database values take precedence over environment variables
func (p *SettingProvider) GetEmailConfig(ctx context.Context) sharedConfig.EmailConfig {
	config := p.emailConfig

	settings, err := p.settingRepo.GetByCategory(ctx, "email")
	if err != nil {
		p.logger.Warnw("failed to get Email settings from database", "error", err)
		return config
	}

	for _, s := range settings {
		switch s.Key() {
		case "smtp_host":
			if s.HasValue() {
				config.SMTPHost = s.GetStringValue()
			}
		case "smtp_port":
			if s.HasValue() {
				if port, err := s.GetIntValue(); err == nil {
					config.SMTPPort = port
				}
			}
		case "smtp_user":
			if s.HasValue() {
				config.SMTPUser = s.GetStringValue()
			}
		case "smtp_password":
			if s.HasValue() {
				config.SMTPPassword = s.GetStringValue()
			}
		case "from_address":
			if s.HasValue() {
				config.FromAddress = s.GetStringValue()
			}
		case "from_name":
			if s.HasValue() {
				config.FromName = s.GetStringValue()
			}
		}
	}

	return config
}

// IsSystemConfigured checks if the essential system configuration is set
func (p *SettingProvider) IsSystemConfigured(ctx context.Context) bool {
	apiBaseURL := p.GetAPIBaseURL(ctx)
	return apiBaseURL.Value != "" && apiBaseURL.Source != settingDTO.SourceDefault
}

// USDTConfig holds USDT payment configuration from settings
type USDTConfig struct {
	Enabled               bool
	POLReceivingAddresses []string
	TRCReceivingAddresses []string
	PolygonScanAPIKey     string
	TronGridAPIKey        string
	PaymentTTLMinutes     int
	POLConfirmations      int
	TRCConfirmations      int
}

// GetUSDTConfig returns the USDT payment configuration
func (p *SettingProvider) GetUSDTConfig(ctx context.Context) USDTConfig {
	config := USDTConfig{
		Enabled:               false,
		POLReceivingAddresses: []string{},
		TRCReceivingAddresses: []string{},
		PaymentTTLMinutes:     10, // Default 10 minutes per flow diagram
		POLConfirmations:      12,
		TRCConfirmations:      19,
	}

	settings, err := p.settingRepo.GetByCategory(ctx, "usdt")
	if err != nil {
		p.logger.Warnw("failed to get USDT settings from database", "error", err)
		return config
	}

	for _, s := range settings {
		switch s.Key() {
		case "enabled":
			if val, err := s.GetBoolValue(); err == nil {
				config.Enabled = val
			}
		case "pol_receiving_addresses":
			if s.HasValue() {
				if addrs, err := s.GetStringArrayValue(); err == nil {
					config.POLReceivingAddresses = addrs
				}
			}
		case "trc_receiving_addresses":
			if s.HasValue() {
				if addrs, err := s.GetStringArrayValue(); err == nil {
					config.TRCReceivingAddresses = addrs
				}
			}
		case "polygonscan_api_key":
			if s.HasValue() {
				config.PolygonScanAPIKey = s.GetStringValue()
			}
		case "trongrid_api_key":
			if s.HasValue() {
				config.TronGridAPIKey = s.GetStringValue()
			}
		case "payment_ttl_minutes":
			if val, err := s.GetIntValue(); err == nil && val > 0 {
				config.PaymentTTLMinutes = val
			}
		case "pol_confirmations":
			if val, err := s.GetIntValue(); err == nil && val > 0 {
				config.POLConfirmations = val
			}
		case "trc_confirmations":
			if val, err := s.GetIntValue(); err == nil && val > 0 {
				config.TRCConfirmations = val
			}
		}
	}

	return config
}

// IsUSDTEnabled checks if USDT payment is enabled
func (p *SettingProvider) IsUSDTEnabled(ctx context.Context) bool {
	config := p.GetUSDTConfig(ctx)
	return config.Enabled
}
