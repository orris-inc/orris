package setting

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/setting/dto"
	"github.com/orris-inc/orris/internal/application/setting/usecases"
	"github.com/orris-inc/orris/internal/domain/setting"
	sharedConfig "github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// TelegramTester combines connection testing and bot link provider
type TelegramTester interface {
	usecases.TelegramConnectionTester
}

// EmailTester provides email connection testing capability
type EmailTester interface {
	SendTestEmail(to string) error
}

// ServiceDDD aggregates all setting-related use cases
type ServiceDDD struct {
	getSettingsUC    *usecases.GetSettingsUseCase
	updateSettingsUC *usecases.UpdateSettingsUseCase
	settingProvider  *usecases.SettingProvider
	emailTester      EmailTester
	logger           logger.Interface
}

// NewServiceDDD creates a new setting service
func NewServiceDDD(
	settingRepo setting.Repository,
	cfg usecases.SettingProviderConfig,
	telegramTester TelegramTester,
	logger logger.Interface,
) *ServiceDDD {
	// Create the setting provider for hot-reload support
	provider := usecases.NewSettingProvider(settingRepo, cfg, logger)

	// Create update use case with provider as notifier
	updateUC := usecases.NewUpdateSettingsUseCase(settingRepo, provider, logger)

	// Create get use case
	getUC := usecases.NewGetSettingsUseCase(settingRepo, cfg.TelegramConfig, telegramTester, logger)

	return &ServiceDDD{
		getSettingsUC:    getUC,
		updateSettingsUC: updateUC,
		settingProvider:  provider,
		logger:           logger,
	}
}

// GetByCategory retrieves all settings in a category
func (s *ServiceDDD) GetByCategory(ctx context.Context, category string) (*dto.CategorySettingsResponse, error) {
	return s.getSettingsUC.GetByCategory(ctx, category)
}

// GetTelegramConfig retrieves Telegram configuration
func (s *ServiceDDD) GetTelegramConfig(ctx context.Context) (*dto.TelegramConfigResponse, error) {
	return s.getSettingsUC.GetTelegramConfig(ctx)
}

// TestTelegramConnection tests the Telegram bot connection
// If testToken is provided, it will be used instead of the saved token
func (s *ServiceDDD) TestTelegramConnection(ctx context.Context, testToken string) (*dto.TelegramTestResult, error) {
	return s.getSettingsUC.TestTelegramConnection(ctx, testToken)
}

// UpdateCategorySettings batch updates settings in a category
func (s *ServiceDDD) UpdateCategorySettings(
	ctx context.Context,
	category string,
	request dto.UpdateCategorySettingsRequest,
	updatedBy uint,
) error {
	return s.updateSettingsUC.UpdateCategorySettings(ctx, category, request, updatedBy)
}

// UpdateTelegramConfig updates Telegram configuration
func (s *ServiceDDD) UpdateTelegramConfig(
	ctx context.Context,
	request dto.UpdateTelegramConfigRequest,
	updatedBy uint,
) error {
	return s.updateSettingsUC.UpdateTelegramConfig(ctx, request, updatedBy)
}

// GetSettingProvider returns the setting provider for hot-reload subscriptions
func (s *ServiceDDD) GetSettingProvider() *usecases.SettingProvider {
	return s.settingProvider
}

// Subscribe registers a subscriber for setting changes
func (s *ServiceDDD) Subscribe(subscriber usecases.SettingChangeSubscriber) {
	s.settingProvider.Subscribe(subscriber)
}

// Unsubscribe removes a subscriber
func (s *ServiceDDD) Unsubscribe(subscriber usecases.SettingChangeSubscriber) {
	s.settingProvider.Unsubscribe(subscriber)
}

// GetTelegramConfigRaw returns the merged Telegram configuration (not DTO)
func (s *ServiceDDD) GetTelegramConfigRaw(ctx context.Context) sharedConfig.TelegramConfig {
	return s.settingProvider.GetTelegramConfig(ctx)
}

// IsTelegramEnabled checks if Telegram is enabled
func (s *ServiceDDD) IsTelegramEnabled(ctx context.Context) bool {
	return s.settingProvider.IsTelegramEnabled(ctx)
}

// SetTelegramTester sets the telegram tester for connection testing.
// This is used to break circular dependency during initialization.
func (s *ServiceDDD) SetTelegramTester(tester TelegramTester) {
	s.getSettingsUC.SetTelegramTester(tester)
}

// SetEmailTester sets the email tester for connection testing.
// This is used to break circular dependency during initialization.
func (s *ServiceDDD) SetEmailTester(tester EmailTester) {
	s.emailTester = tester
}

