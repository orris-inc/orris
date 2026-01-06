package telegram

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/shared/logger"
)

var errMuteServiceNotConfigured = errors.New("mute service not configured")

// UpdateHandler defines the interface for handling Telegram updates
type UpdateHandler interface {
	HandleUpdate(ctx context.Context, update *Update) error
}

// PollingService handles long polling for Telegram updates
type PollingService struct {
	botService   *BotService
	handler      UpdateHandler
	logger       logger.Interface
	pollTimeout  int
	stopChan     chan struct{}
	cancelFunc   context.CancelFunc // Used to cancel ongoing HTTP requests during shutdown
	wg           sync.WaitGroup
	lastUpdateID int64
	isRunning    bool
	runningMu    sync.Mutex
}

// NewPollingService creates a new polling service
func NewPollingService(
	botService *BotService,
	handler UpdateHandler,
	logger logger.Interface,
) *PollingService {
	return &PollingService{
		botService:  botService,
		handler:     handler,
		logger:      logger,
		pollTimeout: 30, // 30 seconds long polling timeout
		stopChan:    make(chan struct{}),
	}
}

// Start begins polling for updates
func (s *PollingService) Start(ctx context.Context) error {
	s.runningMu.Lock()
	if s.isRunning {
		s.runningMu.Unlock()
		return nil
	}
	s.isRunning = true
	// Recreate stopChan for restart capability
	s.stopChan = make(chan struct{})
	// Create a cancellable context for HTTP requests
	pollCtx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.runningMu.Unlock()

	// Delete any existing webhook before starting polling
	if err := s.botService.DeleteWebhook(); err != nil {
		s.logger.Warnw("failed to delete webhook before polling", "error", err)
	}

	s.logger.Infow("starting telegram polling service", "timeout", s.pollTimeout)

	s.wg.Add(1)
	go s.pollLoop(pollCtx)

	return nil
}

// Stop stops the polling service
func (s *PollingService) Stop() {
	s.runningMu.Lock()
	if !s.isRunning {
		s.runningMu.Unlock()
		return
	}
	s.isRunning = false
	// Cancel ongoing HTTP requests first to unblock poll()
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	s.runningMu.Unlock()

	close(s.stopChan)
	s.wg.Wait()
	s.logger.Infow("telegram polling service stopped")
}

func (s *PollingService) pollLoop(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			s.logger.Infow("polling stopped due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Infow("polling stopped by stop signal")
			return
		default:
			s.poll(ctx)
		}
	}
}

func (s *PollingService) poll(ctx context.Context) {
	// Calculate offset: 0 for first poll (to get all pending updates), lastUpdateID+1 for subsequent polls
	offset := int64(0)
	if s.lastUpdateID > 0 {
		offset = s.lastUpdateID + 1
	}
	// Use context-aware GetUpdates for graceful shutdown support
	updates, err := s.botService.GetUpdatesWithContext(ctx, offset, s.pollTimeout)
	if err != nil {
		// Check if the error is due to context cancellation (graceful shutdown)
		if ctx.Err() != nil {
			return
		}
		s.logger.Errorw("failed to get updates", "error", err)
		// Wait a bit before retrying to avoid hammering the API on errors
		// Use select to respond to stop signals during wait
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-time.After(5 * time.Second):
			return
		}
	}

	for _, update := range updates {
		// Update the offset to acknowledge this update
		if update.UpdateID >= s.lastUpdateID {
			s.lastUpdateID = update.UpdateID
		}

		// Process the update
		if err := s.handler.HandleUpdate(ctx, &update); err != nil {
			s.logger.Errorw("failed to handle update",
				"update_id", update.UpdateID,
				"error", err,
			)
		}
	}
}

