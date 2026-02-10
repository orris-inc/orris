package telegram

import (
	"context"
	"fmt"
	"sync"

	"github.com/orris-inc/orris/internal/domain/setting"
	sharedConfig "github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// BotServiceManager manages the lifecycle of Telegram Bot services with hot-reload support.
// It implements SettingChangeSubscriber to receive configuration change notifications.
type BotServiceManager struct {
	provider      setting.SettingProvider
	logger        logger.Interface
	updateHandler UpdateHandler
	offsetStore   OffsetStore // optional, for polling offset persistence

	mu             sync.RWMutex
	currentConfig  *sharedConfig.TelegramConfig
	botService     *BotService
	pollingService *PollingService
	isRunning      bool
}

// NewBotServiceManager creates a new BotServiceManager instance.
func NewBotServiceManager(
	provider setting.SettingProvider,
	updateHandler UpdateHandler,
	logger logger.Interface,
) *BotServiceManager {
	return &BotServiceManager{
		provider:      provider,
		updateHandler: updateHandler,
		logger:        logger,
	}
}

// SetOffsetStore sets the polling offset store for persisting offset across restarts.
func (m *BotServiceManager) SetOffsetStore(store OffsetStore) {
	m.offsetStore = store
}

// Start initializes and starts the Telegram bot service based on current configuration.
// It will start in webhook mode if webhook URL is configured, otherwise polling mode.
func (m *BotServiceManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		m.logger.Infow("telegram bot service already running, skipping start")
		return nil
	}

	// Get current configuration from provider
	config := m.provider.GetTelegramConfig(ctx)

	// Check if telegram is enabled
	if !m.provider.IsTelegramEnabled(ctx) {
		m.logger.Infow("telegram bot is disabled, not starting service")
		return nil
	}

	// Check if configuration is valid
	if !config.IsConfigured() {
		m.logger.Infow("telegram bot is not configured, not starting service")
		return nil
	}

	// Store current config
	m.currentConfig = &config

	// Create bot service
	m.botService = NewBotService(config)

	// Determine mode and start appropriate service
	if config.UsePolling() {
		m.logger.Infow("starting telegram bot in polling mode")
		return m.startPollingMode(ctx)
	}

	m.logger.Infow("starting telegram bot in webhook mode",
		"webhook_url", config.WebhookURL,
	)
	return m.startWebhookMode()
}

// startPollingMode starts the bot in polling mode
func (m *BotServiceManager) startPollingMode(ctx context.Context) error {
	if m.updateHandler == nil {
		m.logger.Warnw("no update handler configured, polling will not process updates")
		m.isRunning = true
		return nil
	}

	m.pollingService = NewPollingService(m.botService, m.updateHandler, m.logger, m.offsetStore)
	if err := m.pollingService.Start(ctx); err != nil {
		m.logger.Errorw("failed to start polling service", "error", err)
		return err
	}

	// Set bot commands for auto-completion (includes admin commands in polling mode)
	m.setupBotCommands()

	m.isRunning = true
	m.logger.Infow("telegram bot polling service started successfully")
	return nil
}

// startWebhookMode sets up webhook for the bot
func (m *BotServiceManager) startWebhookMode() error {
	if err := m.botService.SetWebhook(m.currentConfig.WebhookURL); err != nil {
		m.logger.Errorw("failed to set webhook", "error", err)
		return err
	}

	// Set bot commands for auto-completion
	m.setupBotCommands()

	m.isRunning = true
	m.logger.Infow("telegram bot webhook configured successfully",
		"webhook_url", m.currentConfig.WebhookURL,
	)
	return nil
}

// setupBotCommands sets up the bot command menu for auto-completion
func (m *BotServiceManager) setupBotCommands() {
	if m.botService == nil {
		return
	}

	// Use admin commands as default (includes all commands)
	// Regular users will just see commands they don't have access to fail gracefully
	commands := GetAdminCommands()

	if err := m.botService.SetMyCommands(commands); err != nil {
		m.logger.Warnw("failed to set bot commands", "error", err)
		return
	}

	m.logger.Infow("bot commands configured for auto-completion",
		"command_count", len(commands),
	)
}

