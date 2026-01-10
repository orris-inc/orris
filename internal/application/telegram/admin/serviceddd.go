package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
	"github.com/orris-inc/orris/internal/application/telegram/admin/usecases"
	telegramAdmin "github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// messageSendDelay is the delay between sending messages to avoid Telegram API rate limiting
	// Telegram allows ~30 messages per second to different users
	messageSendDelay = 50 * time.Millisecond

	// startupCooldown is the cooldown period after service startup
	// During this period, online notifications are suppressed to avoid
	// sending batch notifications when nodes/agents reconnect after restart
	startupCooldown = 60 * time.Second
)

// TelegramMessageSender sends messages via Telegram (HTML format)
type TelegramMessageSender interface {
	SendMessage(chatID int64, text string) error
	SendMessageWithInlineKeyboard(chatID int64, text string, keyboard any) error
}

// BotLinkProvider provides the Telegram bot link
type BotLinkProvider interface {
	GetBotLink() string
}

// VerifyCodeStore provides verification code generation and verification
type VerifyCodeStore interface {
	Generate(ctx context.Context, userID uint) (string, error)
	Verify(ctx context.Context, code string) (uint, error)
	Delete(ctx context.Context, code string) error
}

// UserRoleChecker checks if a user is admin
type UserRoleChecker interface {
	IsAdmin(ctx context.Context, userID uint) (bool, error)
}

// ServiceDDD provides admin notification services
type ServiceDDD struct {
	bindingRepo     telegramAdmin.AdminTelegramBindingRepository
	verifyStore     VerifyCodeStore
	botService      TelegramMessageSender
	botLinkProvider BotLinkProvider
	userRoleChecker UserRoleChecker
	logger          logger.Interface
	startedAt       time.Time // Service startup time for cooldown calculation
}

// NewServiceDDD creates a new admin notification service
func NewServiceDDD(
	bindingRepo telegramAdmin.AdminTelegramBindingRepository,
	verifyStore VerifyCodeStore,
	botService TelegramMessageSender,
	botLinkProvider BotLinkProvider,
	userRoleChecker UserRoleChecker,
	logger logger.Interface,
) *ServiceDDD {
	return &ServiceDDD{
		bindingRepo:     bindingRepo,
		verifyStore:     verifyStore,
		botService:      botService,
		botLinkProvider: botLinkProvider,
		userRoleChecker: userRoleChecker,
		logger:          logger,
		startedAt:       biztime.NowUTC(),
	}
}

// GetBindingStatus returns the binding status for an admin user
func (s *ServiceDDD) GetBindingStatus(ctx context.Context, userID uint) (*dto.AdminBindingStatusResponse, error) {
	// Check if user is admin
	isAdmin, err := s.userRoleChecker.IsAdmin(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check admin status: %w", err)
	}
	if !isAdmin {
		return nil, telegramAdmin.ErrNotAdmin
	}

	binding, err := s.bindingRepo.GetByUserID(ctx, userID)
	if err != nil && err != telegramAdmin.ErrBindingNotFound {
		return nil, err
	}

	var botLink string
	if s.botLinkProvider != nil {
		botLink = s.botLinkProvider.GetBotLink()
	}

	if binding != nil {
		return &dto.AdminBindingStatusResponse{
			IsBound: true,
			Binding: &dto.AdminTelegramBindingResponse{
				SID:                     binding.SID(),
				TelegramUserID:          binding.TelegramUserID(),
				TelegramUsername:        binding.TelegramUsername(),
				NotifyNodeOffline:       binding.NotifyNodeOffline(),
				NotifyAgentOffline:      binding.NotifyAgentOffline(),
				NotifyNewUser:           binding.NotifyNewUser(),
				NotifyPaymentSuccess:    binding.NotifyPaymentSuccess(),
				NotifyDailySummary:      binding.NotifyDailySummary(),
				NotifyWeeklySummary:     binding.NotifyWeeklySummary(),
				OfflineThresholdMinutes: binding.OfflineThresholdMinutes(),
				CreatedAt:               binding.CreatedAt(),
			},
			BotLink: botLink,
		}, nil
	}

	// Generate verify code for unbound user
	verifyCode, err := s.verifyStore.Generate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verify code: %w", err)
	}

	return &dto.AdminBindingStatusResponse{
		IsBound:    false,
		VerifyCode: verifyCode,
		BotLink:    botLink,
		ExpiresAt:  biztime.NowUTC().Add(10 * time.Minute),
	}, nil
}

