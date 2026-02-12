package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
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