// GetSystemSettings retrieves system settings with source tracking
func (s *ServiceDDD) GetSystemSettings(ctx context.Context) (*dto.SystemSettingsResponse, error) {
	provider := s.settingProvider

	// api_base_url and timezone are read-only (environment variable only)
	apiBaseURL := provider.GetAPIBaseURL(ctx)
	apiBaseURL.IsReadOnly = true

	timezone := provider.GetTimezone(ctx)
	timezone.IsReadOnly = true

	return &dto.SystemSettingsResponse{
		APIBaseURL:          apiBaseURL,
		SubscriptionBaseURL: provider.GetSubscriptionBaseURL(ctx),
		FrontendURL:         provider.GetFrontendURL(ctx),
		Timezone:            timezone,
	}, nil
}

// UpdateSystemSettings updates system settings
// Note: api_base_url and timezone are read-only and cannot be modified via API
func (s *ServiceDDD) UpdateSystemSettings(ctx context.Context, req dto.UpdateSystemSettingsRequest, updatedBy uint) error {
	changes := make(map[string]any)

	if req.SubscriptionBaseURL != nil {
		if err := s.upsertSetting(ctx, "system", "subscription_base_url", *req.SubscriptionBaseURL, updatedBy); err != nil {
			return err
		}
		changes["subscription_base_url"] = *req.SubscriptionBaseURL
	}
	if req.FrontendURL != nil {
		if err := s.upsertSetting(ctx, "system", "frontend_url", *req.FrontendURL, updatedBy); err != nil {
			return err
		}
		changes["frontend_url"] = *req.FrontendURL
	}

	if len(changes) > 0 {
		// Notify subscribers for hot-reload; log warning if fails but don't return error
		// since database update was successful
		if err := s.settingProvider.NotifyChange(ctx, "system", changes); err != nil {
			s.logger.Warnw("failed to notify system setting changes", "error", err)
		}
	}
	return nil
}

// GetOAuthSettings retrieves OAuth settings
func (s *ServiceDDD) GetOAuthSettings(ctx context.Context) (*dto.OAuthSettingsResponse, error) {
	provider := s.settingProvider

	googleCfg := provider.GetGoogleOAuthConfig(ctx)
	githubCfg := provider.GetGitHubOAuthConfig(ctx)

	return &dto.OAuthSettingsResponse{
		Google: dto.OAuthProviderSettings{
			Enabled:      googleCfg.ClientID != "" && googleCfg.ClientSecret != "",
			ClientID:     s.getSettingWithSource(ctx, "oauth_google", "client_id"),
			ClientSecret: s.getSettingWithSourceMasked(ctx, "oauth_google", "client_secret"),
			RedirectURL:  s.getSettingWithSource(ctx, "oauth_google", "redirect_url"),
		},
		GitHub: dto.OAuthProviderSettings{
			Enabled:      githubCfg.ClientID != "" && githubCfg.ClientSecret != "",
			ClientID:     s.getSettingWithSource(ctx, "oauth_github", "client_id"),
			ClientSecret: s.getSettingWithSourceMasked(ctx, "oauth_github", "client_secret"),
			RedirectURL:  s.getSettingWithSource(ctx, "oauth_github", "redirect_url"),
		},
	}, nil
}

// UpdateOAuthSettings updates OAuth settings
func (s *ServiceDDD) UpdateOAuthSettings(ctx context.Context, req dto.UpdateOAuthSettingsRequest, updatedBy uint) error {
	if req.Google != nil {
		changes := make(map[string]any)
		if req.Google.ClientID != nil {
			if err := s.upsertSetting(ctx, "oauth_google", "client_id", *req.Google.ClientID, updatedBy); err != nil {
				return err
			}
			changes["client_id"] = *req.Google.ClientID
		}
		if req.Google.ClientSecret != nil {
			if err := s.upsertSetting(ctx, "oauth_google", "client_secret", *req.Google.ClientSecret, updatedBy); err != nil {
				return err
			}
			changes["client_secret"] = "[REDACTED]"
		}
		if req.Google.RedirectURL != nil {
			if err := s.upsertSetting(ctx, "oauth_google", "redirect_url", *req.Google.RedirectURL, updatedBy); err != nil {
				return err
			}
			changes["redirect_url"] = *req.Google.RedirectURL
		}
		if len(changes) > 0 {
			if err := s.settingProvider.NotifyChange(ctx, "oauth_google", changes); err != nil {
				s.logger.Warnw("failed to notify oauth_google changes", "error", err)
			}
		}
	}

	if req.GitHub != nil {
		changes := make(map[string]any)
		if req.GitHub.ClientID != nil {
			if err := s.upsertSetting(ctx, "oauth_github", "client_id", *req.GitHub.ClientID, updatedBy); err != nil {
				return err
			}
			changes["client_id"] = *req.GitHub.ClientID
		}
		if req.GitHub.ClientSecret != nil {
			if err := s.upsertSetting(ctx, "oauth_github", "client_secret", *req.GitHub.ClientSecret, updatedBy); err != nil {
				return err
			}
			changes["client_secret"] = "[REDACTED]"
		}
		if req.GitHub.RedirectURL != nil {
			if err := s.upsertSetting(ctx, "oauth_github", "redirect_url", *req.GitHub.RedirectURL, updatedBy); err != nil {
				return err
			}
			changes["redirect_url"] = *req.GitHub.RedirectURL
		}
		if len(changes) > 0 {
			if err := s.settingProvider.NotifyChange(ctx, "oauth_github", changes); err != nil {
				s.logger.Warnw("failed to notify oauth_github changes", "error", err)
			}
		}
	}

	return nil
}

