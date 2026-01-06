package telegram

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	telegramApp "github.com/orris-inc/orris/internal/application/telegram"
	"github.com/orris-inc/orris/internal/application/telegram/dto"
	telegramInfra "github.com/orris-inc/orris/internal/infrastructure/telegram"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// AdminTelegramService defines the interface for admin telegram binding
type AdminTelegramService interface {
	BindFromWebhook(ctx context.Context, verifyCode string, telegramUserID int64, telegramUsername string) (any, error)
	UnbindByTelegramID(ctx context.Context, telegramUserID int64) error
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
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Internal error"))
		return
	}

	resp, err := h.service.GetBindingStatus(c.Request.Context(), uid)
	if err != nil {
		h.logger.Errorw("failed to get binding status", "user_id", uid, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", resp)
}

// Unbind removes the telegram binding
// DELETE /telegram/binding
func (h *Handler) Unbind(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Internal error"))
		return
	}

	if err := h.service.Unbind(c.Request.Context(), uid); err != nil {
		h.logger.Errorw("failed to unbind telegram", "user_id", uid, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Telegram unbound successfully", nil)
}

// UpdatePreferences updates notification preferences
// PATCH /telegram/preferences
func (h *Handler) UpdatePreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Internal error"))
		return
	}

	var req dto.UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update preferences", "user_id", uid, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	resp, err := h.service.UpdatePreferences(c.Request.Context(), uid, req)
	if err != nil {
		h.logger.Errorw("failed to update preferences", "user_id", uid, "error", err)
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

	// Handle commands
	switch {
	case strings.HasPrefix(text, "/bind "):
		code := strings.TrimSpace(strings.TrimPrefix(text, "/bind "))
		h.handleBindCommand(c, telegramUserID, username, code)
	case text == "/unbind":
		h.handleUnbindCommand(c, telegramUserID)
	case text == "/status":
		h.handleStatusCommand(c, telegramUserID)
	case text == "/start" || text == "/help":
		h.handleHelpCommand(c, telegramUserID)
	case strings.HasPrefix(text, "/adminbind "):
		code := strings.TrimSpace(strings.TrimPrefix(text, "/adminbind "))
		h.handleAdminBindCommand(c, telegramUserID, username, code)
	case text == "/adminunbind":
		h.handleAdminUnbindCommand(c, telegramUserID)
	case text == "/adminstatus":
		h.handleAdminStatusCommand(c, telegramUserID)
	default:
		// Unknown command, show help
		h.handleHelpCommand(c, telegramUserID)
	}
}

func (h *Handler) handleBindCommand(c *gin.Context, telegramUserID int64, username, code string) {
	if code == "" {
		_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgBindMissingCode)
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "missing code"})
		return
	}

	resp, err := h.service.BindFromWebhook(c.Request.Context(), telegramUserID, username, code)
	if err != nil {
		h.logger.Errorw("failed to bind telegram from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgBindFailed)
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "binding failed"})
		return
	}

	// Send success message with reply keyboard for easy access to commands
	_ = h.service.SendBotMessageWithKeyboard(telegramUserID, telegramInfra.MsgBindSuccess)
	utils.SuccessResponse(c, http.StatusOK, "success", resp)
}

