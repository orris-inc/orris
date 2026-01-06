package admin

import (
	"context"

	"github.com/orris-inc/orris/internal/application/telegram/admin/usecases"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/subscription"
	telegramAdmin "github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AdminNotificationProcessor implements the scheduler.AdminNotificationProcessor interface
type AdminNotificationProcessor struct {
	checkOfflineUC      *usecases.CheckOfflineUseCase
	sendDailySummaryUC  *usecases.SendDailySummaryUseCase
	sendWeeklySummaryUC *usecases.SendWeeklySummaryUseCase
	logger              logger.Interface
}

// BotServiceProvider provides access to the bot service for sending messages
type BotServiceProvider interface {
	GetBotService() usecases.TelegramMessageSender
}

// NewAdminNotificationProcessor creates a new AdminNotificationProcessor
func NewAdminNotificationProcessor(
	bindingRepo telegramAdmin.AdminTelegramBindingRepository,
	userRepo user.Repository,
	subscriptionRepo subscription.SubscriptionRepository,
	usageRepo subscription.SubscriptionUsageRepository,
	nodeRepo node.NodeRepository,
	agentRepo forward.AgentRepository,
	alertDeduplicator *cache.AlertDeduplicator,
	botServiceProvider BotServiceProvider,
	log logger.Interface,
) *AdminNotificationProcessor {
	// Create a wrapper that gets the bot service dynamically
	botServiceWrapper := &dynamicBotServiceWrapper{provider: botServiceProvider}

	checkOfflineUC := usecases.NewCheckOfflineUseCase(
		bindingRepo,
		nodeRepo,
		agentRepo,
		alertDeduplicator,
		botServiceWrapper,
		log,
	)

	sendDailySummaryUC := usecases.NewSendDailySummaryUseCase(
		bindingRepo,
		userRepo,
		subscriptionRepo,
		usageRepo,
		nodeRepo,
		agentRepo,
		botServiceWrapper,
		log,
	)

	sendWeeklySummaryUC := usecases.NewSendWeeklySummaryUseCase(
		bindingRepo,
		userRepo,
		subscriptionRepo,
		usageRepo,
		nodeRepo,
		agentRepo,
		botServiceWrapper,
		log,
	)

	return &AdminNotificationProcessor{
		checkOfflineUC:      checkOfflineUC,
		sendDailySummaryUC:  sendDailySummaryUC,
		sendWeeklySummaryUC: sendWeeklySummaryUC,
		logger:              log,
	}
}

// CheckOffline checks for offline nodes and agents, sends alerts
func (p *AdminNotificationProcessor) CheckOffline(ctx context.Context) error {
	return p.checkOfflineUC.CheckAndNotify(ctx)
}

// SendDailySummary sends daily business summary
func (p *AdminNotificationProcessor) SendDailySummary(ctx context.Context) error {
	return p.sendDailySummaryUC.SendSummary(ctx)
}

// SendWeeklySummary sends weekly business summary
func (p *AdminNotificationProcessor) SendWeeklySummary(ctx context.Context) error {
	return p.sendWeeklySummaryUC.SendSummary(ctx)
}

// dynamicBotServiceWrapper wraps a BotServiceProvider to implement TelegramMessageSender
// This allows the bot service to be retrieved dynamically (supports hot-reload)
type dynamicBotServiceWrapper struct {
	provider BotServiceProvider
}

// SendMessage implements TelegramMessageSender (HTML format)
func (w *dynamicBotServiceWrapper) SendMessage(chatID int64, text string) error {
	botService := w.provider.GetBotService()
	if botService == nil {
		return nil // Bot service not available, skip silently
	}
	return botService.SendMessage(chatID, text)
}

// SendMessageWithInlineKeyboard implements TelegramMessageSender (HTML format with inline keyboard)
func (w *dynamicBotServiceWrapper) SendMessageWithInlineKeyboard(chatID int64, text string, keyboard any) error {
	botService := w.provider.GetBotService()
	if botService == nil {
		return nil // Bot service not available, skip silently
	}
	return botService.SendMessageWithInlineKeyboard(chatID, text, keyboard)
}