// GetEmailSettings retrieves email settings
func (s *ServiceDDD) GetEmailSettings(ctx context.Context) (*dto.EmailSettingsResponse, error) {
	return &dto.EmailSettingsResponse{
		SMTPHost:     s.getSettingWithSource(ctx, "email", "smtp_host"),
		SMTPPort:     s.getSettingWithSource(ctx, "email", "smtp_port"),
		SMTPUser:     s.getSettingWithSource(ctx, "email", "smtp_user"),
		SMTPPassword: s.getSettingWithSourceMasked(ctx, "email", "smtp_password"),
		FromAddress:  s.getSettingWithSource(ctx, "email", "from_address"),
		FromName:     s.getSettingWithSource(ctx, "email", "from_name"),
	}, nil
}

// UpdateEmailSettings updates email settings
func (s *ServiceDDD) UpdateEmailSettings(ctx context.Context, req dto.UpdateEmailSettingsRequest, updatedBy uint) error {
	changes := make(map[string]any)

	if req.SMTPHost != nil {
		if err := s.upsertSetting(ctx, "email", "smtp_host", *req.SMTPHost, updatedBy); err != nil {
			return err
		}
		changes["smtp_host"] = *req.SMTPHost
	}
	if req.SMTPPort != nil {
		if err := s.upsertSettingInt(ctx, "email", "smtp_port", *req.SMTPPort, updatedBy); err != nil {
			return err
		}
		changes["smtp_port"] = *req.SMTPPort
	}
	if req.SMTPUser != nil {
		if err := s.upsertSetting(ctx, "email", "smtp_user", *req.SMTPUser, updatedBy); err != nil {
			return err
		}
		changes["smtp_user"] = *req.SMTPUser
	}
	if req.SMTPPassword != nil {
		if err := s.upsertSetting(ctx, "email", "smtp_password", *req.SMTPPassword, updatedBy); err != nil {
			return err
		}
		changes["smtp_password"] = "[REDACTED]"
	}
	if req.FromAddress != nil {
		if err := s.upsertSetting(ctx, "email", "from_address", *req.FromAddress, updatedBy); err != nil {
			return err
		}
		changes["from_address"] = *req.FromAddress
	}
	if req.FromName != nil {
		if err := s.upsertSetting(ctx, "email", "from_name", *req.FromName, updatedBy); err != nil {
			return err
		}
		changes["from_name"] = *req.FromName
	}

	if len(changes) > 0 {
		return s.settingProvider.NotifyChange(ctx, "email", changes)
	}
	return nil
}

