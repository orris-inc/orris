package telegram

import (
	"context"
	"strings"

	"github.com/orris-inc/orris/internal/infrastructure/telegram/i18n"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

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
	langCode := update.Message.From.LanguageCode
	lang := i18n.DetectLang(langCode)

	switch {
	case strings.HasPrefix(text, "/start "):
		// Deep link: /start <payload>
		payload := strings.TrimSpace(strings.TrimPrefix(text, "/start "))
		return h.handleStartPayload(ctx, telegramUserID, username, lang, payload)
	case strings.HasPrefix(text, "/bind "):
		code := strings.TrimSpace(strings.TrimPrefix(text, "/bind "))
		return h.handleBindCommand(ctx, telegramUserID, username, lang, code)
	case text == "/unbind":
		return h.handleUnbindCommand(ctx, telegramUserID, lang)
	case text == "/status":
		return h.handleStatusCommand(ctx, telegramUserID, lang)
	case strings.HasPrefix(text, "/adminbind "):
		code := strings.TrimSpace(strings.TrimPrefix(text, "/adminbind "))
		return h.handleAdminBindCommand(ctx, telegramUserID, username, lang, code)
	case text == "/adminunbind":
		return h.handleAdminUnbindCommand(ctx, telegramUserID, lang)
	case text == "/adminstatus":
		return h.handleAdminStatusCommand(ctx, telegramUserID, lang)
	case text == "/start" || text == "/help":
		return h.handleHelpCommand(telegramUserID, lang)
	default:
		return h.handleHelpCommand(telegramUserID, lang)
	}
}

func (h *PollingUpdateHandler) handleStartPayload(ctx context.Context, telegramUserID int64, username string, lang i18n.Lang, payload string) error {
	// Reject overly long payloads to prevent abuse
	const maxPayloadLen = 128
	if len(payload) > maxPayloadLen {
		return h.handleHelpCommand(telegramUserID, lang)
	}

	switch {
	case strings.HasPrefix(payload, "bind_"):
		code := strings.TrimPrefix(payload, "bind_")
		return h.handleBindCommand(ctx, telegramUserID, username, lang, code)
	case strings.HasPrefix(payload, "adminbind_"):
		code := strings.TrimPrefix(payload, "adminbind_")
		return h.handleAdminBindCommand(ctx, telegramUserID, username, lang, code)
	default:
		return h.handleHelpCommand(telegramUserID, lang)
	}
}

func (h *PollingUpdateHandler) handleBindCommand(ctx context.Context, telegramUserID int64, username string, lang i18n.Lang, code string) error {
	if code == "" {
		return h.service.SendBotMessage(telegramUserID, i18n.MsgBindMissingCode(lang))
	}

	_ = h.service.SendBotChatAction(telegramUserID, "typing")
	err := h.service.BindFromWebhookForPolling(ctx, telegramUserID, username, code)
	if err != nil {
		h.logger.Errorw("failed to bind telegram from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, i18n.MsgBindFailed(lang))
	}

	// Update language after successful binding
	if err := h.service.UpdateBindingLanguage(ctx, telegramUserID, string(lang)); err != nil {
		h.logger.Debugw("failed to update binding language", "telegram_user_id", telegramUserID, "error", err)
	}

	return h.service.SendBotMessageWithKeyboard(telegramUserID, i18n.MsgBindSuccess(lang))
}

func (h *PollingUpdateHandler) handleUnbindCommand(ctx context.Context, telegramUserID int64, lang i18n.Lang) error {
	err := h.service.UnbindByTelegramID(ctx, telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to unbind telegram from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, i18n.MsgUnbindFailed(lang))
	}

	return h.service.SendBotMessage(telegramUserID, i18n.MsgUnbindSuccess(lang))
}