// BindFromWebhook binds an admin telegram account from webhook
func (s *ServiceDDD) BindFromWebhook(ctx context.Context, verifyCode string, telegramUserID int64, telegramUsername string) (any, error) {
	// Verify code and get user ID
	userID, err := s.verifyStore.Verify(ctx, verifyCode)
	if err != nil {
		return nil, telegramAdmin.ErrInvalidVerifyCode
	}

	// Check if user is admin
	isAdmin, err := s.userRoleChecker.IsAdmin(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check admin status: %w", err)
	}
	if !isAdmin {
		return nil, telegramAdmin.ErrNotAdmin
	}

	// Check if user already has a binding
	existing, err := s.bindingRepo.GetByUserID(ctx, userID)
	if err != nil && err != telegramAdmin.ErrBindingNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, telegramAdmin.ErrAlreadyBound
	}

	// Check if telegram account is already used
	existingTg, err := s.bindingRepo.GetByTelegramUserID(ctx, telegramUserID)
	if err != nil && err != telegramAdmin.ErrBindingNotFound {
		return nil, err
	}
	if existingTg != nil {
		return nil, telegramAdmin.ErrTelegramAlreadyUsed
	}

	// Create binding
	binding, err := telegramAdmin.NewAdminTelegramBinding(userID, telegramUserID, telegramUsername)
	if err != nil {
		return nil, fmt.Errorf("failed to create binding: %w", err)
	}

	if err := s.bindingRepo.Create(ctx, binding); err != nil {
		return nil, fmt.Errorf("failed to save binding: %w", err)
	}

	s.logger.Infow("admin telegram binding created",
		"user_id", userID,
		"telegram_user_id", telegramUserID,
		"binding_sid", binding.SID(),
	)

	return binding, nil
}

// Unbind unbinds an admin telegram account
func (s *ServiceDDD) Unbind(ctx context.Context, userID uint) error {
	// Check if user is admin
	isAdmin, err := s.userRoleChecker.IsAdmin(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to check admin status: %w", err)
	}
	if !isAdmin {
		return telegramAdmin.ErrNotAdmin
	}

	binding, err := s.bindingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.bindingRepo.Delete(ctx, binding.ID()); err != nil {
		return fmt.Errorf("failed to delete binding: %w", err)
	}

	s.logger.Infow("admin telegram binding deleted",
		"user_id", userID,
		"binding_sid", binding.SID(),
	)

	return nil
}

// UnbindByTelegramID unbinds by telegram user ID (for /adminunbind command)
func (s *ServiceDDD) UnbindByTelegramID(ctx context.Context, telegramUserID int64) error {
	binding, err := s.bindingRepo.GetByTelegramUserID(ctx, telegramUserID)
	if err != nil {
		return err
	}

	if err := s.bindingRepo.Delete(ctx, binding.ID()); err != nil {
		return fmt.Errorf("failed to delete binding: %w", err)
	}

	s.logger.Infow("admin telegram binding deleted via telegram command",
		"telegram_user_id", telegramUserID,
		"binding_sid", binding.SID(),
	)

	return nil
}