// TestEmailConnection tests email SMTP connection by sending a test email
func (s *ServiceDDD) TestEmailConnection(_ context.Context, recipientEmail string) (*dto.EmailTestResponse, error) {
	if s.emailTester == nil {
		return &dto.EmailTestResponse{
			Success: false,
			Error:   "Email service not configured",
		}, nil
	}

	err := s.emailTester.SendTestEmail(recipientEmail)
	if err != nil {
		return &dto.EmailTestResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &dto.EmailTestResponse{
		Success: true,
	}, nil
}

// GetSetupStatus checks if the system is configured
func (s *ServiceDDD) GetSetupStatus(ctx context.Context) (*dto.SetupStatusResponse, error) {
	missingSettings := []string{}

	apiBaseURL := s.settingProvider.GetAPIBaseURL(ctx)
	if apiBaseURL.Source == dto.SourceDefault {
		missingSettings = append(missingSettings, "api_base_url")
	}

	isConfigured := len(missingSettings) == 0

	return &dto.SetupStatusResponse{
		IsConfigured:    isConfigured,
		RequiresSetup:   !isConfigured,
		MissingSettings: missingSettings,
	}, nil
}

// upsertSetting creates or updates a string setting
func (s *ServiceDDD) upsertSetting(ctx context.Context, category, key, value string, updatedBy uint) error {
	existing, err := s.getSettingsUC.GetSettingByKey(ctx, category, key)
	if err != nil || existing == nil {
		// Create new setting
		newSetting, err := setting.NewSystemSetting(category, key, setting.ValueTypeString, "")
		if err != nil {
			return err
		}
		if err := newSetting.SetStringValue(value, updatedBy); err != nil {
			return err
		}
		return s.updateSettingsUC.UpsertSetting(ctx, newSetting)
	}
	if err := existing.SetStringValue(value, updatedBy); err != nil {
		return err
	}
	return s.updateSettingsUC.UpsertSetting(ctx, existing)
}

// upsertSettingInt creates or updates an int setting
func (s *ServiceDDD) upsertSettingInt(ctx context.Context, category, key string, value int, updatedBy uint) error {
	existing, err := s.getSettingsUC.GetSettingByKey(ctx, category, key)
	if err != nil || existing == nil {
		newSetting, err := setting.NewSystemSetting(category, key, setting.ValueTypeInt, "")
		if err != nil {
			return err
		}
		if err := newSetting.SetIntValue(value, updatedBy); err != nil {
			return err
		}
		return s.updateSettingsUC.UpsertSetting(ctx, newSetting)
	}
	if err := existing.SetIntValue(value, updatedBy); err != nil {
		return err
	}
	return s.updateSettingsUC.UpsertSetting(ctx, existing)
}

// getSettingWithSource retrieves a setting value with its source
// It checks database first, then falls back to environment variable configuration
func (s *ServiceDDD) getSettingWithSource(ctx context.Context, category, key string) dto.SettingWithSource {
	// 1. Check database
	existing, err := s.getSettingsUC.GetSettingByKey(ctx, category, key)
	if err == nil && existing != nil && existing.HasValue() {
		return dto.SettingWithSource{
			Value:       existing.Value(),
			Source:      dto.SourceDatabase,
			IsSensitive: dto.IsSensitiveKey(key),
		}
	}

	// 2. Check environment variable configuration
	envValue := s.getEnvConfigValue(ctx, category, key)
	if envValue != "" {
		return dto.SettingWithSource{
			Value:       envValue,
			Source:      dto.SourceEnvironment,
			IsSensitive: dto.IsSensitiveKey(key),
		}
	}

	// 3. Default
	return dto.SettingWithSource{
		Value:  "",
		Source: dto.SourceDefault,
	}
}

// getEnvConfigValue retrieves the environment variable configuration value for a setting
func (s *ServiceDDD) getEnvConfigValue(ctx context.Context, category, key string) string {
	switch category {
	case "oauth_google":
		cfg := s.settingProvider.GetGoogleOAuthConfig(ctx)
		switch key {
		case "client_id":
			return cfg.ClientID
		case "client_secret":
			return cfg.ClientSecret
		case "redirect_url":
			// Use GetRedirectURL to auto-generate if not explicitly set
			apiBaseURL := s.settingProvider.GetAPIBaseURL(ctx)
			if baseURL, ok := apiBaseURL.Value.(string); ok {
				return cfg.GetRedirectURL(baseURL)
			}
			return cfg.RedirectURL
		}
	case "oauth_github":
		cfg := s.settingProvider.GetGitHubOAuthConfig(ctx)
		switch key {
		case "client_id":
			return cfg.ClientID
		case "client_secret":
			return cfg.ClientSecret
		case "redirect_url":
			// Use GetRedirectURL to auto-generate if not explicitly set
			apiBaseURL := s.settingProvider.GetAPIBaseURL(ctx)
			if baseURL, ok := apiBaseURL.Value.(string); ok {
				return cfg.GetRedirectURL(baseURL)
			}
			return cfg.RedirectURL
		}
	case "email":
		cfg := s.settingProvider.GetEmailConfig(ctx)
		switch key {
		case "smtp_host":
			return cfg.SMTPHost
		case "smtp_port":
			if cfg.SMTPPort > 0 {
				return fmt.Sprintf("%d", cfg.SMTPPort)
			}
		case "smtp_user":
			return cfg.SMTPUser
		case "smtp_password":
			return cfg.SMTPPassword
		case "from_address":
			return cfg.FromAddress
		case "from_name":
			return cfg.FromName
		}
	}
	return ""
}

// getSettingWithSourceMasked retrieves a setting value with its source and masks sensitive values
func (s *ServiceDDD) getSettingWithSourceMasked(ctx context.Context, category, key string) dto.SettingWithSource {
	result := s.getSettingWithSource(ctx, category, key)
	// Check nil first, then try type assertion
	if result.Value != nil {
		if strVal, ok := result.Value.(string); ok && strVal != "" {
			result.Value = dto.MaskSensitiveValue(strVal)
			result.IsSensitive = true
		}
	}
	return result
}
