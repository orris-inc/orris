package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
	telegramAdmin "github.com/orris-inc/orris/internal/domain/telegram/admin"
	telegram "github.com/orris-inc/orris/internal/infrastructure/telegram"
	"github.com/orris-inc/orris/internal/infrastructure/telegram/i18n"
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
	SendChatAction(chatID int64, action string) error
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
	if err != nil && !errors.Is(err, telegramAdmin.ErrBindingNotFound) {
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
				SID:                         binding.SID(),
				TelegramUserID:              binding.TelegramUserID(),
				TelegramUsername:            binding.TelegramUsername(),
				NotifyNodeOffline:           binding.NotifyNodeOffline(),
				NotifyAgentOffline:          binding.NotifyAgentOffline(),
				NotifyNewUser:               binding.NotifyNewUser(),
				NotifyPaymentSuccess:        binding.NotifyPaymentSuccess(),
				NotifyDailySummary:          binding.NotifyDailySummary(),
				NotifyWeeklySummary:         binding.NotifyWeeklySummary(),
				OfflineThresholdMinutes:     binding.OfflineThresholdMinutes(),
				NotifyResourceExpiring:      binding.NotifyResourceExpiring(),
				ResourceExpiringDays:        binding.ResourceExpiringDays(),
				DailySummaryHour:            binding.DailySummaryHour(),
				WeeklySummaryHour:           binding.WeeklySummaryHour(),
				WeeklySummaryWeekday:        binding.WeeklySummaryWeekday(),
				OfflineCheckIntervalMinutes: binding.OfflineCheckIntervalMinutes(),
				CreatedAt:                   binding.CreatedAt(),
			},
			BotLink: botLink,
		}, nil
	}

	// Generate verify code for unbound user
	verifyCode, err := s.verifyStore.Generate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verify code: %w", err)
	}

	expiresAt := biztime.NowUTC().Add(10 * time.Minute)
	resp := &dto.AdminBindingStatusResponse{
		IsBound:    false,
		VerifyCode: verifyCode,
		BotLink:    botLink,
		ExpiresAt:  &expiresAt,
	}
	if botLink != "" {
		resp.DeepBindLink = fmt.Sprintf("%s?start=adminbind_%s", botLink, verifyCode)
	}
	return resp, nil
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
	if err != nil && !errors.Is(err, telegramAdmin.ErrBindingNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, telegramAdmin.ErrAlreadyBound
	}

	// Check if telegram account is already used
	existingTg, err := s.bindingRepo.GetByTelegramUserID(ctx, telegramUserID)
	if err != nil && !errors.Is(err, telegramAdmin.ErrBindingNotFound) {
		return nil, err
	}
	if existingTg != nil {
		return nil, telegramAdmin.ErrTelegramAlreadyUsed
	}

	// Create binding
	binding, err := telegramAdmin.NewAdminTelegramBinding(userID, telegramUserID, telegramUsername, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create binding: %w", err)
	}

	if err := s.bindingRepo.Create(ctx, binding); err != nil {
		return nil, fmt.Errorf("failed to save binding: %w", err)
	}

	// Delete the verify code (cleanup)
	_ = s.verifyStore.Delete(ctx, verifyCode)

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
		req.NotifyResourceExpiring,
		req.ResourceExpiringDays,
		req.DailySummaryHour,
		req.WeeklySummaryHour,
		req.WeeklySummaryWeekday,
		req.OfflineCheckIntervalMinutes,
	); err != nil {
		return nil, err
	}

	if err := s.bindingRepo.Update(ctx, binding); err != nil {
		return nil, fmt.Errorf("failed to update binding: %w", err)
	}

	return &dto.AdminTelegramBindingResponse{
		SID:                         binding.SID(),
		TelegramUserID:              binding.TelegramUserID(),
		TelegramUsername:            binding.TelegramUsername(),
		NotifyNodeOffline:           binding.NotifyNodeOffline(),
		NotifyAgentOffline:          binding.NotifyAgentOffline(),
		NotifyNewUser:               binding.NotifyNewUser(),
		NotifyPaymentSuccess:        binding.NotifyPaymentSuccess(),
		NotifyDailySummary:          binding.NotifyDailySummary(),
		NotifyWeeklySummary:         binding.NotifyWeeklySummary(),
		OfflineThresholdMinutes:     binding.OfflineThresholdMinutes(),
		NotifyResourceExpiring:      binding.NotifyResourceExpiring(),
		ResourceExpiringDays:        binding.ResourceExpiringDays(),
		DailySummaryHour:            binding.DailySummaryHour(),
		WeeklySummaryHour:           binding.WeeklySummaryHour(),
		WeeklySummaryWeekday:        binding.WeeklySummaryWeekday(),
		OfflineCheckIntervalMinutes: binding.OfflineCheckIntervalMinutes(),
		CreatedAt:                   binding.CreatedAt(),
	}, nil
}