// TelegramServiceForPolling defines the interface for telegram service operations needed by polling
type TelegramServiceForPolling interface {
	BindFromWebhookForPolling(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error
	UnbindByTelegramID(ctx context.Context, telegramUserID int64) error
	IsBoundByTelegramID(ctx context.Context, telegramUserID int64) (bool, error)
	SendBotMessage(chatID int64, text string) error
	SendBotMessageWithKeyboard(chatID int64, text string) error
	// Admin binding
	AdminBindFromPolling(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error
	// Callback query handling
	IsAdminBound(ctx context.Context, telegramUserID int64) (bool, error)
	MuteAgentNotification(ctx context.Context, agentSID string) error
	MuteNodeNotification(ctx context.Context, nodeSID string) error
	UnmuteAgentNotification(ctx context.Context, agentSID string) error
	UnmuteNodeNotification(ctx context.Context, nodeSID string) error
	AnswerCallbackQuery(callbackQueryID string, text string, showAlert bool) error
	EditMessageWithInlineKeyboard(chatID int64, messageID int64, text string, keyboard any) error
	EditMessageReplyMarkup(chatID int64, messageID int64, keyboard any) error
}

// TelegramBinderService defines the interface for binding operations
type TelegramBinderService interface {
	UnbindByTelegramID(ctx context.Context, telegramUserID int64) error
}

// BotServiceGetter provides access to the current BotService instance
type BotServiceGetter interface {
	GetBotService() *BotService
}

// AdminBinderService defines the interface for admin binding operations
type AdminBinderService interface {
	BindFromWebhook(ctx context.Context, verifyCode string, telegramUserID int64, telegramUsername string) (any, error)
	GetBindingByTelegramID(ctx context.Context, telegramUserID int64) (any, error)
}

// MuteNotificationService defines the interface for muting resource notifications
type MuteNotificationService interface {
	MuteAgentNotification(ctx context.Context, agentSID string) error
	MuteNodeNotification(ctx context.Context, nodeSID string) error
	UnmuteAgentNotification(ctx context.Context, agentSID string) error
	UnmuteNodeNotification(ctx context.Context, nodeSID string) error
}

// CallbackAnswerer defines the interface for answering Telegram callback queries
type CallbackAnswerer interface {
	AnswerCallbackQuery(callbackQueryID string, text string, showAlert bool) error
	EditMessageWithInlineKeyboard(chatID int64, messageID int64, text string, keyboard any) error
	EditMessageReplyMarkup(chatID int64, messageID int64, keyboard any) error
}

// ServiceAdapter wraps the telegram ServiceDDD to implement TelegramServiceForPolling
type ServiceAdapter struct {
	binder           TelegramBinderService
	adminBinder      AdminBinderService
	botServiceGetter BotServiceGetter
	muteService      MuteNotificationService
	callbackAnswerer CallbackAnswerer
	bindFunc         func(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error
	getBindingStatus func(ctx context.Context, telegramUserID int64) (bool, error)
}

// NewServiceAdapter creates a new service adapter from telegram ServiceDDD
func NewServiceAdapter(service interface {
	UnbindByTelegramID(ctx context.Context, telegramUserID int64) error
},
	bindFunc func(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error,
	getBindingStatus func(ctx context.Context, telegramUserID int64) (bool, error),
) *ServiceAdapter {
	return &ServiceAdapter{
		binder:           service,
		bindFunc:         bindFunc,
		getBindingStatus: getBindingStatus,
	}
}

// SetBotServiceGetter sets the bot service getter (used to break circular dependency)
func (a *ServiceAdapter) SetBotServiceGetter(getter BotServiceGetter) {
	a.botServiceGetter = getter
}

// BindFromWebhookForPolling implements TelegramServiceForPolling
func (a *ServiceAdapter) BindFromWebhookForPolling(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error {
	return a.bindFunc(ctx, telegramUserID, telegramUsername, verifyCode)
}

// UnbindByTelegramID implements TelegramServiceForPolling
func (a *ServiceAdapter) UnbindByTelegramID(ctx context.Context, telegramUserID int64) error {
	return a.binder.UnbindByTelegramID(ctx, telegramUserID)
}

// IsBoundByTelegramID implements TelegramServiceForPolling
func (a *ServiceAdapter) IsBoundByTelegramID(ctx context.Context, telegramUserID int64) (bool, error) {
	return a.getBindingStatus(ctx, telegramUserID)
}

// SendBotMessage implements TelegramServiceForPolling
func (a *ServiceAdapter) SendBotMessage(chatID int64, text string) error {
	botService := a.botServiceGetter.GetBotService()
	if botService == nil {
		return nil
	}
	return botService.SendMessage(chatID, text)
}

// SendBotMessageWithKeyboard implements TelegramServiceForPolling
func (a *ServiceAdapter) SendBotMessageWithKeyboard(chatID int64, text string) error {
	botService := a.botServiceGetter.GetBotService()
	if botService == nil {
		return nil
	}
	keyboard := botService.GetDefaultReplyKeyboard()
	return botService.SendMessageWithKeyboard(chatID, text, keyboard)
}

// SetAdminBinder sets the admin binder service (used to break circular dependency)
func (a *ServiceAdapter) SetAdminBinder(binder AdminBinderService) {
	a.adminBinder = binder
}

// AdminBindFromPolling implements TelegramServiceForPolling for admin binding
func (a *ServiceAdapter) AdminBindFromPolling(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error {
	if a.adminBinder == nil {
		return nil
	}
	_, err := a.adminBinder.BindFromWebhook(ctx, verifyCode, telegramUserID, telegramUsername)
	return err
}

// SetMuteService sets the mute notification service (used to break circular dependency)
func (a *ServiceAdapter) SetMuteService(muteService MuteNotificationService) {
	a.muteService = muteService
}

// SetCallbackAnswerer sets the callback answerer (used to break circular dependency)
func (a *ServiceAdapter) SetCallbackAnswerer(answerer CallbackAnswerer) {
	a.callbackAnswerer = answerer
}

// IsAdminBound implements TelegramServiceForPolling to check if a telegram user is a bound admin
func (a *ServiceAdapter) IsAdminBound(ctx context.Context, telegramUserID int64) (bool, error) {
	if a.adminBinder == nil {
		return false, nil
	}
	binding, err := a.adminBinder.GetBindingByTelegramID(ctx, telegramUserID)
	if err != nil {
		return false, nil // Treat error as not bound
	}
	return binding != nil, nil
}

// MuteAgentNotification implements TelegramServiceForPolling
func (a *ServiceAdapter) MuteAgentNotification(ctx context.Context, agentSID string) error {
	if a.muteService == nil {
		return errMuteServiceNotConfigured
	}
	return a.muteService.MuteAgentNotification(ctx, agentSID)
}

// MuteNodeNotification implements TelegramServiceForPolling
func (a *ServiceAdapter) MuteNodeNotification(ctx context.Context, nodeSID string) error {
	if a.muteService == nil {
		return errMuteServiceNotConfigured
	}
	return a.muteService.MuteNodeNotification(ctx, nodeSID)
}

// UnmuteAgentNotification implements TelegramServiceForPolling
func (a *ServiceAdapter) UnmuteAgentNotification(ctx context.Context, agentSID string) error {
	if a.muteService == nil {
		return errMuteServiceNotConfigured
	}
	return a.muteService.UnmuteAgentNotification(ctx, agentSID)
}

// UnmuteNodeNotification implements TelegramServiceForPolling
func (a *ServiceAdapter) UnmuteNodeNotification(ctx context.Context, nodeSID string) error {
	if a.muteService == nil {
		return errMuteServiceNotConfigured
	}
	return a.muteService.UnmuteNodeNotification(ctx, nodeSID)
}

// AnswerCallbackQuery implements TelegramServiceForPolling
func (a *ServiceAdapter) AnswerCallbackQuery(callbackQueryID string, text string, showAlert bool) error {
	if a.callbackAnswerer == nil {
		return nil
	}
	return a.callbackAnswerer.AnswerCallbackQuery(callbackQueryID, text, showAlert)
}

// EditMessageWithInlineKeyboard implements TelegramServiceForPolling
func (a *ServiceAdapter) EditMessageWithInlineKeyboard(chatID int64, messageID int64, text string, keyboard any) error {
	if a.callbackAnswerer == nil {
		return nil
	}
	return a.callbackAnswerer.EditMessageWithInlineKeyboard(chatID, messageID, text, keyboard)
}

// EditMessageReplyMarkup implements TelegramServiceForPolling
func (a *ServiceAdapter) EditMessageReplyMarkup(chatID int64, messageID int64, keyboard any) error {
	if a.callbackAnswerer == nil {
		return nil
	}
	return a.callbackAnswerer.EditMessageReplyMarkup(chatID, messageID, keyboard)
}

// PollingUpdateHandler implements UpdateHandler for the telegram service
type PollingUpdateHandler struct {
	service TelegramServiceForPolling
	logger  logger.Interface
}

// NewPollingUpdateHandler creates a new polling update handler
func NewPollingUpdateHandler(
	service TelegramServiceForPolling,
	logger logger.Interface,
) *PollingUpdateHandler {
	return &PollingUpdateHandler{
		service: service,
		logger:  logger,
	}
}

// HandleUpdate processes a single Telegram update
func (h *PollingUpdateHandler) HandleUpdate(ctx context.Context, update *Update) error {
	// Handle callback query from inline keyboard buttons
	if update.CallbackQuery != nil {
		return h.handleCallbackQuery(ctx, update.CallbackQuery)
	}

	if update.Message == nil || update.Message.From == nil {
		return nil
	}

	text := strings.TrimSpace(update.Message.Text)
	telegramUserID := update.Message.From.ID
	username := update.Message.From.Username

	switch {
	case strings.HasPrefix(text, "/bind "):
		code := strings.TrimSpace(strings.TrimPrefix(text, "/bind "))
		return h.handleBindCommand(ctx, telegramUserID, username, code)
	case text == "/unbind":
		return h.handleUnbindCommand(ctx, telegramUserID)
	case text == "/status":
		return h.handleStatusCommand(ctx, telegramUserID)
	case strings.HasPrefix(text, "/adminbind "):
		code := strings.TrimSpace(strings.TrimPrefix(text, "/adminbind "))
		return h.handleAdminBindCommand(ctx, telegramUserID, username, code)
	case text == "/start" || text == "/help":
		return h.handleHelpCommand(telegramUserID)
	default:
		return h.handleHelpCommand(telegramUserID)
	}
}

func (h *PollingUpdateHandler) handleBindCommand(ctx context.Context, telegramUserID int64, username, code string) error {
	if code == "" {
		return h.service.SendBotMessage(telegramUserID, MsgBindMissingCode)
	}

	err := h.service.BindFromWebhookForPolling(ctx, telegramUserID, username, code)
	if err != nil {
		h.logger.Errorw("failed to bind telegram from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, MsgBindFailed)
	}

	return h.service.SendBotMessageWithKeyboard(telegramUserID, MsgBindSuccess)
}

func (h *PollingUpdateHandler) handleUnbindCommand(ctx context.Context, telegramUserID int64) error {
	err := h.service.UnbindByTelegramID(ctx, telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to unbind telegram from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, MsgUnbindFailed)
	}

	return h.service.SendBotMessage(telegramUserID, MsgUnbindSuccess)
}

func (h *PollingUpdateHandler) handleStatusCommand(ctx context.Context, telegramUserID int64) error {
	isBound, err := h.service.IsBoundByTelegramID(ctx, telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to get binding status from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, MsgStatusError)
	}

	if !isBound {
		return h.service.SendBotMessage(telegramUserID, MsgStatusNotConnected)
	}

	// For bound status, just send a generic message since we don't have detailed info in polling mode
	return h.service.SendBotMessage(telegramUserID, MsgStatusConnectedSimple)
}

func (h *PollingUpdateHandler) handleHelpCommand(telegramUserID int64) error {
	return h.service.SendBotMessageWithKeyboard(telegramUserID, MsgHelpFull)
}

func (h *PollingUpdateHandler) handleAdminBindCommand(ctx context.Context, telegramUserID int64, username, code string) error {
	if code == "" {
		return h.service.SendBotMessage(telegramUserID, MsgAdminBindMissingCodePolling)
	}

	err := h.service.AdminBindFromPolling(ctx, telegramUserID, username, code)
	if err != nil {
		h.logger.Errorw("failed to bind admin telegram from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, MsgAdminBindFailedPolling)
	}

	return h.service.SendBotMessageWithKeyboard(telegramUserID, MsgAdminBindSuccessPolling)
}

// handleCallbackQuery handles callback queries from inline keyboard buttons
func (h *PollingUpdateHandler) handleCallbackQuery(ctx context.Context, query *CallbackQuery) error {
	if query == nil || query.Data == "" {
		return nil
	}

	// Parse callback data: format is "action:type:sid"
	// Example: "mute:agent:fa_xxx" or "mute:node:nd_xxx"
	parts := strings.SplitN(query.Data, ":", 3)
	if len(parts) != 3 {
		h.logger.Warnw("invalid callback data format", "data", query.Data)
		_ = h.service.AnswerCallbackQuery(query.ID, MsgCallbackInvalidAction, true)
		return nil
	}

	action := parts[0]
	resourceType := parts[1]
	resourceSID := parts[2]

	switch action {
	case "mute":
		return h.handleMuteCallback(ctx, query, resourceType, resourceSID)
	case "unmute":
		return h.handleUnmuteCallback(ctx, query, resourceType, resourceSID)
	default:
		h.logger.Warnw("unknown callback action", "action", action)
		_ = h.service.AnswerCallbackQuery(query.ID, MsgCallbackUnknownAction, true)
		return nil
	}
}

// handleMuteCallback handles the mute notification callback
func (h *PollingUpdateHandler) handleMuteCallback(ctx context.Context, query *CallbackQuery, resourceType, resourceSID string) error {
	// Verify the user is a bound admin (security check)
	if query.From == nil {
		h.logger.Warnw("callback query missing from user")
		_ = h.service.AnswerCallbackQuery(query.ID, MsgCallbackInvalidRequest, true)
		return nil
	}

	// Check if the telegram user is a bound admin
	isAdmin, err := h.service.IsAdminBound(ctx, query.From.ID)
	if err != nil || !isAdmin {
		h.logger.Warnw("mute callback from non-admin user",
			"telegram_user_id", query.From.ID,
		)
		_ = h.service.AnswerCallbackQuery(query.ID, MsgCallbackPermissionDenied, true)
		return nil
	}

	var muteErr error
	var resourceName string

	switch resourceType {
	case "agent":
		muteErr = h.service.MuteAgentNotification(ctx, resourceSID)
		resourceName = "转发代理 / Forward Agent"
	case "node":
		muteErr = h.service.MuteNodeNotification(ctx, resourceSID)
		resourceName = "Node Agent"
	default:
		h.logger.Warnw("unknown resource type for mute", "type", resourceType)
		_ = h.service.AnswerCallbackQuery(query.ID, MsgCallbackUnknownResourceType, true)
		return nil
	}

	if muteErr != nil {
		h.logger.Errorw("failed to mute notification",
			"resource_type", resourceType,
			"resource_sid", resourceSID,
			"error", muteErr,
		)
		_ = h.service.AnswerCallbackQuery(query.ID, MsgCallbackOperationFailed, true)
		return nil
	}

	// Answer callback with success message
	_ = h.service.AnswerCallbackQuery(query.ID, MsgCallbackMuteSuccess+resourceName, false)

	// Update the button to show unmute option
	if query.Message != nil && query.Message.Chat != nil {
		chatID := query.Message.Chat.ID
		messageID := query.Message.MessageID
		// Only update the button, don't modify message text
		unmuteKeyboard := buildUnmuteKeyboard(resourceType, resourceSID)
		if editErr := h.service.EditMessageReplyMarkup(chatID, messageID, unmuteKeyboard); editErr != nil {
			h.logger.Errorw("failed to update message reply markup after mute",
				"chat_id", chatID,
				"message_id", messageID,
				"error", editErr,
			)
		}
	}

	h.logger.Infow("notification muted via telegram callback (polling)",
		"resource_type", resourceType,
		"resource_sid", resourceSID,
		"telegram_user_id", query.From.ID,
	)

	return nil
}

// handleUnmuteCallback handles the unmute notification callback
func (h *PollingUpdateHandler) handleUnmuteCallback(ctx context.Context, query *CallbackQuery, resourceType, resourceSID string) error {
	// Verify the user is a bound admin (security check)
	if query.From == nil {
		h.logger.Warnw("callback query missing from user")
		_ = h.service.AnswerCallbackQuery(query.ID, MsgCallbackInvalidRequest, true)
		return nil
	}

	// Check if the telegram user is a bound admin
	isAdmin, err := h.service.IsAdminBound(ctx, query.From.ID)
	if err != nil || !isAdmin {
		h.logger.Warnw("unmute callback from non-admin user",
			"telegram_user_id", query.From.ID,
		)
		_ = h.service.AnswerCallbackQuery(query.ID, MsgCallbackPermissionDenied, true)
		return nil
	}

	var unmuteErr error
	var resourceName string

	switch resourceType {
	case "agent":
		unmuteErr = h.service.UnmuteAgentNotification(ctx, resourceSID)
		resourceName = "转发代理 / Forward Agent"
	case "node":
		unmuteErr = h.service.UnmuteNodeNotification(ctx, resourceSID)
		resourceName = "Node Agent"
	default:
		h.logger.Warnw("unknown resource type for unmute", "type", resourceType)
		_ = h.service.AnswerCallbackQuery(query.ID, MsgCallbackUnknownResourceType, true)
		return nil
	}

	if unmuteErr != nil {
		h.logger.Errorw("failed to unmute notification",
			"resource_type", resourceType,
			"resource_sid", resourceSID,
			"error", unmuteErr,
		)
		_ = h.service.AnswerCallbackQuery(query.ID, MsgCallbackOperationFailed, true)
		return nil
	}

	// Answer callback with success message
	_ = h.service.AnswerCallbackQuery(query.ID, MsgCallbackUnmuteSuccess+resourceName, false)

	// Update the button to show mute option again
	if query.Message != nil && query.Message.Chat != nil {
		chatID := query.Message.Chat.ID
		messageID := query.Message.MessageID
		// Only update the button, don't modify message text
		muteKeyboard := buildMuteKeyboard(resourceType, resourceSID)
		if editErr := h.service.EditMessageReplyMarkup(chatID, messageID, muteKeyboard); editErr != nil {
			h.logger.Errorw("failed to update message reply markup after unmute",
				"chat_id", chatID,
				"message_id", messageID,
				"error", editErr,
			)
		}
	}

	h.logger.Infow("notification unmuted via telegram callback (polling)",
		"resource_type", resourceType,
		"resource_sid", resourceSID,
		"telegram_user_id", query.From.ID,
	)

	return nil
}

// buildMuteKeyboard builds an inline keyboard with mute button
func buildMuteKeyboard(resourceType, resourceSID string) map[string]any {
	return map[string]any{
		"inline_keyboard": [][]map[string]string{
			{
				{
					"text":          "🔕 静默此通知 / Mute",
					"callback_data": "mute:" + resourceType + ":" + resourceSID,
				},
			},
		},
	}
}

// buildUnmuteKeyboard builds an inline keyboard with unmute button
func buildUnmuteKeyboard(resourceType, resourceSID string) map[string]any {
	return map[string]any{
		"inline_keyboard": [][]map[string]string{
			{
				{
					"text":          "🔔 解除静默 / Unmute",
					"callback_data": "unmute:" + resourceType + ":" + resourceSID,
				},
			},
		},
	}
}
