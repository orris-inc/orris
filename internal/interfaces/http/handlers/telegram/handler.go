package telegram

import (
	"crypto/subtle"
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
	service       *telegramApp.ServiceDDD
	logger        logger.Interface
	webhookSecret string
}

// NewHandler creates a new telegram handler
func NewHandler(service *telegramApp.ServiceDDD, logger logger.Interface, webhookSecret string) *Handler {
	return &Handler{
		service:       service,
		logger:        logger,
		webhookSecret: webhookSecret,
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
	// Verify webhook secret - REQUIRED for security
	// If webhook secret is not configured, reject all requests to prevent unauthorized access
	if h.webhookSecret == "" {
		h.logger.Errorw("webhook secret not configured, rejecting request for security")
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "webhook not configured")
		return
	}

	secretHeader := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(secretHeader), []byte(h.webhookSecret)) != 1 {
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
		msg := "⚠️ *缺少验证码 / Missing Code*\n\n" +
			"用法 Usage: `/bind <code>`\n\n" +
			"请在网站设置页面获取验证码\n" +
			"Get your code from website settings"
		_ = h.service.SendBotMessage(telegramUserID, msg)
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "missing code"})
		return
	}

	resp, err := h.service.BindFromWebhook(c.Request.Context(), telegramUserID, username, code)
	if err != nil {
		h.logger.Errorw("failed to bind telegram from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		msg := "❌ *绑定失败 / Binding Failed*\n\n" +
			"验证码无效或已过期\n" +
			"Invalid or expired verification code\n\n" +
			"请检查验证码后重试\n" +
			"Please check your code and try again"
		_ = h.service.SendBotMessage(telegramUserID, msg)
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "binding failed"})
		return
	}

	// Send success message with reply keyboard for easy access to commands
	msg := "✅ *绑定成功 / Binding Successful*\n\n" +
		"🔔 您将收到以下通知 / You will receive:\n" +
		"• 订阅到期提醒 / Expiry reminders\n" +
		"• 流量使用警告 / Traffic alerts\n\n" +
		"使用 /status 查看设置，/unbind 解绑"
	_ = h.service.SendBotMessageWithKeyboard(telegramUserID, msg)
	utils.SuccessResponse(c, http.StatusOK, "success", resp)
}

func (h *Handler) handleUnbindCommand(c *gin.Context, telegramUserID int64) {
	err := h.service.UnbindByTelegramID(c.Request.Context(), telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to unbind telegram from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		msg := "❌ *解绑失败 / Unbind Failed*\n\n" +
			"操作失败，请稍后重试\n" +
			"Operation failed, please try again later"
		_ = h.service.SendBotMessage(telegramUserID, msg)
		utils.SuccessResponse(c, http.StatusOK, "error", gin.H{"message": "unbind failed"})
		return
	}

	msg := "✅ *已解绑 / Account Unbound*\n\n" +
		"🔕 您将不再收到通知\n" +
		"You will no longer receive notifications\n\n" +
		"随时使用 /bind <code> 重新连接"
	_ = h.service.SendBotMessage(telegramUserID, msg)
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func (h *Handler) handleStatusCommand(c *gin.Context, telegramUserID int64) {
	status, err := h.service.GetBindingStatusByTelegramID(c.Request.Context(), telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to get binding status from webhook",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		msg := "❌ *错误 / Error*\n\n" +
			"获取状态失败，请稍后重试\n" +
			"Failed to get status, please try again later"
		_ = h.service.SendBotMessage(telegramUserID, msg)
		utils.SuccessResponse(c, http.StatusOK, "error", nil)
		return
	}

	if !status.IsBound {
		msg := "🔗 *未连接 / Not Connected*\n\n" +
			"您的 Telegram 尚未绑定账户\n\n" +
			"*绑定步骤 / How to connect:*\n" +
			"1️⃣ 访问网站设置页面\n" +
			"2️⃣ 点击「绑定 Telegram」\n" +
			"3️⃣ 复制验证码\n" +
			"4️⃣ 发送 `/bind <验证码>`"
		_ = h.service.SendBotMessage(telegramUserID, msg)
	} else {
		msg := "📊 *通知设置 / Settings*\n\n" +
			"*状态 Status:* 🟢 已连接 Connected\n\n" +
			"┌ *到期提醒 / Expiry Reminders*\n" +
			"│ " + boolToStatusBilingual(status.Binding.NotifyExpiring) + "\n" +
			"│ 提前 " + strconv.Itoa(status.Binding.ExpiringDays) + " 天提醒\n" +
			"└\n" +
			"┌ *流量警告 / Traffic Alerts*\n" +
			"│ " + boolToStatusBilingual(status.Binding.NotifyTraffic) + "\n" +
			"│ 阈值 Threshold: " + strconv.Itoa(status.Binding.TrafficThreshold) + "%\n" +
			"└\n\n" +
			"_在网站修改设置 / Modify on website_"
		_ = h.service.SendBotMessage(telegramUserID, msg)
	}
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func (h *Handler) handleHelpCommand(c *gin.Context, telegramUserID int64) {
	helpMsg := "🤖 *Orris 通知机器人*\n\n" +
		"订阅到期和流量使用提醒服务\n" +
		"Subscription & traffic notification service\n\n" +
		"*命令 Commands:*\n" +
		"├ /bind `<code>` — 绑定账户 Link account\n" +
		"├ /status — 查看设置 View settings\n" +
		"├ /unbind — 解绑账户 Disconnect\n" +
		"└ /help — 显示帮助 Show help\n\n" +
		"*开始使用 Getting Started:*\n" +
		"在网站设置页面获取验证码，然后发送 `/bind <code>` 完成绑定"
	// Send help message with reply keyboard for easy access to commands
	_ = h.service.SendBotMessageWithKeyboard(telegramUserID, helpMsg)
	utils.SuccessResponse(c, http.StatusOK, "success", nil)
}

func boolToStatus(b bool) string {
	if b {
		return "✅ ON"
	}
	return "❌ OFF"
}

func boolToStatusBilingual(b bool) string {
	if b {
		return "✅ 开启 ON"
	}
	return "❌ 关闭 OFF"
}