// Stop gracefully stops all running Telegram bot services.
func (m *BotServiceManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		m.logger.Debugw("telegram bot service not running, nothing to stop")
		return
	}

	m.logger.Infow("stopping telegram bot service")

	// Stop polling service if running
	if m.pollingService != nil {
		m.pollingService.Stop()
		m.pollingService = nil
	}

	// Delete webhook if in webhook mode
	if m.currentConfig != nil && !m.currentConfig.UsePolling() && m.botService != nil {
		if err := m.botService.DeleteWebhook(); err != nil {
			m.logger.Warnw("failed to delete webhook during stop", "error", err)
		}
	}

	m.botService = nil
	m.isRunning = false
	m.logger.Infow("telegram bot service stopped")
}

// Restart stops and restarts the service with updated configuration.
func (m *BotServiceManager) Restart(ctx context.Context) error {
	m.logger.Infow("restarting telegram bot service")

	// Stop needs its own lock, so we unlock here
	m.Stop()

	// Start will acquire its own lock
	return m.Start(ctx)
}

// OnSettingChange handles configuration changes from the SettingProvider.
// It implements the SettingChangeSubscriber interface.
func (m *BotServiceManager) OnSettingChange(ctx context.Context, category string, changes map[string]any) error {
	// Only handle telegram category
	if category != "telegram" {
		return nil
	}

	m.logger.Infow("received telegram configuration change notification",
		"changes", changes,
	)

	// Check if enabled changed to false
	if enabled, ok := changes["enabled"]; ok {
		if enabledBool, isBool := enabled.(bool); isBool && !enabledBool {
			m.logger.Infow("telegram bot disabled via configuration, stopping service")
			m.Stop()
			return nil
		}
	}

	// Check if we need to restart for config changes
	needsRestart := m.shouldRestartForChanges(changes)
	if !needsRestart {
		m.logger.Debugw("configuration changes do not require restart")
		return nil
	}

	m.logger.Infow("configuration changes require service restart",
		"changed_keys", getChangedKeys(changes),
	)

	return m.Restart(ctx)
}

// shouldRestartForChanges determines if the service needs to restart based on changed settings.
func (m *BotServiceManager) shouldRestartForChanges(changes map[string]any) bool {
	// Keys that require a restart when changed
	restartKeys := map[string]bool{
		"bot_token":      true,
		"webhook_url":    true,
		"webhook_secret": true,
		"enabled":        true,
	}

	for key := range changes {
		if restartKeys[key] {
			return true
		}
	}

	return false
}

// GetBotService returns the current BotService instance.
// Returns nil if the service is not running.
func (m *BotServiceManager) GetBotService() *BotService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.botService
}

// GetPollingService returns the current PollingService instance.
// Returns nil if not running in polling mode.
func (m *BotServiceManager) GetPollingService() *PollingService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pollingService
}

// IsRunning returns whether the bot service is currently running.
func (m *BotServiceManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// GetCurrentConfig returns a copy of the current configuration.
// Returns nil if no configuration is loaded.
func (m *BotServiceManager) GetCurrentConfig() *sharedConfig.TelegramConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentConfig == nil {
		return nil
	}

	// Return a copy to prevent external modification
	configCopy := *m.currentConfig
	return &configCopy
}

// GetBotLink returns the Telegram bot link from the running BotService.
// Returns empty string if bot service is not running or bot username is not available.
func (m *BotServiceManager) GetBotLink() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.botService == nil {
		return ""
	}
	return m.botService.GetBotLink()
}

// getChangedKeys extracts the keys from a changes map for logging purposes.
func getChangedKeys(changes map[string]any) []string {
	keys := make([]string, 0, len(changes))
	for k := range changes {
		keys = append(keys, k)
	}
	return keys
}

// TestConnection tests the Telegram bot connection with the given token.
// This method creates a temporary BotService to test the connection without
// affecting the running service. Implements TelegramConnectionTester interface.
func (m *BotServiceManager) TestConnection(botToken string) (botUsername string, err error) {
	if botToken == "" {
		return "", fmt.Errorf("bot token is empty")
	}

	// Create a temporary config with the provided token
	tempConfig := sharedConfig.TelegramConfig{
		BotToken: botToken,
	}

	// Create a temporary BotService to test connection
	tempService := NewBotService(tempConfig)

	// Get bot info to verify connection
	username := tempService.GetBotUsername()
	if username == "" {
		return "", fmt.Errorf("failed to get bot info, token may be invalid")
	}

	return username, nil
}