// UpdatePreferences updates notification preferences
func (s *ServiceDDD) UpdatePreferences(ctx context.Context, userID uint, req *dto.UpdateAdminPreferencesRequest) (*dto.AdminTelegramBindingResponse, error) {
	// Check if user is admin
	isAdmin, err := s.userRoleChecker.IsAdmin(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check admin status: %w", err)
	}
	if !isAdmin {
		return nil, telegramAdmin.ErrNotAdmin
	}

	binding, err := s.bindingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := binding.UpdatePreferences(
		req.NotifyNodeOffline,
		req.NotifyAgentOffline,
		req.NotifyNewUser,
		req.NotifyPaymentSuccess,
		req.NotifyDailySummary,
		req.NotifyWeeklySummary,
		req.OfflineThresholdMinutes,
	); err != nil {
		return nil, err
	}

	if err := s.bindingRepo.Update(ctx, binding); err != nil {
		return nil, fmt.Errorf("failed to update binding: %w", err)
	}

	return &dto.AdminTelegramBindingResponse{
		SID:                     binding.SID(),
		TelegramUserID:          binding.TelegramUserID(),
		TelegramUsername:        binding.TelegramUsername(),
		NotifyNodeOffline:       binding.NotifyNodeOffline(),
		NotifyAgentOffline:      binding.NotifyAgentOffline(),
		NotifyNewUser:           binding.NotifyNewUser(),
		NotifyPaymentSuccess:    binding.NotifyPaymentSuccess(),
		NotifyDailySummary:      binding.NotifyDailySummary(),
		NotifyWeeklySummary:     binding.NotifyWeeklySummary(),
		OfflineThresholdMinutes: binding.OfflineThresholdMinutes(),
		CreatedAt:               binding.CreatedAt(),
	}, nil
}

// GetBindingByTelegramID gets binding by telegram user ID
func (s *ServiceDDD) GetBindingByTelegramID(ctx context.Context, telegramUserID int64) (any, error) {
	return s.bindingRepo.GetByTelegramUserID(ctx, telegramUserID)
}

// NotifyNewUser implements AdminNotifier interface
func (s *ServiceDDD) NotifyNewUser(ctx context.Context, cmd NotifyNewUserCommand) error {
	if s.botService == nil {
		s.logger.Debugw("admin notification skipped: bot service not available", "type", "new_user")
		return nil
	}

	bindings, err := s.bindingRepo.FindBindingsForNewUserNotification(ctx)
	if err != nil {
		s.logger.Errorw("failed to find bindings for new user notification", "error", err)
		return err
	}

	if len(bindings) == 0 {
		return nil
	}

	message := usecases.BuildNewUserMessage(cmd.UserSID, cmd.Email, cmd.Name, cmd.Source, cmd.CreatedAt)

	for i, binding := range bindings {
		if err := s.botService.SendMessage(binding.TelegramUserID(), message); err != nil {
			s.logger.Errorw("failed to send new user notification",
				"telegram_user_id", binding.TelegramUserID(),
				"error", err,
			)
			continue
		}
		// Rate limiting: add delay between messages to avoid Telegram API throttling
		if i < len(bindings)-1 {
			time.Sleep(messageSendDelay)
		}
	}

	return nil
}

// NotifyPaymentSuccess implements AdminNotifier interface
func (s *ServiceDDD) NotifyPaymentSuccess(ctx context.Context, cmd NotifyPaymentSuccessCommand) error {
	if s.botService == nil {
		s.logger.Debugw("admin notification skipped: bot service not available", "type", "payment_success")
		return nil
	}

	bindings, err := s.bindingRepo.FindBindingsForPaymentSuccessNotification(ctx)
	if err != nil {
		s.logger.Errorw("failed to find bindings for payment success notification", "error", err)
		return err
	}

	if len(bindings) == 0 {
		return nil
	}

	message := usecases.BuildPaymentSuccessMessage(
		cmd.PaymentSID,
		cmd.UserSID,
		cmd.UserEmail,
		cmd.PlanName,
		cmd.Amount,
		cmd.Currency,
		cmd.PaymentMethod,
		cmd.TransactionID,
		cmd.PaidAt,
	)

	for i, binding := range bindings {
		if err := s.botService.SendMessage(binding.TelegramUserID(), message); err != nil {
			s.logger.Errorw("failed to send payment success notification",
				"telegram_user_id", binding.TelegramUserID(),
				"error", err,
			)
			continue
		}
		// Rate limiting: add delay between messages to avoid Telegram API throttling
		if i < len(bindings)-1 {
			time.Sleep(messageSendDelay)
		}
	}

	return nil
}

