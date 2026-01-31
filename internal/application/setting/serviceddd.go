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

// getSettingWithSourceBool retrieves a bool setting value with its source
func (s *ServiceDDD) getSettingWithSourceBool(ctx context.Context, category, key string) dto.SettingWithSource {
	existing, err := s.getSettingsUC.GetSettingByKey(ctx, category, key)
	if err == nil && existing != nil && existing.HasValue() {
		val, err := existing.GetBoolValue()
		if err == nil {
			return dto.SettingWithSource{
				Value:  val,
				Source: dto.SourceDatabase,
			}
		}
	}
	return dto.SettingWithSource{
		Value:  false,
		Source: dto.SourceDefault,
	}
}

// getSettingWithSourceInt retrieves an int setting value with its source
func (s *ServiceDDD) getSettingWithSourceInt(ctx context.Context, category, key string, defaultVal int) dto.SettingWithSource {
	existing, err := s.getSettingsUC.GetSettingByKey(ctx, category, key)
	if err == nil && existing != nil && existing.HasValue() {
		val, err := existing.GetIntValue()
		if err == nil {
			return dto.SettingWithSource{
				Value:  val,
				Source: dto.SourceDatabase,
			}
		}
	}
	return dto.SettingWithSource{
		Value:  defaultVal,
		Source: dto.SourceDefault,
	}
}

// upsertSettingBool creates or updates a bool setting
func (s *ServiceDDD) upsertSettingBool(ctx context.Context, category, key string, value bool, updatedBy uint) error {
	existing, err := s.getSettingsUC.GetSettingByKey(ctx, category, key)
	if err != nil || existing == nil {
		newSetting, err := setting.NewSystemSetting(category, key, setting.ValueTypeBool, "")
		if err != nil {
			return err
		}
		if err := newSetting.SetBoolValue(value, updatedBy); err != nil {
			return err
		}
		return s.updateSettingsUC.UpsertSetting(ctx, newSetting)
	}
	if err := existing.SetBoolValue(value, updatedBy); err != nil {
		return err
	}
	return s.updateSettingsUC.UpsertSetting(ctx, existing)
}

// upsertSettingStringArray creates or updates a string array setting (stored as JSON)
func (s *ServiceDDD) upsertSettingStringArray(ctx context.Context, category, key string, value []string, updatedBy uint) error {
	existing, err := s.getSettingsUC.GetSettingByKey(ctx, category, key)
	if err != nil || existing == nil {
		newSetting, err := setting.NewSystemSetting(category, key, setting.ValueTypeJSON, "")
		if err != nil {
			return err
		}
		if err := newSetting.SetJSONValue(value, updatedBy); err != nil {
			return err
		}
		return s.updateSettingsUC.UpsertSetting(ctx, newSetting)
	}
	if err := existing.SetJSONValue(value, updatedBy); err != nil {
		return err
	}
	return s.updateSettingsUC.UpsertSetting(ctx, existing)
}

// getSettingWithSourceStringArray retrieves a string array setting value with its source
func (s *ServiceDDD) getSettingWithSourceStringArray(ctx context.Context, category, key string) dto.SettingWithSource {
	existing, err := s.getSettingsUC.GetSettingByKey(ctx, category, key)
	if err == nil && existing != nil && existing.HasValue() {
		val, err := existing.GetStringArrayValue()
		if err == nil {
			return dto.SettingWithSource{
				Value:  val,
				Source: dto.SourceDatabase,
			}
		}
	}
	return dto.SettingWithSource{
		Value:  []string{},
		Source: dto.SourceDefault,
	}
}
