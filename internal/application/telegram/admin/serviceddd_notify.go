package admin

import (
	"context"
	"time"

	telegram "github.com/orris-inc/orris/internal/infrastructure/telegram"
	"github.com/orris-inc/orris/internal/infrastructure/telegram/i18n"
	"github.com/orris-inc/orris/internal/shared/biztime"
)

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
