package usecases

import (
	"context"
	"fmt"
	"sync"

	"github.com/orris-inc/orris/internal/domain/setting"
	sharedConfig "github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SettingChangeSubscriber defines the interface for setting change subscribers
type SettingChangeSubscriber interface {
	OnSettingChange(ctx context.Context, category string, changes map[string]any) error
}

// SettingProvider provides hot-reloadable configuration with database-first, env-fallback logic
type SettingProvider struct {
	settingRepo    setting.Repository
	telegramConfig sharedConfig.TelegramConfig
	apiBaseURL     string // Server API base URL for auto-generating webhook URLs
	logger         logger.Interface

	subscribers []SettingChangeSubscriber
	mu          sync.RWMutex
}

// NewSettingProvider creates a new SettingProvider
func NewSettingProvider(
	settingRepo setting.Repository,
	telegramConfig sharedConfig.TelegramConfig,
	apiBaseURL string,
	logger logger.Interface,
) *SettingProvider {
	return &SettingProvider{
		settingRepo:    settingRepo,
		telegramConfig: telegramConfig,
		apiBaseURL:     apiBaseURL,
		logger:         logger,
		subscribers:    make([]SettingChangeSubscriber, 0),
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

	var lastErr error
	for _, subscriber := range subscribers {
		if err := subscriber.OnSettingChange(ctx, category, changes); err != nil {
			p.logger.Errorw("subscriber failed to handle setting change",
				"category", category,
				"error", err,
			)
			lastErr = err
		}
	}

	return lastErr
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
