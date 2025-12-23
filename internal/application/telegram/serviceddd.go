package telegram

import (
	"context"
	"errors"

	"github.com/orris-inc/orris/internal/application/telegram/dto"
	"github.com/orris-inc/orris/internal/application/telegram/usecases"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/telegram"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// VerifyCodeStore combines generator and verifier interfaces
type VerifyCodeStore interface {
	usecases.VerifyCodeGenerator
	usecases.VerifyCodeVerifier
}

// BotService defines the telegram bot service interface
type BotService interface {
	usecases.TelegramMessageSender
	SendMessage(chatID int64, text string) error
}

// ServiceDDD aggregates all telegram-related use cases
type ServiceDDD struct {
	getStatusUC         *usecases.GetBindingStatusUseCase
	bindUC              *usecases.BindTelegramUseCase
	unbindUC            *usecases.UnbindTelegramUseCase
	updatePreferencesUC *usecases.UpdatePreferencesUseCase
	processReminderUC   *usecases.ProcessReminderUseCase
	botService          BotService
	bindingRepo         telegram.TelegramBindingRepository
	logger              logger.Interface
}

// NewServiceDDD creates a new telegram service
func NewServiceDDD(
	bindingRepo telegram.TelegramBindingRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	usageRepo subscription.SubscriptionUsageRepository,
	planRepo subscription.PlanRepository,
	verifyCodeStore VerifyCodeStore,
	botService BotService,
	logger logger.Interface,
) *ServiceDDD {
	return &ServiceDDD{
		getStatusUC:         usecases.NewGetBindingStatusUseCase(bindingRepo, verifyCodeStore, logger),
		bindUC:              usecases.NewBindTelegramUseCase(bindingRepo, verifyCodeStore, logger),
		unbindUC:            usecases.NewUnbindTelegramUseCase(bindingRepo, logger),
		updatePreferencesUC: usecases.NewUpdatePreferencesUseCase(bindingRepo, logger),
		processReminderUC: usecases.NewProcessReminderUseCase(
			bindingRepo, subscriptionRepo, usageRepo, planRepo, botService, logger,
		),
		botService:  botService,
		bindingRepo: bindingRepo,
		logger:      logger,
	}
}

// GetBindingStatus retrieves the telegram binding status for a user
func (s *ServiceDDD) GetBindingStatus(ctx context.Context, userID uint) (*dto.BindingStatusResponse, error) {
	return s.getStatusUC.Execute(ctx, userID)
}

// BindFromWebhook binds a telegram account via webhook (verify code)
func (s *ServiceDDD) BindFromWebhook(
	ctx context.Context,
	telegramUserID int64,
	telegramUsername string,
	verifyCode string,
) (*dto.TelegramBindingResponse, error) {
	return s.bindUC.Execute(ctx, telegramUserID, telegramUsername, verifyCode)
}

// Unbind removes the telegram binding for a user
func (s *ServiceDDD) Unbind(ctx context.Context, userID uint) error {
	return s.unbindUC.ExecuteByUserID(ctx, userID)
}

// UnbindByTelegramID removes the telegram binding by Telegram user ID
func (s *ServiceDDD) UnbindByTelegramID(ctx context.Context, telegramUserID int64) error {
	return s.unbindUC.ExecuteByTelegramID(ctx, telegramUserID)
}

// UpdatePreferences updates notification preferences
func (s *ServiceDDD) UpdatePreferences(
	ctx context.Context,
	userID uint,
	req dto.UpdatePreferencesRequest,
) (*dto.TelegramBindingResponse, error) {
	return s.updatePreferencesUC.Execute(ctx, userID, req)
}

// GetBindingStatusByTelegramID retrieves binding status by Telegram user ID
func (s *ServiceDDD) GetBindingStatusByTelegramID(ctx context.Context, telegramUserID int64) (*dto.BindingStatusResponse, error) {
	binding, err := s.bindingRepo.GetByTelegramUserID(ctx, telegramUserID)
	if err != nil {
		// Only treat "not found" as unbound, propagate other errors
		if errors.Is(err, telegram.ErrBindingNotFound) {
			return &dto.BindingStatusResponse{IsBound: false}, nil
		}
		s.logger.Errorw("failed to get binding by telegram user ID",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return nil, err
	}
	if binding == nil {
		return &dto.BindingStatusResponse{IsBound: false}, nil
	}

	return &dto.BindingStatusResponse{
		IsBound: true,
		Binding: &dto.TelegramBindingResponse{
			SID:              binding.SID(),
			TelegramUserID:   binding.TelegramUserID(),
			TelegramUsername: binding.TelegramUsername(),
			NotifyExpiring:   binding.NotifyExpiring(),
			NotifyTraffic:    binding.NotifyTraffic(),
			ExpiringDays:     binding.ExpiringDays(),
			TrafficThreshold: binding.TrafficThreshold(),
			CreatedAt:        binding.CreatedAt(),
		},
	}, nil
}

// SendBotMessage sends a message via the telegram bot
func (s *ServiceDDD) SendBotMessage(chatID int64, text string) error {
	return s.botService.SendMessage(chatID, text)
}

// GetProcessReminderUseCase returns the reminder processor for scheduler use
func (s *ServiceDDD) GetProcessReminderUseCase() *usecases.ProcessReminderUseCase {
	return s.processReminderUC
}