func (h *Handler) handleUnbindCommand(c *gin.Context, telegramUserID int64) {
	err := h.service.UnbindByTelegramID(c.Request.Context(), telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to unbind telegram from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgUnbindFailed)
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "unbind failed"})
		return
	}

	_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgUnbindSuccess)
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func (h *Handler) handleStatusCommand(c *gin.Context, telegramUserID int64) {
	status, err := h.service.GetBindingStatusByTelegramID(c.Request.Context(), telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to get binding status from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgStatusError)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	if !status.IsBound {
		_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgStatusNotConnected)
	} else {
		msg := telegramInfra.BuildStatusConnectedMessage(
			status.Binding.NotifyExpiring,
			status.Binding.ExpiringDays,
			status.Binding.NotifyTraffic,
			status.Binding.TrafficThreshold,
		)
		_ = h.service.SendBotMessage(telegramUserID, msg)
	}
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func (h *Handler) handleHelpCommand(c *gin.Context, telegramUserID int64) {
	// Send help message with reply keyboard for easy access to commands
	_ = h.service.SendBotMessageWithKeyboard(telegramUserID, telegramInfra.MsgHelpUser)
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

// Admin command handlers

func (h *Handler) handleAdminBindCommand(c *gin.Context, telegramUserID int64, username, code string) {
	if h.adminService == nil {
		_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgAdminFeatureNotEnabled)
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "admin service not configured"})
		return
	}

	if code == "" {
		_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgAdminBindMissingCode)
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "missing code"})
		return
	}

	_, err := h.adminService.BindFromWebhook(c.Request.Context(), code, telegramUserID, username)
	if err != nil {
		h.logger.Errorw("failed to bind admin telegram from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgAdminBindFailed)
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "binding failed"})
		return
	}

	_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgAdminBindSuccess)
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func (h *Handler) handleAdminUnbindCommand(c *gin.Context, telegramUserID int64) {
	if h.adminService == nil {
		_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgAdminFeatureNotEnabledShort)
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "admin service not configured"})
		return
	}

	err := h.adminService.UnbindByTelegramID(c.Request.Context(), telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to unbind admin telegram from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgAdminUnbindFailed)
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "unbind failed"})
		return
	}

	_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgAdminUnbindSuccess)
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func (h *Handler) handleAdminStatusCommand(c *gin.Context, telegramUserID int64) {
	if h.adminService == nil {
		_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgAdminFeatureNotEnabledShort)
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "admin service not configured"})
		return
	}

	binding, err := h.adminService.GetBindingByTelegramID(c.Request.Context(), telegramUserID)
	if err != nil || binding == nil {
		_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgAdminStatusNotBound)
		utils.SuccessResponse(c, http.StatusOK, "success", nil)
		return
	}

	_ = h.service.SendBotMessage(telegramUserID, telegramInfra.MsgAdminStatusBound)
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

