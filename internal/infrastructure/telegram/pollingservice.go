package telegram

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdateHandler defines the interface for handling Telegram updates
type UpdateHandler interface {
	HandleUpdate(ctx context.Context, update *Update) error
}

// PollingService handles long polling for Telegram updates
type PollingService struct {
	botService    *BotService
	handler       UpdateHandler
	logger        logger.Interface
	pollTimeout   int
	stopChan      chan struct{}
	wg            sync.WaitGroup
	lastUpdateID  int64
	isRunning     bool
	runningMu     sync.Mutex
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
	s.runningMu.Unlock()

	// Delete any existing webhook before starting polling
	if err := s.botService.DeleteWebhook(); err != nil {
		s.logger.Warnw("failed to delete webhook before polling", "error", err)
	}

	s.logger.Infow("starting telegram polling service", "timeout", s.pollTimeout)

	s.wg.Add(1)
	go s.pollLoop(ctx)

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
	updates, err := s.botService.GetUpdates(offset, s.pollTimeout)
	if err != nil {
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
}

// TelegramBinderService defines the interface for binding operations
type TelegramBinderService interface {
	UnbindByTelegramID(ctx context.Context, telegramUserID int64) error
	SendBotMessage(chatID int64, text string) error
	SendBotMessageWithKeyboard(chatID int64, text string) error
}

// ServiceAdapter wraps the telegram ServiceDDD to implement TelegramServiceForPolling
type ServiceAdapter struct {
	binder           TelegramBinderService
	bindFunc         func(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error
	getBindingStatus func(ctx context.Context, telegramUserID int64) (bool, error)
}

// NewServiceAdapter creates a new service adapter from telegram ServiceDDD
func NewServiceAdapter(service interface {
	UnbindByTelegramID(ctx context.Context, telegramUserID int64) error
	SendBotMessage(chatID int64, text string) error
	SendBotMessageWithKeyboard(chatID int64, text string) error
}, bindFunc func(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error,
	getBindingStatus func(ctx context.Context, telegramUserID int64) (bool, error),
) *ServiceAdapter {
	return &ServiceAdapter{
		binder:           service,
		bindFunc:         bindFunc,
		getBindingStatus: getBindingStatus,
	}
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
	return a.binder.SendBotMessage(chatID, text)
}

// SendBotMessageWithKeyboard implements TelegramServiceForPolling
func (a *ServiceAdapter) SendBotMessageWithKeyboard(chatID int64, text string) error {
	return a.binder.SendBotMessageWithKeyboard(chatID, text)
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
	case text == "/start" || text == "/help":
		return h.handleHelpCommand(telegramUserID)
	default:
		return h.handleHelpCommand(telegramUserID)
	}
}

func (h *PollingUpdateHandler) handleBindCommand(ctx context.Context, telegramUserID int64, username, code string) error {
	if code == "" {
		msg := "⚠️ *缺少验证码 / Missing Code*\n\n" +
			"用法 Usage: `/bind <code>`\n\n" +
			"请在网站设置页面获取验证码\n" +
			"Get your code from website settings"
		return h.service.SendBotMessage(telegramUserID, msg)
	}

	err := h.service.BindFromWebhookForPolling(ctx, telegramUserID, username, code)
	if err != nil {
		h.logger.Errorw("failed to bind telegram from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		msg := "❌ *绑定失败 / Binding Failed*\n\n" +
			"验证码无效或已过期\n" +
			"Invalid or expired verification code\n\n" +
			"请检查验证码后重试\n" +
			"Please check your code and try again"
		return h.service.SendBotMessage(telegramUserID, msg)
	}

	msg := "✅ *绑定成功 / Binding Successful*\n\n" +
		"🔔 您将收到以下通知 / You will receive:\n" +
		"• 订阅到期提醒 / Expiry reminders\n" +
		"• 流量使用警告 / Traffic alerts\n\n" +
		"使用 /status 查看设置，/unbind 解绑"
	return h.service.SendBotMessageWithKeyboard(telegramUserID, msg)
}

func (h *PollingUpdateHandler) handleUnbindCommand(ctx context.Context, telegramUserID int64) error {
	err := h.service.UnbindByTelegramID(ctx, telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to unbind telegram from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		msg := "❌ *解绑失败 / Unbind Failed*\n\n" +
			"操作失败，请稍后重试\n" +
			"Operation failed, please try again later"
		return h.service.SendBotMessage(telegramUserID, msg)
	}

	msg := "✅ *已解绑 / Account Unbound*\n\n" +
		"🔕 您将不再收到通知\n" +
		"You will no longer receive notifications\n\n" +
		"随时使用 /bind <code> 重新连接"
	return h.service.SendBotMessage(telegramUserID, msg)
}

func (h *PollingUpdateHandler) handleStatusCommand(ctx context.Context, telegramUserID int64) error {
	isBound, err := h.service.IsBoundByTelegramID(ctx, telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to get binding status from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		msg := "❌ *错误 / Error*\n\n" +
			"获取状态失败，请稍后重试\n" +
			"Failed to get status, please try again later"
		return h.service.SendBotMessage(telegramUserID, msg)
	}

	if !isBound {
		msg := "🔗 *未连接 / Not Connected*\n\n" +
			"您的 Telegram 尚未绑定账户\n\n" +
			"*绑定步骤 / How to connect:*\n" +
			"1️⃣ 访问网站设置页面\n" +
			"2️⃣ 点击「绑定 Telegram」\n" +
			"3️⃣ 复制验证码\n" +
			"4️⃣ 发送 `/bind <验证码>`"
		return h.service.SendBotMessage(telegramUserID, msg)
	}

	// For bound status, just send a generic message since we don't have detailed info in polling mode
	msg := "📊 *已连接 / Connected*\n\n" +
		"您的账户已绑定\n" +
		"Your account is linked\n\n" +
		"使用 /unbind 解绑"
	return h.service.SendBotMessage(telegramUserID, msg)
}

func (h *PollingUpdateHandler) handleHelpCommand(telegramUserID int64) error {
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
	return h.service.SendBotMessageWithKeyboard(telegramUserID, helpMsg)
}
