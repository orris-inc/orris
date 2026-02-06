package telegram

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	telegramApp "github.com/orris-inc/orris/internal/application/telegram"
	"github.com/orris-inc/orris/internal/application/telegram/dto"
	"github.com/orris-inc/orris/internal/infrastructure/telegram/i18n"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// AdminTelegramService defines the interface for admin telegram binding
type AdminTelegramService interface {
	BindFromWebhook(ctx context.Context, verifyCode string, telegramUserID int64, telegramUsername string) (any, error)
	UnbindByTelegramID(ctx context.Context, telegramUserID int64) error
	GetBindingByTelegramID(ctx context.Context, telegramUserID int64) (any, error)
	UpdateAdminBindingLanguage(ctx context.Context, telegramUserID int64, language string) error
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

// WebhookSecretProvider provides webhook secret with hot-reload support
type WebhookSecretProvider interface {
	GetWebhookSecret(ctx context.Context) string
}

// Handler handles telegram-related HTTP requests
type Handler struct {
	service               *telegramApp.ServiceDDD
	adminService          AdminTelegramService    // Optional, for admin binding
	muteService           MuteNotificationService // Optional, for muting notifications
	callbackAnswerer      CallbackAnswerer        // Optional, for answering callback queries
	logger                logger.Interface
	webhookSecret         string                // Initial/fallback webhook secret from config
	webhookSecretProvider WebhookSecretProvider // Optional, for hot-reload from database
}

// NewHandler creates a new telegram handler
func NewHandler(service *telegramApp.ServiceDDD, logger logger.Interface, webhookSecret string) *Handler {
	return &Handler{
		service:       service,
		logger:        logger,
		webhookSecret: webhookSecret,
	}
}

// SetWebhookSecretProvider sets the provider for hot-reloadable webhook secret
func (h *Handler) SetWebhookSecretProvider(provider WebhookSecretProvider) {
	h.webhookSecretProvider = provider
}

// getWebhookSecret returns the current webhook secret (from provider if available, otherwise fallback)
func (h *Handler) getWebhookSecret(ctx context.Context) string {
	if h.webhookSecretProvider != nil {
		if secret := h.webhookSecretProvider.GetWebhookSecret(ctx); secret != "" {
			return secret
		}
	}
	return h.webhookSecret
}

// SetAdminService sets the admin telegram service (optional dependency injection)
func (h *Handler) SetAdminService(adminService AdminTelegramService) {
	h.adminService = adminService
}

// SetMuteService sets the mute notification service (optional dependency injection)
func (h *Handler) SetMuteService(muteService MuteNotificationService) {
	h.muteService = muteService
}

// SetCallbackAnswerer sets the callback answerer (optional dependency injection)
func (h *Handler) SetCallbackAnswerer(answerer CallbackAnswerer) {
	h.callbackAnswerer = answerer
}

// GetBindingStatus returns the current telegram binding status
// GET /telegram/binding
func (h *Handler) GetBindingStatus(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	resp, err := h.service.GetBindingStatus(c.Request.Context(), userID)
	if err != nil {
		h.logger.Errorw("failed to get binding status", "user_id", userID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", resp)
}

// Unbind removes the telegram binding
// DELETE /telegram/binding
func (h *Handler) Unbind(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := h.service.Unbind(c.Request.Context(), userID); err != nil {
		h.logger.Errorw("failed to unbind telegram", "user_id", userID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Telegram unbound successfully", nil)
}

// UpdatePreferences updates notification preferences
// PATCH /telegram/preferences
func (h *Handler) UpdatePreferences(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req dto.UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update preferences", "user_id", userID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	resp, err := h.service.UpdatePreferences(c.Request.Context(), userID, req)
	if err != nil {
		h.logger.Errorw("failed to update preferences", "user_id", userID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Preferences updated successfully", resp)
}

// HandleWebhook handles Telegram webhook updates
// POST /webhooks/telegram
func (h *Handler) HandleWebhook(c *gin.Context) {
	// Get webhook secret with hot-reload support (database first, then fallback to config)
	webhookSecret := h.getWebhookSecret(c.Request.Context())

	// Verify webhook secret - REQUIRED for security
	// If webhook secret is not configured, reject all requests to prevent unauthorized access
	if webhookSecret == "" {
		h.logger.Errorw("webhook secret not configured, rejecting request for security")
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "webhook not configured")
		return
	}

	secretHeader := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(secretHeader), []byte(webhookSecret)) != 1 {
		h.logger.Warnw("webhook secret verification failed",
			"expected_secret_configured", true,
			"received_secret_empty", secretHeader == "",
		)
		utils.ErrorResponse(c, http.StatusUnauthorized, "invalid webhook secret")
		return
	}

	var update dto.WebhookUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		h.logger.Errorw("failed to parse webhook update", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Handle callback query from inline keyboard buttons
	if update.CallbackQuery != nil {
		h.handleCallbackQuery(c, update.CallbackQuery)
		return
	}

	if update.Message == nil || update.Message.From == nil {
		// Not a message we care about
		utils.SuccessResponse(c, http.StatusOK, "ignored", nil)
		return
	}

	text := strings.TrimSpace(update.Message.Text)
	telegramUserID := update.Message.From.ID
	username := update.Message.From.Username
	langCode := update.Message.From.LanguageCode
	lang := i18n.DetectLang(langCode)

	// Handle commands
	switch {
	case strings.HasPrefix(text, "/start "):
		// Deep link: /start <payload>
		payload := strings.TrimSpace(strings.TrimPrefix(text, "/start "))
		h.handleStartPayload(c, telegramUserID, username, lang, payload)
	case strings.HasPrefix(text, "/bind "):
		code := strings.TrimSpace(strings.TrimPrefix(text, "/bind "))
		h.handleBindCommand(c, telegramUserID, username, lang, code)
	case text == "/unbind":
		h.handleUnbindCommand(c, telegramUserID, lang)
	case text == "/status":
		h.handleStatusCommand(c, telegramUserID, lang)
	case text == "/start" || text == "/help":
		h.handleHelpCommand(c, telegramUserID, lang)
	case strings.HasPrefix(text, "/adminbind "):
		code := strings.TrimSpace(strings.TrimPrefix(text, "/adminbind "))
		h.handleAdminBindCommand(c, telegramUserID, username, lang, code)
	case text == "/adminunbind":
		h.handleAdminUnbindCommand(c, telegramUserID, lang)
	case text == "/adminstatus":
		h.handleAdminStatusCommand(c, telegramUserID, lang)
	default:
		// Unknown command, show help
		h.handleHelpCommand(c, telegramUserID, lang)
	}
}

func (h *Handler) handleStartPayload(c *gin.Context, telegramUserID int64, username string, lang i18n.Lang, payload string) {
	// Reject overly long payloads to prevent abuse
	const maxPayloadLen = 128
	if len(payload) > maxPayloadLen {
		h.handleHelpCommand(c, telegramUserID, lang)
		return
	}

	switch {
	case strings.HasPrefix(payload, "bind_"):
		code := strings.TrimPrefix(payload, "bind_")
		h.handleBindCommand(c, telegramUserID, username, lang, code)
	case strings.HasPrefix(payload, "adminbind_"):
		code := strings.TrimPrefix(payload, "adminbind_")
		h.handleAdminBindCommand(c, telegramUserID, username, lang, code)
	default:
		h.handleHelpCommand(c, telegramUserID, lang)
	}
}

func (h *Handler) handleBindCommand(c *gin.Context, telegramUserID int64, username string, lang i18n.Lang, code string) {
	if code == "" {
		_ = h.service.SendBotMessage(telegramUserID, i18n.MsgBindMissingCode(lang))
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "missing code"})
		return
	}

	_ = h.service.SendBotChatAction(telegramUserID, "typing")
	resp, err := h.service.BindFromWebhook(c.Request.Context(), telegramUserID, username, code)
	if err != nil {
		h.logger.Errorw("failed to bind telegram from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		_ = h.service.SendBotMessage(telegramUserID, i18n.MsgBindFailed(lang))
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "binding failed"})
		return
	}

	// Update language after successful binding
	if err := h.service.UpdateBindingLanguage(c.Request.Context(), telegramUserID, string(lang)); err != nil {
		h.logger.Debugw("failed to update binding language", "telegram_user_id", telegramUserID, "error", err)
	}

	// Send success message with reply keyboard for easy access to commands
	_ = h.service.SendBotMessageWithKeyboard(telegramUserID, i18n.MsgBindSuccess(lang))
	utils.SuccessResponse(c, http.StatusOK, "success", resp)
}

func (h *Handler) handleUnbindCommand(c *gin.Context, telegramUserID int64, lang i18n.Lang) {
	err := h.service.UnbindByTelegramID(c.Request.Context(), telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to unbind telegram from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		_ = h.service.SendBotMessage(telegramUserID, i18n.MsgUnbindFailed(lang))
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "unbind failed"})
		return
	}

	_ = h.service.SendBotMessage(telegramUserID, i18n.MsgUnbindSuccess(lang))
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func (h *Handler) handleStatusCommand(c *gin.Context, telegramUserID int64, lang i18n.Lang) {
	_ = h.service.SendBotChatAction(telegramUserID, "typing")
	status, err := h.service.GetBindingStatusByTelegramID(c.Request.Context(), telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to get binding status from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		_ = h.service.SendBotMessage(telegramUserID, i18n.MsgStatusError(lang))
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	if !status.IsBound {
		_ = h.service.SendBotMessage(telegramUserID, i18n.MsgStatusNotConnected(lang))
	} else {
		// Update language if bound
		if err := h.service.UpdateBindingLanguage(c.Request.Context(), telegramUserID, string(lang)); err != nil {
			h.logger.Debugw("failed to update binding language", "telegram_user_id", telegramUserID, "error", err)
		}

		msg := i18n.BuildStatusConnectedMessage(
			lang,
			status.Binding.NotifyExpiring,
			status.Binding.ExpiringDays,
			status.Binding.NotifyTraffic,
			status.Binding.TrafficThreshold,
		)
		_ = h.service.SendBotMessage(telegramUserID, msg)
	}
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func (h *Handler) handleHelpCommand(c *gin.Context, telegramUserID int64, lang i18n.Lang) {
	// Send help message with reply keyboard for easy access to commands
	_ = h.service.SendBotMessageWithKeyboard(telegramUserID, i18n.MsgHelpUser(lang))
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

// Admin command handlers

func (h *Handler) handleAdminBindCommand(c *gin.Context, telegramUserID int64, username string, lang i18n.Lang, code string) {
	if h.adminService == nil {
		_ = h.service.SendBotMessage(telegramUserID, i18n.MsgAdminFeatureNotEnabled(lang))
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "admin service not configured"})
		return
	}

	if code == "" {
		_ = h.service.SendBotMessage(telegramUserID, i18n.MsgAdminBindMissingCode(lang))
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "missing code"})
		return
	}

	_, err := h.adminService.BindFromWebhook(c.Request.Context(), code, telegramUserID, username)
	if err != nil {
		h.logger.Errorw("failed to bind admin telegram from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		_ = h.service.SendBotMessage(telegramUserID, i18n.MsgAdminBindFailed(lang))
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "binding failed"})
		return
	}

	// Update language after successful binding
	if err := h.adminService.UpdateAdminBindingLanguage(c.Request.Context(), telegramUserID, string(lang)); err != nil {
		h.logger.Debugw("failed to update admin binding language", "telegram_user_id", telegramUserID, "error", err)
	}

	_ = h.service.SendBotMessage(telegramUserID, i18n.MsgAdminBindSuccess(lang))
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func (h *Handler) handleAdminUnbindCommand(c *gin.Context, telegramUserID int64, lang i18n.Lang) {
	if h.adminService == nil {
		_ = h.service.SendBotMessage(telegramUserID, i18n.MsgAdminFeatureNotEnabledShort(lang))
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "admin service not configured"})
		return
	}

	err := h.adminService.UnbindByTelegramID(c.Request.Context(), telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to unbind admin telegram from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		_ = h.service.SendBotMessage(telegramUserID, i18n.MsgAdminUnbindFailed(lang))
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "unbind failed"})
		return
	}

	_ = h.service.SendBotMessage(telegramUserID, i18n.MsgAdminUnbindSuccess(lang))
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func (h *Handler) handleAdminStatusCommand(c *gin.Context, telegramUserID int64, lang i18n.Lang) {
	if h.adminService == nil {
		_ = h.service.SendBotMessage(telegramUserID, i18n.MsgAdminFeatureNotEnabledShort(lang))
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "admin service not configured"})
		return
	}

	binding, err := h.adminService.GetBindingByTelegramID(c.Request.Context(), telegramUserID)
	if err != nil || binding == nil {
		_ = h.service.SendBotMessage(telegramUserID, i18n.MsgAdminStatusNotBound(lang))
		utils.SuccessResponse(c, http.StatusOK, "success", nil)
		return
	}

	// Update language if bound
	if err := h.adminService.UpdateAdminBindingLanguage(c.Request.Context(), telegramUserID, string(lang)); err != nil {
		h.logger.Debugw("failed to update admin binding language", "telegram_user_id", telegramUserID, "error", err)
	}

	_ = h.service.SendBotMessage(telegramUserID, i18n.MsgAdminStatusBound(lang))
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