// NotifyNodeOnline implements AdminNotifier interface
func (s *ServiceDDD) NotifyNodeOnline(ctx context.Context, cmd NotifyNodeOnlineCommand) error {
	if s.botService == nil {
		s.logger.Debugw("admin notification skipped: bot service not available", "type", "node_online")
		return nil
	}

	// Skip if notification is muted for this node
	if cmd.MuteNotification {
		s.logger.Debugw("node online notification skipped: muted",
			"node_sid", cmd.NodeSID,
			"node_name", cmd.NodeName,
		)
		return nil
	}

	// Skip online notifications during startup cooldown period
	// This prevents batch notifications when nodes reconnect after service restart
	if time.Since(s.startedAt) < startupCooldown {
		s.logger.Debugw("node online notification skipped: startup cooldown",
			"node_sid", cmd.NodeSID,
			"cooldown_remaining", startupCooldown-time.Since(s.startedAt),
		)
		return nil
	}

	// Use dedicated method for online notification (no deduplication threshold)
	bindings, err := s.bindingRepo.FindBindingsForNodeOnlineNotification(ctx)
	if err != nil {
		s.logger.Errorw("failed to find bindings for node online notification", "error", err)
		return err
	}

	if len(bindings) == 0 {
		return nil
	}

	message := usecases.BuildNodeOnlineMessage(cmd.NodeSID, cmd.NodeName, biztime.NowUTC())
	keyboard := buildMuteKeyboard("node", cmd.NodeSID)

	for i, binding := range bindings {
		if err := s.botService.SendMessageWithInlineKeyboard(binding.TelegramUserID(), message, keyboard); err != nil {
			s.logger.Errorw("failed to send node online notification",
				"telegram_user_id", binding.TelegramUserID(),
				"error", err,
			)
			continue
		}
		// Rate limiting: add delay between messages to avoid Telegram API throttling
		if i < len(bindings)-1 {
			time.Sleep(messageSendDelay)
		}
	}

	return nil
}

// NotifyNodeOffline implements AdminNotifier interface
func (s *ServiceDDD) NotifyNodeOffline(ctx context.Context, cmd NotifyNodeOfflineCommand) error {
	if s.botService == nil {
		s.logger.Debugw("admin notification skipped: bot service not available", "type", "node_offline")
		return nil
	}

	// Skip if notification is muted for this node
	if cmd.MuteNotification {
		s.logger.Debugw("node offline notification skipped: muted",
			"node_sid", cmd.NodeSID,
			"node_name", cmd.NodeName,
		)
		return nil
	}

	bindings, err := s.bindingRepo.FindBindingsForNodeOfflineNotification(ctx)
	if err != nil {
		s.logger.Errorw("failed to find bindings for node offline notification", "error", err)
		return err
	}

	if len(bindings) == 0 {
		return nil
	}

	message := usecases.BuildNodeOfflineMessage(cmd.NodeSID, cmd.NodeName, cmd.LastSeenAt, cmd.OfflineMinutes)
	keyboard := buildMuteKeyboard("node", cmd.NodeSID)

	for i, binding := range bindings {
		if err := s.botService.SendMessageWithInlineKeyboard(binding.TelegramUserID(), message, keyboard); err != nil {
			s.logger.Errorw("failed to send node offline notification",
				"telegram_user_id", binding.TelegramUserID(),
				"error", err,
			)
			continue
		}
		// Rate limiting: add delay between messages to avoid Telegram API throttling
		if i < len(bindings)-1 {
			time.Sleep(messageSendDelay)
		}
	}

	return nil
}