func (h *PollingUpdateHandler) handleStatusCommand(ctx context.Context, telegramUserID int64, lang i18n.Lang) error {
	_ = h.service.SendBotChatAction(telegramUserID, "typing")
	isBound, err := h.service.IsBoundByTelegramID(ctx, telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to get binding status from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, i18n.MsgStatusError(lang))
	}

	if !isBound {
		return h.service.SendBotMessage(telegramUserID, i18n.MsgStatusNotConnected(lang))
	}

	// Update language if bound
	if err := h.service.UpdateBindingLanguage(ctx, telegramUserID, string(lang)); err != nil {
		h.logger.Debugw("failed to update binding language", "telegram_user_id", telegramUserID, "error", err)
	}

	// For bound status, just send a generic message since we don't have detailed info in polling mode
	return h.service.SendBotMessage(telegramUserID, i18n.MsgStatusConnectedSimple(lang))
}

func (h *PollingUpdateHandler) handleHelpCommand(telegramUserID int64, lang i18n.Lang) error {
	return h.service.SendBotMessageWithKeyboard(telegramUserID, i18n.MsgHelpFull(lang))
}

func (h *PollingUpdateHandler) handleAdminBindCommand(ctx context.Context, telegramUserID int64, username string, lang i18n.Lang, code string) error {
	if code == "" {
		return h.service.SendBotMessage(telegramUserID, i18n.MsgAdminBindMissingCode(lang))
	}

	err := h.service.AdminBindFromPolling(ctx, telegramUserID, username, code)
	if err != nil {
		h.logger.Errorw("failed to bind admin telegram from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, i18n.MsgAdminBindFailedPolling(lang))
	}

	// Update language after successful binding
	if err := h.service.UpdateAdminBindingLanguage(ctx, telegramUserID, string(lang)); err != nil {
		h.logger.Debugw("failed to update admin binding language", "telegram_user_id", telegramUserID, "error", err)
	}

	return h.service.SendBotMessageWithKeyboard(telegramUserID, i18n.MsgAdminBindSuccessPolling(lang))
}

func (h *PollingUpdateHandler) handleAdminUnbindCommand(ctx context.Context, telegramUserID int64, lang i18n.Lang) error {
	err := h.service.AdminUnbindByTelegramID(ctx, telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to unbind admin telegram from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, i18n.MsgAdminUnbindFailed(lang))
	}

	return h.service.SendBotMessage(telegramUserID, i18n.MsgAdminUnbindSuccess(lang))
}

func (h *PollingUpdateHandler) handleAdminStatusCommand(ctx context.Context, telegramUserID int64, lang i18n.Lang) error {
	isAdmin, err := h.service.IsAdminBound(ctx, telegramUserID)
	if err != nil || !isAdmin {
		return h.service.SendBotMessage(telegramUserID, i18n.MsgAdminStatusNotBound(lang))
	}

	// Update language if bound
	if err := h.service.UpdateAdminBindingLanguage(ctx, telegramUserID, string(lang)); err != nil {
		h.logger.Debugw("failed to update admin binding language", "telegram_user_id", telegramUserID, "error", err)
	}

	return h.service.SendBotMessage(telegramUserID, i18n.MsgAdminStatusBound(lang))
}

// handleCallbackQuery handles callback queries from inline keyboard buttons
func (h *PollingUpdateHandler) handleCallbackQuery(ctx context.Context, query *CallbackQuery) error {
	if query == nil || query.Data == "" {
		return nil
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
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackInvalidAction(lang), true)
		return nil
	}

	action := parts[0]
	resourceType := parts[1]
	resourceSID := parts[2]

	// Validate SID format (defense-in-depth)
	if _, _, err := id.ParsePrefixedID(resourceSID); err != nil {
		h.logger.Warnw("invalid resource SID in callback", "sid", resourceSID)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackInvalidAction(lang), true)
		return nil
	}

	switch action {
	case "mute":
		return h.handleMuteCallback(ctx, query, lang, resourceType, resourceSID)
	case "unmute":
		return h.handleUnmuteCallback(ctx, query, lang, resourceType, resourceSID)
	default:
		h.logger.Warnw("unknown callback action", "action", action)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackUnknownAction(lang), true)
		return nil
	}
}