// handleCallbackQuery handles callback queries from inline keyboard buttons
func (h *Handler) handleCallbackQuery(c *gin.Context, query *dto.CallbackQuery) {
	if query == nil || query.Data == "" {
		utils.SuccessResponse(c, http.StatusOK, "ignored", nil)
		return
	}

	// Detect language from callback query user
	lang := i18n.ZH
	if query.From != nil && query.From.LanguageCode != "" {
		lang = i18n.DetectLang(query.From.LanguageCode)
	}

	// Parse callback data: format is "action:type:sid"
	// Example: "mute:agent:fa_xxx" or "mute:node:nd_xxx"
	parts := strings.SplitN(query.Data, ":", 3)
	if len(parts) != 3 {
		h.logger.Warnw("invalid callback data format", "data", query.Data)
		h.answerCallback(query.ID, i18n.MsgCallbackInvalidAction(lang), true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	action := parts[0]
	resourceType := parts[1]
	resourceSID := parts[2]

	// Validate SID format (defense-in-depth)
	if _, _, err := id.ParsePrefixedID(resourceSID); err != nil {
		h.logger.Warnw("invalid resource SID in callback", "sid", resourceSID)
		h.answerCallback(query.ID, i18n.MsgCallbackInvalidAction(lang), true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	switch action {
	case "mute":
		h.handleMuteCallback(c, query, lang, resourceType, resourceSID)
	case "unmute":
		h.handleUnmuteCallback(c, query, lang, resourceType, resourceSID)
	default:
		h.logger.Warnw("unknown callback action", "action", action)
		h.answerCallback(query.ID, i18n.MsgCallbackUnknownAction(lang), true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
	}
}

// handleMuteCallback handles the mute notification callback
func (h *Handler) handleMuteCallback(c *gin.Context, query *dto.CallbackQuery, lang i18n.Lang, resourceType, resourceSID string) {
	if h.muteService == nil || h.adminService == nil {
		h.logger.Warnw("mute/admin service not configured")
		h.answerCallback(query.ID, i18n.MsgCallbackFeatureNotEnabled(lang), true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	// Verify the user is a bound admin (security check)
	if query.From == nil {
		h.logger.Warnw("callback query missing from user")
		h.answerCallback(query.ID, i18n.MsgCallbackInvalidRequest(lang), true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	binding, authErr := h.adminService.GetBindingByTelegramID(c.Request.Context(), query.From.ID)
	if authErr != nil || binding == nil {
		h.logger.Warnw("mute callback from non-admin user",
			"telegram_user_id", query.From.ID,
		)
		h.answerCallback(query.ID, i18n.MsgCallbackPermissionDenied(lang), true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	// Execute mute operation
	var err error
	switch resourceType {
	case "agent":
		err = h.muteService.MuteAgentNotification(c.Request.Context(), resourceSID)
	case "node":
		err = h.muteService.MuteNodeNotification(c.Request.Context(), resourceSID)
	default:
		h.logger.Warnw("unknown resource type for mute", "type", resourceType)
		h.answerCallback(query.ID, i18n.MsgCallbackUnknownResourceType(lang), true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	if err != nil {
		h.logger.Errorw("failed to mute notification",
			"resource_type", resourceType,
			"resource_sid", resourceSID,
			"error", err,
		)
		h.answerCallback(query.ID, i18n.MsgCallbackOperationFailed(lang), true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	// Answer callback with success message
	successMsg := i18n.MsgCallbackMuteSuccess(lang) + i18n.ResourceName(lang, resourceType)
	h.answerCallback(query.ID, successMsg, false)

	// Update the button to show unmute option
	if query.Message != nil && query.Message.Chat != nil && h.callbackAnswerer != nil {
		chatID := query.Message.Chat.ID
		messageID := query.Message.MessageID
		unmuteKeyboard := i18n.BuildUnmuteKeyboard(lang, resourceType, resourceSID)
		if editErr := h.callbackAnswerer.EditMessageReplyMarkup(chatID, messageID, unmuteKeyboard); editErr != nil {
			h.logger.Errorw("failed to update message reply markup after mute",
				"chat_id", chatID,
				"message_id", messageID,
				"error", editErr,
			)
		}
	}

	h.logger.Infow("notification muted via telegram callback",
		"resource_type", resourceType,
		"resource_sid", resourceSID,
		"telegram_user_id", query.From.ID,
	)

	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

// handleUnmuteCallback handles the unmute notification callback
func (h *Handler) handleUnmuteCallback(c *gin.Context, query *dto.CallbackQuery, lang i18n.Lang, resourceType, resourceSID string) {
	if h.muteService == nil || h.adminService == nil {
		h.logger.Warnw("mute/admin service not configured")
		h.answerCallback(query.ID, i18n.MsgCallbackFeatureNotEnabled(lang), true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	// Verify the user is a bound admin (security check)
	if query.From == nil {
		h.logger.Warnw("callback query missing from user")
		h.answerCallback(query.ID, i18n.MsgCallbackInvalidRequest(lang), true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	binding, authErr := h.adminService.GetBindingByTelegramID(c.Request.Context(), query.From.ID)
	if authErr != nil || binding == nil {
		h.logger.Warnw("unmute callback from non-admin user",
			"telegram_user_id", query.From.ID,
		)
		h.answerCallback(query.ID, i18n.MsgCallbackPermissionDenied(lang), true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	// Execute unmute operation
	var err error
	switch resourceType {
	case "agent":
		err = h.muteService.UnmuteAgentNotification(c.Request.Context(), resourceSID)
	case "node":
		err = h.muteService.UnmuteNodeNotification(c.Request.Context(), resourceSID)
	default:
		h.logger.Warnw("unknown resource type for unmute", "type", resourceType)
		h.answerCallback(query.ID, i18n.MsgCallbackUnknownResourceType(lang), true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	if err != nil {
		h.logger.Errorw("failed to unmute notification",
			"resource_type", resourceType,
			"resource_sid", resourceSID,
			"error", err,
		)
		h.answerCallback(query.ID, i18n.MsgCallbackOperationFailed(lang), true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	// Answer callback with success message
	successMsg := i18n.MsgCallbackUnmuteSuccess(lang) + i18n.ResourceName(lang, resourceType)
	h.answerCallback(query.ID, successMsg, false)

	// Update the button to show mute option again
	if query.Message != nil && query.Message.Chat != nil && h.callbackAnswerer != nil {
		chatID := query.Message.Chat.ID
		messageID := query.Message.MessageID
		muteKeyboard := i18n.BuildMuteKeyboard(lang, resourceType, resourceSID)
		if editErr := h.callbackAnswerer.EditMessageReplyMarkup(chatID, messageID, muteKeyboard); editErr != nil {
			h.logger.Errorw("failed to update message reply markup after unmute",
				"chat_id", chatID,
				"message_id", messageID,
				"error", editErr,
			)
		}
	}

	h.logger.Infow("notification unmuted via telegram callback",
		"resource_type", resourceType,
		"resource_sid", resourceSID,
		"telegram_user_id", query.From.ID,
	)

	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

// answerCallback sends a response to a callback query
func (h *Handler) answerCallback(callbackQueryID, text string, showAlert bool) {
	if h.callbackAnswerer != nil {
		_ = h.callbackAnswerer.AnswerCallbackQuery(callbackQueryID, text, showAlert)
	}
}

