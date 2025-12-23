package telegram

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	telegramApp "github.com/orris-inc/orris/internal/application/telegram"
	"github.com/orris-inc/orris/internal/application/telegram/dto"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// Handler handles telegram-related HTTP requests
type Handler struct {
	service *telegramApp.ServiceDDD
	logger  logger.Interface
}

// NewHandler creates a new telegram handler
func NewHandler(service *telegramApp.ServiceDDD, logger logger.Interface) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
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
	var update dto.WebhookUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		h.logger.Errorw("failed to parse webhook update", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
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
	default:
		// Unknown command, show help
		h.handleHelpCommand(c, telegramUserID)
	}
}

func (h *Handler) handleBindCommand(c *gin.Context, telegramUserID int64, username, code string) {
	if code == "" {
		_ = h.service.SendBotMessage(telegramUserID, "Please provide a verification code. Usage: /bind <code>")
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "missing code"})
		return
	}

	resp, err := h.service.BindFromWebhook(c.Request.Context(), telegramUserID, username, code)
	if err != nil {
		h.logger.Errorw("failed to bind telegram from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		_ = h.service.SendBotMessage(telegramUserID, "Binding failed: "+err.Error())
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": err.Error()})
		return
	}

	_ = h.service.SendBotMessage(telegramUserID, "✅ Binding successful! You will now receive notifications.\n\nUse /status to check your settings.")
	utils.SuccessResponse(c, http.StatusOK, "success", resp)
}

func (h *Handler) handleUnbindCommand(c *gin.Context, telegramUserID int64) {
	err := h.service.UnbindByTelegramID(c.Request.Context(), telegramUserID)
	if err != nil {
		_ = h.service.SendBotMessage(telegramUserID, "Unbind failed: "+err.Error())
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": err.Error()})
		return
	}

	_ = h.service.SendBotMessage(telegramUserID, "✅ Successfully unbound. You will no longer receive notifications.")
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func (h *Handler) handleStatusCommand(c *gin.Context, telegramUserID int64) {
	status, err := h.service.GetBindingStatusByTelegramID(c.Request.Context(), telegramUserID)
	if err != nil {
		_ = h.service.SendBotMessage(telegramUserID, "Failed to get status: "+err.Error())
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	if !status.IsBound {
		_ = h.service.SendBotMessage(telegramUserID, "❌ You are not bound to any account.\n\nTo bind your account:\n1. Go to your account settings on the website\n2. Click 'Bind Telegram'\n3. Copy the verification code\n4. Send /bind <code> here")
	} else {
		msg := "📊 *Notification Settings*\n\n" +
			"Expiring Notifications: " + boolToStatus(status.Binding.NotifyExpiring) + "\n" +
			"Traffic Notifications: " + boolToStatus(status.Binding.NotifyTraffic) + "\n" +
			"Expiring Days: " + strconv.Itoa(status.Binding.ExpiringDays) + " days\n" +
			"Traffic Threshold: " + strconv.Itoa(status.Binding.TrafficThreshold) + "%"
		_ = h.service.SendBotMessage(telegramUserID, msg)
	}
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func (h *Handler) handleHelpCommand(c *gin.Context, telegramUserID int64) {
	helpMsg := "🤖 *Available Commands*\n\n" +
		"/bind <code> - Bind your account using the verification code\n" +
		"/unbind - Unbind your account\n" +
		"/status - View your notification settings\n" +
		"/help - Show this help message\n\n" +
		"To get started, visit your account settings on the website to get a verification code."
	_ = h.service.SendBotMessage(telegramUserID, helpMsg)
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func boolToStatus(b bool) string {
	if b {
		return "✅ ON"
	}
	return "❌ OFF"
}
