package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/application/setting/dto"
	"github.com/orris-inc/orris/internal/domain/setting"
	sharedConfig "github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// TelegramConnectionTester defines the interface for testing Telegram connection
type TelegramConnectionTester interface {
	TestConnection(botToken string) (botUsername string, err error)
	GetBotLink() string
}

// GetSettingsUseCase handles retrieval of system settings
type GetSettingsUseCase struct {
	settingRepo    setting.Repository
	telegramConfig sharedConfig.TelegramConfig
	telegramTester TelegramConnectionTester
	logger         logger.Interface
}

// NewGetSettingsUseCase creates a new GetSettingsUseCase
func NewGetSettingsUseCase(
	settingRepo setting.Repository,
	telegramConfig sharedConfig.TelegramConfig,
	telegramTester TelegramConnectionTester,
	logger logger.Interface,
) *GetSettingsUseCase {
	return &GetSettingsUseCase{
		settingRepo:    settingRepo,
		telegramConfig: telegramConfig,
		telegramTester: telegramTester,
		logger:         logger,
	}
}

// SetTelegramTester sets the telegram tester for connection testing.
// This is used to break circular dependency during initialization.
func (uc *GetSettingsUseCase) SetTelegramTester(tester TelegramConnectionTester) {
	uc.telegramTester = tester
}

// GetByCategory retrieves all settings in a category
func (uc *GetSettingsUseCase) GetByCategory(ctx context.Context, category string) (*dto.CategorySettingsResponse, error) {
	settings, err := uc.settingRepo.GetByCategory(ctx, category)
	if err != nil {
		uc.logger.Errorw("failed to get settings by category",
			"category", category,
			"error", err,
		)
		return nil, err
	}

	response := &dto.CategorySettingsResponse{
		Category: category,
		Settings: make([]dto.SystemSettingResponse, 0, len(settings)),
	}

	for _, s := range settings {
		isSensitive := dto.IsSensitiveKey(s.Key())
		value := uc.parseValue(s)

		// Mask sensitive values
		if isSensitive {
			if strVal, ok := value.(string); ok {
				value = dto.MaskSensitiveValue(strVal)
			}
		}

		response.Settings = append(response.Settings, dto.SystemSettingResponse{
			SID:         s.SID(),
			Category:    s.Category(),
			Key:         s.Key(),
			Value:       value,
			ValueType:   string(s.ValueType()),
			Description: s.Description(),
			IsSensitive: isSensitive,
			UpdatedAt:   s.UpdatedAt(),
		})
	}

	return response, nil
}

// GetTelegramConfig retrieves Telegram configuration (merged from database and env)
func (uc *GetSettingsUseCase) GetTelegramConfig(ctx context.Context) (*dto.TelegramConfigResponse, error) {
	// Start with environment variable config as defaults
	response := &dto.TelegramConfigResponse{
		Enabled:       uc.telegramConfig.IsConfigured(),
		BotToken:      dto.MaskSensitiveValue(uc.telegramConfig.BotToken),
		WebhookURL:    uc.telegramConfig.WebhookURL,
		WebhookSecret: dto.MaskSensitiveValue(uc.telegramConfig.WebhookSecret),
		Mode:          "polling",
	}

	if uc.telegramConfig.WebhookURL != "" {
		response.Mode = "webhook"
	}

	// Get bot link if telegram tester is available
	if uc.telegramTester != nil {
		response.BotLink = uc.telegramTester.GetBotLink()
	}

	// Try to get database overrides
	settings, err := uc.settingRepo.GetByCategory(ctx, "telegram")
	if err != nil {
		// Log but don't fail - return env config
		uc.logger.Warnw("failed to get telegram settings from database, using env config",
			"error", err,
		)
		return response, nil
	}

	// Override with database values if present
	for _, s := range settings {
		switch s.Key() {
		case "enabled":
			if val, err := s.GetBoolValue(); err == nil {
				response.Enabled = val
			}
		case "bot_token":
			if s.HasValue() {
				response.BotToken = dto.MaskSensitiveValue(s.GetStringValue())
			}
		case "webhook_url":
			if s.HasValue() {
				response.WebhookURL = s.GetStringValue()
				response.Mode = "webhook"
			}
		case "webhook_secret":
			if s.HasValue() {
				response.WebhookSecret = dto.MaskSensitiveValue(s.GetStringValue())
			}
		}
	}

	// Update mode based on final webhook URL
	if response.WebhookURL == "" {
		response.Mode = "polling"
	}

	return response, nil
}

// TestTelegramConnection tests the Telegram bot connection
// If testToken is provided, it will be used instead of the saved token
func (uc *GetSettingsUseCase) TestTelegramConnection(ctx context.Context, testToken string) (*dto.TelegramTestResult, error) {
	result := &dto.TelegramTestResult{}

	// Use provided test token if available, otherwise get saved token
	botToken := testToken
	if botToken == "" {
		// Get the current bot token (prefer database, fallback to env)
		botToken = uc.telegramConfig.BotToken

		// Check if database has a different token
		tokenSetting, err := uc.settingRepo.GetByKey(ctx, "telegram", "bot_token")
		if err == nil && tokenSetting != nil && tokenSetting.HasValue() {
			botToken = tokenSetting.GetStringValue()
		}
	}

	if botToken == "" {
		result.Success = false
		result.Error = "bot token is not configured"
		return result, nil
	}

	if uc.telegramTester == nil {
		result.Success = false
		result.Error = "telegram tester is not available"
		return result, nil
	}

	username, err := uc.telegramTester.TestConnection(botToken)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}

	result.Success = true
	result.BotUsername = username
	return result, nil
}

// parseValue parses the setting value based on its type
func (uc *GetSettingsUseCase) parseValue(s *setting.SystemSetting) any {
	switch s.ValueType() {
	case setting.ValueTypeInt:
		if val, err := s.GetIntValue(); err == nil {
			return val
		}
	case setting.ValueTypeBool:
		if val, err := s.GetBoolValue(); err == nil {
			return val
		}
	case setting.ValueTypeJSON:
		var val any
		if err := s.GetJSONValue(&val); err == nil {
			return val
		}
	}
	return s.GetStringValue()
}

// GetSettingByKey retrieves a setting by category and key
func (uc *GetSettingsUseCase) GetSettingByKey(ctx context.Context, category, key string) (*setting.SystemSetting, error) {
	return uc.settingRepo.GetByKey(ctx, category, key)
}