// GetBindingByTelegramID gets binding by telegram user ID
func (s *ServiceDDD) GetBindingByTelegramID(ctx context.Context, telegramUserID int64) (any, error) {
	return s.bindingRepo.GetByTelegramUserID(ctx, telegramUserID)
}

// UpdateAdminBindingLanguage updates the language preference for an admin binding
func (s *ServiceDDD) UpdateAdminBindingLanguage(ctx context.Context, telegramUserID int64, language string) error {
	binding, err := s.bindingRepo.GetByTelegramUserID(ctx, telegramUserID)
	if err != nil {
		if errors.Is(err, telegramAdmin.ErrBindingNotFound) {
			return nil
		}
		return err
	}
	binding.UpdateLanguage(language)
	return s.bindingRepo.Update(ctx, binding)
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

	for i, binding := range bindings {
		lang := i18n.ParseLang(binding.Language())
		message := i18n.BuildNewUserMessage(lang, cmd.UserSID, cmd.Email, cmd.Name, cmd.Source, cmd.CreatedAt)
		if err := s.botService.SendMessage(binding.TelegramUserID(), message); err != nil {
			if telegram.IsBotBlocked(err) {
				s.logger.Warnw("bot blocked by user, skipping notification",
					"telegram_user_id", binding.TelegramUserID())
				continue
			}
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

	for i, binding := range bindings {
		lang := i18n.ParseLang(binding.Language())
		message := i18n.BuildPaymentSuccessMessage(
			lang,
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
		if err := s.botService.SendMessage(binding.TelegramUserID(), message); err != nil {
			if telegram.IsBotBlocked(err) {
				s.logger.Warnw("bot blocked by user, skipping notification",
					"telegram_user_id", binding.TelegramUserID())
				continue
			}
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

	for i, binding := range bindings {
		lang := i18n.ParseLang(binding.Language())
		message := i18n.BuildNodeOnlineMessage(lang, cmd.NodeSID, cmd.NodeName, biztime.NowUTC())
		keyboard := i18n.BuildMuteKeyboard(lang, "node", cmd.NodeSID)
		if err := s.botService.SendMessageWithInlineKeyboard(binding.TelegramUserID(), message, keyboard); err != nil {
			if telegram.IsBotBlocked(err) {
				s.logger.Warnw("bot blocked by user, skipping notification",
					"telegram_user_id", binding.TelegramUserID())
				continue
			}
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

	for i, binding := range bindings {
		lang := i18n.ParseLang(binding.Language())
		message := i18n.BuildNodeOfflineMessage(lang, cmd.NodeSID, cmd.NodeName, cmd.LastSeenAt, cmd.OfflineMinutes)
		keyboard := i18n.BuildMuteKeyboard(lang, "node", cmd.NodeSID)
		if err := s.botService.SendMessageWithInlineKeyboard(binding.TelegramUserID(), message, keyboard); err != nil {
			if telegram.IsBotBlocked(err) {
				s.logger.Warnw("bot blocked by user, skipping notification",
					"telegram_user_id", binding.TelegramUserID())
				continue
			}
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

	for i, binding := range bindings {
		lang := i18n.ParseLang(binding.Language())
		message := i18n.BuildAgentOnlineMessage(lang, cmd.AgentSID, cmd.AgentName, biztime.NowUTC())
		keyboard := i18n.BuildMuteKeyboard(lang, "agent", cmd.AgentSID)
		if err := s.botService.SendMessageWithInlineKeyboard(binding.TelegramUserID(), message, keyboard); err != nil {
			if telegram.IsBotBlocked(err) {
				s.logger.Warnw("bot blocked by user, skipping notification",
					"telegram_user_id", binding.TelegramUserID())
				continue
			}
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

	for i, binding := range bindings {
		lang := i18n.ParseLang(binding.Language())
		message := i18n.BuildAgentOfflineMessage(lang, cmd.AgentSID, cmd.AgentName, cmd.LastSeenAt, cmd.OfflineMinutes)
		keyboard := i18n.BuildMuteKeyboard(lang, "agent", cmd.AgentSID)
		if err := s.botService.SendMessageWithInlineKeyboard(binding.TelegramUserID(), message, keyboard); err != nil {
			if telegram.IsBotBlocked(err) {
				s.logger.Warnw("bot blocked by user, skipping notification",
					"telegram_user_id", binding.TelegramUserID())
				continue
			}
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

// NotifyNodeRecovery implements AdminNotifier interface
// This is called when a node transitions from Firing state back to Normal
func (s *ServiceDDD) NotifyNodeRecovery(ctx context.Context, cmd NotifyNodeRecoveryCommand) error {
	if s.botService == nil {
		s.logger.Debugw("admin notification skipped: bot service not available", "type", "node_recovery")
		return nil
	}

	// Skip if notification is muted for this node
	if cmd.MuteNotification {
		s.logger.Debugw("node recovery notification skipped: muted",
			"node_sid", cmd.NodeSID,
			"node_name", cmd.NodeName,
		)
		return nil
	}

	// Use the same bindings as offline notification (recovery is the counterpart)
	bindings, err := s.bindingRepo.FindBindingsForNodeOfflineNotification(ctx)
	if err != nil {
		s.logger.Errorw("failed to find bindings for node recovery notification", "error", err)
		return err
	}

	if len(bindings) == 0 {
		return nil
	}

	for i, binding := range bindings {
		lang := i18n.ParseLang(binding.Language())
		message := i18n.BuildNodeRecoveryMessage(lang, cmd.NodeSID, cmd.NodeName, cmd.OnlineAt, cmd.DowntimeMinutes)
		if err := s.botService.SendMessage(binding.TelegramUserID(), message); err != nil {
			if telegram.IsBotBlocked(err) {
				s.logger.Warnw("bot blocked by user, skipping notification",
					"telegram_user_id", binding.TelegramUserID())
				continue
			}
			s.logger.Errorw("failed to send node recovery notification",
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

// NotifyAgentRecovery implements AdminNotifier interface
// This is called when an agent transitions from Firing state back to Normal
func (s *ServiceDDD) NotifyAgentRecovery(ctx context.Context, cmd NotifyAgentRecoveryCommand) error {
	if s.botService == nil {
		s.logger.Debugw("admin notification skipped: bot service not available", "type", "agent_recovery")
		return nil
	}

	// Skip if notification is muted for this agent
	if cmd.MuteNotification {
		s.logger.Debugw("agent recovery notification skipped: muted",
			"agent_sid", cmd.AgentSID,
			"agent_name", cmd.AgentName,
		)
		return nil
	}

	// Use the same bindings as offline notification (recovery is the counterpart)
	bindings, err := s.bindingRepo.FindBindingsForAgentOfflineNotification(ctx)
	if err != nil {
		s.logger.Errorw("failed to find bindings for agent recovery notification", "error", err)
		return err
	}

	if len(bindings) == 0 {
		return nil
	}

	for i, binding := range bindings {
		lang := i18n.ParseLang(binding.Language())
		message := i18n.BuildAgentRecoveryMessage(lang, cmd.AgentSID, cmd.AgentName, cmd.OnlineAt, cmd.DowntimeMinutes)
		if err := s.botService.SendMessage(binding.TelegramUserID(), message); err != nil {
			if telegram.IsBotBlocked(err) {
				s.logger.Warnw("bot blocked by user, skipping notification",
					"telegram_user_id", binding.TelegramUserID())
				continue
			}
			s.logger.Errorw("failed to send agent recovery notification",
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