// handleMuteCallback handles the mute notification callback
func (h *PollingUpdateHandler) handleMuteCallback(ctx context.Context, query *CallbackQuery, lang i18n.Lang, resourceType, resourceSID string) error {
	// Verify the user is a bound admin (security check)
	if query.From == nil {
		h.logger.Warnw("callback query missing from user")
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackInvalidRequest(lang), true)
		return nil
	}

	isAdmin, err := h.service.IsAdminBound(ctx, query.From.ID)
	if err != nil || !isAdmin {
		h.logger.Warnw("mute callback from non-admin user",
			"telegram_user_id", query.From.ID,
		)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackPermissionDenied(lang), true)
		return nil
	}

	// Execute mute operation
	var muteErr error
	switch resourceType {
	case "agent":
		muteErr = h.service.MuteAgentNotification(ctx, resourceSID)
	case "node":
		muteErr = h.service.MuteNodeNotification(ctx, resourceSID)
	default:
		h.logger.Warnw("unknown resource type for mute", "type", resourceType)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackUnknownResourceType(lang), true)
		return nil
	}

	if muteErr != nil {
		h.logger.Errorw("failed to mute notification",
			"resource_type", resourceType,
			"resource_sid", resourceSID,
			"error", muteErr,
		)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackOperationFailed(lang), true)
		return nil
	}

	// Answer callback with success message
	successMsg := i18n.MsgCallbackMuteSuccess(lang) + i18n.ResourceName(lang, resourceType)
	_ = h.service.AnswerCallbackQuery(query.ID, successMsg, false)

	// Update the button to show unmute option
	if query.Message != nil && query.Message.Chat != nil {
		chatID := query.Message.Chat.ID
		messageID := query.Message.MessageID
		unmuteKeyboard := i18n.BuildUnmuteKeyboard(lang, resourceType, resourceSID)
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
func (h *PollingUpdateHandler) handleUnmuteCallback(ctx context.Context, query *CallbackQuery, lang i18n.Lang, resourceType, resourceSID string) error {
	// Verify the user is a bound admin (security check)
	if query.From == nil {
		h.logger.Warnw("callback query missing from user")
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackInvalidRequest(lang), true)
		return nil
	}

	isAdmin, err := h.service.IsAdminBound(ctx, query.From.ID)
	if err != nil || !isAdmin {
		h.logger.Warnw("unmute callback from non-admin user",
			"telegram_user_id", query.From.ID,
		)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackPermissionDenied(lang), true)
		return nil
	}

	// Execute unmute operation
	var unmuteErr error
	switch resourceType {
	case "agent":
		unmuteErr = h.service.UnmuteAgentNotification(ctx, resourceSID)
	case "node":
		unmuteErr = h.service.UnmuteNodeNotification(ctx, resourceSID)
	default:
		h.logger.Warnw("unknown resource type for unmute", "type", resourceType)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackUnknownResourceType(lang), true)
		return nil
	}

	if unmuteErr != nil {
		h.logger.Errorw("failed to unmute notification",
			"resource_type", resourceType,
			"resource_sid", resourceSID,
			"error", unmuteErr,
		)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackOperationFailed(lang), true)
		return nil
	}

	// Answer callback with success message
	successMsg := i18n.MsgCallbackUnmuteSuccess(lang) + i18n.ResourceName(lang, resourceType)
	_ = h.service.AnswerCallbackQuery(query.ID, successMsg, false)

	// Update the button to show mute option again
	if query.Message != nil && query.Message.Chat != nil {
		chatID := query.Message.Chat.ID
		messageID := query.Message.MessageID
		muteKeyboard := i18n.BuildMuteKeyboard(lang, resourceType, resourceSID)
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
