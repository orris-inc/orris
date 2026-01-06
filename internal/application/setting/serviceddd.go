package setting

import (
	"context"

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

// ServiceDDD aggregates all setting-related use cases
type ServiceDDD struct {
	getSettingsUC    *usecases.GetSettingsUseCase
	updateSettingsUC *usecases.UpdateSettingsUseCase
	settingProvider  *usecases.SettingProvider
	logger           logger.Interface
}

// NewServiceDDD creates a new setting service
func NewServiceDDD(
	settingRepo setting.Repository,
	telegramConfig sharedConfig.TelegramConfig,
	apiBaseURL string,
	telegramTester TelegramTester,
	logger logger.Interface,
) *ServiceDDD {
	// Create the setting provider for hot-reload support
	provider := usecases.NewSettingProvider(settingRepo, telegramConfig, apiBaseURL, logger)

	// Create update use case with provider as notifier
	updateUC := usecases.NewUpdateSettingsUseCase(settingRepo, provider, logger)

	// Create get use case
	getUC := usecases.NewGetSettingsUseCase(settingRepo, telegramConfig, telegramTester, logger)

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