// handleCallbackQuery handles callback queries from inline keyboard buttons
func (h *Handler) handleCallbackQuery(c *gin.Context, query *dto.CallbackQuery) {
	if query == nil || query.Data == "" {
		utils.SuccessResponse(c, http.StatusOK, "ignored", nil)
		return
	}

	// Parse callback data: format is "action:type:sid"
	// Example: "mute:agent:fa_xxx" or "mute:node:nd_xxx"
	parts := strings.SplitN(query.Data, ":", 3)
	if len(parts) != 3 {
		h.logger.Warnw("invalid callback data format", "data", query.Data)
		h.answerCallback(query.ID, "❌ 无效操作 / Invalid action", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	action := parts[0]
	resourceType := parts[1]
	resourceSID := parts[2]

	switch action {
	case "mute":
		h.handleMuteCallback(c, query, resourceType, resourceSID)
	case "unmute":
		h.handleUnmuteCallback(c, query, resourceType, resourceSID)
	default:
		h.logger.Warnw("unknown callback action", "action", action)
		h.answerCallback(query.ID, "❌ 未知操作 / Unknown action", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
	}
}

// handleMuteCallback handles the mute notification callback
func (h *Handler) handleMuteCallback(c *gin.Context, query *dto.CallbackQuery, resourceType, resourceSID string) {
	if h.muteService == nil {
		h.logger.Warnw("mute service not configured")
		h.answerCallback(query.ID, "❌ 功能未启用 / Feature not enabled", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	// Verify the user is a bound admin (security check)
	if query.From == nil {
		h.logger.Warnw("callback query missing from user")
		h.answerCallback(query.ID, "❌ 无效请求 / Invalid request", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	// Check if the telegram user is a bound admin
	// SECURITY: adminService must be configured to verify permissions
	if h.adminService == nil {
		h.logger.Errorw("admin service not configured, cannot verify permissions")
		h.answerCallback(query.ID, "❌ 功能未启用 / Feature not enabled", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	binding, authErr := h.adminService.GetBindingByTelegramID(c.Request.Context(), query.From.ID)
	if authErr != nil || binding == nil {
		h.logger.Warnw("mute callback from non-admin user",
			"telegram_user_id", query.From.ID,
		)
		h.answerCallback(query.ID, "❌ 无权限操作 / Permission denied", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	var err error
	var resourceName string

	switch resourceType {
	case "agent":
		err = h.muteService.MuteAgentNotification(c.Request.Context(), resourceSID)
		resourceName = "转发代理 / Forward Agent"
	case "node":
		err = h.muteService.MuteNodeNotification(c.Request.Context(), resourceSID)
		resourceName = "Node Agent"
	default:
		h.logger.Warnw("unknown resource type for mute", "type", resourceType)
		h.answerCallback(query.ID, "❌ 未知资源类型 / Unknown resource type", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	if err != nil {
		h.logger.Errorw("failed to mute notification",
			"resource_type", resourceType,
			"resource_sid", resourceSID,
			"error", err,
		)
		h.answerCallback(query.ID, "❌ 操作失败 / Operation failed", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	// Answer callback with success message
	h.answerCallback(query.ID, "✅ 已静默此"+resourceName+"的通知 / Notifications muted", false)

	// Update the button to show unmute option
	if query.Message != nil && query.Message.Chat != nil && h.callbackAnswerer != nil {
		chatID := query.Message.Chat.ID
		messageID := query.Message.MessageID
		// Only update the button, don't modify message text (original message is HTML formatted)
		unmuteKeyboard := buildUnmuteKeyboard(resourceType, resourceSID)
		if editErr := h.callbackAnswerer.EditMessageReplyMarkup(chatID, messageID, unmuteKeyboard); editErr != nil {
			h.logger.Errorw("failed to update message reply markup after mute",
				"chat_id", chatID,
				"message_id", messageID,
				"error", editErr,
			)
		}
	}

	// Log with nil-safe access to telegram user ID
	var telegramUserID int64
	if query.From != nil {
		telegramUserID = query.From.ID
	}
	h.logger.Infow("notification muted via telegram callback",
		"resource_type", resourceType,
		"resource_sid", resourceSID,
		"telegram_user_id", telegramUserID,
	)

	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

// handleUnmuteCallback handles the unmute notification callback
func (h *Handler) handleUnmuteCallback(c *gin.Context, query *dto.CallbackQuery, resourceType, resourceSID string) {
	if h.muteService == nil {
		h.logger.Warnw("mute service not configured")
		h.answerCallback(query.ID, "❌ 功能未启用 / Feature not enabled", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	// Verify the user is a bound admin (security check)
	if query.From == nil {
		h.logger.Warnw("callback query missing from user")
		h.answerCallback(query.ID, "❌ 无效请求 / Invalid request", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	// Check if the telegram user is a bound admin
	// SECURITY: adminService must be configured to verify permissions
	if h.adminService == nil {
		h.logger.Errorw("admin service not configured, cannot verify permissions")
		h.answerCallback(query.ID, "❌ 功能未启用 / Feature not enabled", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	binding, authErr := h.adminService.GetBindingByTelegramID(c.Request.Context(), query.From.ID)
	if authErr != nil || binding == nil {
		h.logger.Warnw("unmute callback from non-admin user",
			"telegram_user_id", query.From.ID,
		)
		h.answerCallback(query.ID, "❌ 无权限操作 / Permission denied", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	var err error
	var resourceName string

	switch resourceType {
	case "agent":
		err = h.muteService.UnmuteAgentNotification(c.Request.Context(), resourceSID)
		resourceName = "转发代理 / Forward Agent"
	case "node":
		err = h.muteService.UnmuteNodeNotification(c.Request.Context(), resourceSID)
		resourceName = "Node Agent"
	default:
		h.logger.Warnw("unknown resource type for unmute", "type", resourceType)
		h.answerCallback(query.ID, "❌ 未知资源类型 / Unknown resource type", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	if err != nil {
		h.logger.Errorw("failed to unmute notification",
			"resource_type", resourceType,
			"resource_sid", resourceSID,
			"error", err,
		)
		h.answerCallback(query.ID, "❌ 操作失败 / Operation failed", true)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	// Answer callback with success message
	h.answerCallback(query.ID, "✅ 已解除静默此"+resourceName+"的通知 / Notifications unmuted", false)

	// Update the button to show mute option again
	if query.Message != nil && query.Message.Chat != nil && h.callbackAnswerer != nil {
		chatID := query.Message.Chat.ID
		messageID := query.Message.MessageID
		// Only update the button, don't modify message text
		muteKeyboard := buildMuteKeyboard(resourceType, resourceSID)
		if editErr := h.callbackAnswerer.EditMessageReplyMarkup(chatID, messageID, muteKeyboard); editErr != nil {
			h.logger.Errorw("failed to update message reply markup after unmute",
				"chat_id", chatID,
				"message_id", messageID,
				"error", editErr,
			)
		}
	}

	// Log with nil-safe access to telegram user ID
	var telegramUserID int64
	if query.From != nil {
		telegramUserID = query.From.ID
	}
	h.logger.Infow("notification unmuted via telegram callback",
		"resource_type", resourceType,
		"resource_sid", resourceSID,
		"telegram_user_id", telegramUserID,
	)

	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

// answerCallback sends a response to a callback query
func (h *Handler) answerCallback(callbackQueryID, text string, showAlert bool) {
	if h.callbackAnswerer != nil {
		_ = h.callbackAnswerer.AnswerCallbackQuery(callbackQueryID, text, showAlert)
	}
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