// NotifyAgentOnline implements AdminNotifier interface
func (s *ServiceDDD) NotifyAgentOnline(ctx context.Context, cmd NotifyAgentOnlineCommand) error {
	if s.botService == nil {
		s.logger.Debugw("admin notification skipped: bot service not available", "type", "agent_online")
		return nil
	}

	// Skip if notification is muted for this agent
	if cmd.MuteNotification {
		s.logger.Debugw("agent online notification skipped: muted",
			"agent_sid", cmd.AgentSID,
			"agent_name", cmd.AgentName,
		)
		return nil
	}

	// Skip online notifications during startup cooldown period
	// This prevents batch notifications when agents reconnect after service restart
	if time.Since(s.startedAt) < startupCooldown {
		s.logger.Debugw("agent online notification skipped: startup cooldown",
			"agent_sid", cmd.AgentSID,
			"cooldown_remaining", startupCooldown-time.Since(s.startedAt),
		)
		return nil
	}

	// Use dedicated method for online notification (no deduplication threshold)
	bindings, err := s.bindingRepo.FindBindingsForAgentOnlineNotification(ctx)
	if err != nil {
		s.logger.Errorw("failed to find bindings for agent online notification", "error", err)
		return err
	}

	if len(bindings) == 0 {
		return nil
	}

	message := usecases.BuildAgentOnlineMessage(cmd.AgentSID, cmd.AgentName, biztime.NowUTC())
	keyboard := buildMuteKeyboard("agent", cmd.AgentSID)

	for i, binding := range bindings {
		if err := s.botService.SendMessageWithInlineKeyboard(binding.TelegramUserID(), message, keyboard); err != nil {
			s.logger.Errorw("failed to send agent online notification",
				"telegram_user_id", binding.TelegramUserID(),
				"error", err,
			)
			continue
		}
		// Rate limiting: add delay between messages to avoid Telegram API throttling
		if i < len(bindings)-1 {
			time.Sleep(messageSendDelay)
		}
	}

	return nil
}

// NotifyAgentOffline implements AdminNotifier interface
func (s *ServiceDDD) NotifyAgentOffline(ctx context.Context, cmd NotifyAgentOfflineCommand) error {
	if s.botService == nil {
		s.logger.Debugw("admin notification skipped: bot service not available", "type", "agent_offline")
		return nil
	}

	// Skip if notification is muted for this agent
	if cmd.MuteNotification {
		s.logger.Debugw("agent offline notification skipped: muted",
			"agent_sid", cmd.AgentSID,
			"agent_name", cmd.AgentName,
		)
		return nil
	}

	bindings, err := s.bindingRepo.FindBindingsForAgentOfflineNotification(ctx)
	if err != nil {
		s.logger.Errorw("failed to find bindings for agent offline notification", "error", err)
		return err
	}

	if len(bindings) == 0 {
		return nil
	}

	message := usecases.BuildAgentOfflineMessage(cmd.AgentSID, cmd.AgentName, cmd.LastSeenAt, cmd.OfflineMinutes)
	keyboard := buildMuteKeyboard("agent", cmd.AgentSID)

	for i, binding := range bindings {
		if err := s.botService.SendMessageWithInlineKeyboard(binding.TelegramUserID(), message, keyboard); err != nil {
			s.logger.Errorw("failed to send agent offline notification",
				"telegram_user_id", binding.TelegramUserID(),
				"error", err,
			)
			continue
		}
		// Rate limiting: add delay between messages to avoid Telegram API throttling
		if i < len(bindings)-1 {
			time.Sleep(messageSendDelay)
		}
	}

	return nil
}

// inlineKeyboardMarkup represents a Telegram inline keyboard
type inlineKeyboardMarkup struct {
	InlineKeyboard [][]inlineKeyboardButton `json:"inline_keyboard"`
}

// inlineKeyboardButton represents a button in an inline keyboard
type inlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

// buildMuteKeyboard builds an inline keyboard with mute button
// resourceType is "node" or "agent", resourceSID is the SID of the resource
func buildMuteKeyboard(resourceType, resourceSID string) *inlineKeyboardMarkup {
	callbackData := fmt.Sprintf("mute:%s:%s", resourceType, resourceSID)
	return &inlineKeyboardMarkup{
		InlineKeyboard: [][]inlineKeyboardButton{
			{
				{
					Text:         "🔕 静默此通知 / Mute",
					CallbackData: callbackData,
				},
			},
		},
	}
}