// DynamicBotService provides a wrapper around BotServiceManager that implements
// the BotService interface with dynamic lookup. This allows services to be
// initialized before the BotServiceManager is started, and supports hot-reload.
type DynamicBotService struct {
	manager *BotServiceManager
	logger  logger.Interface
}

// NewDynamicBotService creates a new DynamicBotService wrapper
func NewDynamicBotService(manager *BotServiceManager, logger logger.Interface) *DynamicBotService {
	return &DynamicBotService{
		manager: manager,
		logger:  logger,
	}
}

// SendMessage sends a plain text message to a chat
func (d *DynamicBotService) SendMessage(chatID int64, text string) error {
	botService := d.manager.GetBotService()
	if botService == nil {
		d.logger.Debugw("telegram message skipped: bot service not available", "chat_id", chatID)
		return nil
	}
	return botService.SendMessage(chatID, text)
}

// SendMessageMarkdown sends a markdown formatted message to a chat
func (d *DynamicBotService) SendMessageMarkdown(chatID int64, text string) error {
	botService := d.manager.GetBotService()
	if botService == nil {
		d.logger.Debugw("telegram markdown message skipped: bot service not available", "chat_id", chatID)
		return nil
	}
	return botService.SendMessageMarkdown(chatID, text)
}

// SendMessageWithKeyboard sends a message with a reply keyboard
func (d *DynamicBotService) SendMessageWithKeyboard(chatID int64, text string, keyboard any) error {
	botService := d.manager.GetBotService()
	if botService == nil {
		d.logger.Debugw("telegram message with keyboard skipped: bot service not available", "chat_id", chatID)
		return nil
	}
	return botService.SendMessageWithKeyboard(chatID, text, keyboard)
}

// SendMessageWithInlineKeyboard sends a message with an inline keyboard
func (d *DynamicBotService) SendMessageWithInlineKeyboard(chatID int64, text string, keyboard any) error {
	botService := d.manager.GetBotService()
	if botService == nil {
		d.logger.Debugw("telegram message with inline keyboard skipped: bot service not available", "chat_id", chatID)
		return nil
	}
	return botService.SendMessageWithInlineKeyboard(chatID, text, keyboard)
}

// GetDefaultReplyKeyboard returns the default reply keyboard with common commands
func (d *DynamicBotService) GetDefaultReplyKeyboard() any {
	botService := d.manager.GetBotService()
	if botService == nil {
		return nil
	}
	return botService.GetDefaultReplyKeyboard()
}

// GetBotLink returns the Telegram bot link (https://t.me/username)
func (d *DynamicBotService) GetBotLink() string {
	return d.manager.GetBotLink()
}

// AnswerCallbackQuery answers a callback query from an inline keyboard
func (d *DynamicBotService) AnswerCallbackQuery(callbackQueryID string, text string, showAlert bool) error {
	botService := d.manager.GetBotService()
	if botService == nil {
		d.logger.Debugw("telegram callback answer skipped: bot service not available")
		return nil
	}
	return botService.AnswerCallbackQuery(callbackQueryID, text, showAlert)
}

// EditMessageWithInlineKeyboard edits a message with an inline keyboard
func (d *DynamicBotService) EditMessageWithInlineKeyboard(chatID int64, messageID int64, text string, keyboard any) error {
	botService := d.manager.GetBotService()
	if botService == nil {
		d.logger.Debugw("telegram message edit skipped: bot service not available", "chat_id", chatID)
		return nil
	}
	return botService.EditMessageWithInlineKeyboard(chatID, messageID, text, keyboard)
}

// EditMessageReplyMarkup edits only the inline keyboard of a message
func (d *DynamicBotService) EditMessageReplyMarkup(chatID int64, messageID int64, keyboard any) error {
	botService := d.manager.GetBotService()
	if botService == nil {
		d.logger.Debugw("telegram message reply markup edit skipped: bot service not available", "chat_id", chatID)
		return nil
	}
	return botService.EditMessageReplyMarkup(chatID, messageID, keyboard)
}

// SendChatAction sends a chat action (e.g., "typing") to a chat
func (d *DynamicBotService) SendChatAction(chatID int64, action string) error {
	botService := d.manager.GetBotService()
	if botService == nil {
		return nil
	}
	return botService.SendChatAction(chatID, action)
}
