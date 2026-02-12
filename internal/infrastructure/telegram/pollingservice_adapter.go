package telegram

import (
	"context"
)

// TelegramServiceForPolling defines the interface for telegram service operations needed by polling
type TelegramServiceForPolling interface {
	BindFromWebhookForPolling(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error
	UnbindByTelegramID(ctx context.Context, telegramUserID int64) error
	IsBoundByTelegramID(ctx context.Context, telegramUserID int64) (bool, error)
	SendBotMessage(chatID int64, text string) error
	SendBotMessageWithKeyboard(chatID int64, text string) error
	SendBotChatAction(chatID int64, action string) error
	UpdateBindingLanguage(ctx context.Context, telegramUserID int64, language string) error
	UpdateAdminBindingLanguage(ctx context.Context, telegramUserID int64, language string) error
	// Admin binding
	AdminBindFromPolling(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error
	AdminUnbindByTelegramID(ctx context.Context, telegramUserID int64) error
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

// ServiceAdapter wraps the telegram ServiceDDD to implement TelegramServiceForPolling
type ServiceAdapter struct {
	binder               TelegramBinderService
	adminBinder          AdminBinderService
	botServiceGetter     BotServiceGetter
	muteService          MuteNotificationService
	callbackAnswerer     CallbackAnswerer
	bindFunc             func(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error
	getBindingStatus     func(ctx context.Context, telegramUserID int64) (bool, error)
	updateLanguageFunc   func(ctx context.Context, telegramUserID int64, language string) error
	updateAdminLangFunc  func(ctx context.Context, telegramUserID int64, language string) error
}

// NewServiceAdapter creates a new service adapter from telegram ServiceDDD
func NewServiceAdapter(service interface {
	UnbindByTelegramID(ctx context.Context, telegramUserID int64) error
},
	bindFunc func(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error,
	getBindingStatus func(ctx context.Context, telegramUserID int64) (bool, error),
	updateLanguageFunc func(ctx context.Context, telegramUserID int64, language string) error,
	updateAdminLangFunc func(ctx context.Context, telegramUserID int64, language string) error,
) *ServiceAdapter {
	return &ServiceAdapter{
		binder:              service,
		bindFunc:            bindFunc,
		getBindingStatus:    getBindingStatus,
		updateLanguageFunc:  updateLanguageFunc,
		updateAdminLangFunc: updateAdminLangFunc,
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

// SendBotChatAction implements TelegramServiceForPolling
func (a *ServiceAdapter) SendBotChatAction(chatID int64, action string) error {
	botService := a.botServiceGetter.GetBotService()
	if botService == nil {
		return nil
	}
	return botService.SendChatAction(chatID, action)
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

// AdminUnbindByTelegramID implements TelegramServiceForPolling for admin unbinding
func (a *ServiceAdapter) AdminUnbindByTelegramID(ctx context.Context, telegramUserID int64) error {
	if a.adminBinder == nil {
		return nil
	}
	return a.adminBinder.UnbindByTelegramID(ctx, telegramUserID)
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

// UpdateBindingLanguage implements TelegramServiceForPolling
func (a *ServiceAdapter) UpdateBindingLanguage(ctx context.Context, telegramUserID int64, language string) error {
	if a.updateLanguageFunc == nil {
		return nil
	}
	return a.updateLanguageFunc(ctx, telegramUserID, language)
}

// UpdateAdminBindingLanguage implements TelegramServiceForPolling
func (a *ServiceAdapter) UpdateAdminBindingLanguage(ctx context.Context, telegramUserID int64, language string) error {
	if a.updateAdminLangFunc == nil {
		return nil
	}
	return a.updateAdminLangFunc(ctx, telegramUserID, language)
}
