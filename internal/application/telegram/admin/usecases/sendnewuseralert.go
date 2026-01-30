package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SendNewUserAlertUseCase handles sending new user registration alerts to admins
type SendNewUserAlertUseCase struct {
	bindingRepo admin.AdminTelegramBindingRepository
	botService  TelegramMessageSender
	logger      logger.Interface
}

// NewSendNewUserAlertUseCase creates a new SendNewUserAlertUseCase
func NewSendNewUserAlertUseCase(
	bindingRepo admin.AdminTelegramBindingRepository,
	botService TelegramMessageSender,
	logger logger.Interface,
) *SendNewUserAlertUseCase {
	return &SendNewUserAlertUseCase{
		bindingRepo: bindingRepo,
		botService:  botService,
		logger:      logger,
	}
}

// SendAlert sends new user registration alert to all subscribed admins
func (uc *SendNewUserAlertUseCase) SendAlert(ctx context.Context, newUser *user.User) error {
	if uc.botService == nil {
		uc.logger.Debugw("new user alert skipped: bot service not available")
		return nil
	}

	// Get bindings that want new user notifications
	bindings, err := uc.bindingRepo.FindBindingsForNewUserNotification(ctx)
	if err != nil {
		uc.logger.Errorw("failed to find bindings for new user notification", "error", err)
		return fmt.Errorf("failed to find bindings: %w", err)
	}

	if len(bindings) == 0 {
		uc.logger.Debugw("no bindings configured for new user notifications")
		return nil
	}

	// Build user info
	userInfo := dto.NewUserInfo{
		SID:       newUser.SID(),
		Email:     newUser.Email().String(),
		Name:      newUser.Name().DisplayName(),
		Source:    "registration",
		CreatedAt: newUser.CreatedAt(),
	}

	message := BuildNewUserMessage(userInfo.SID, userInfo.Email, userInfo.Name, userInfo.Source, userInfo.CreatedAt)

	sentCount := 0
	errorCount := 0

	for _, binding := range bindings {
		if !binding.NotifyNewUser() {
			continue
		}

		if err := uc.botService.SendMessage(binding.TelegramUserID(), message); err != nil {
			uc.logger.Errorw("failed to send new user notification",
				"telegram_user_id", binding.TelegramUserID(),
				"user_sid", userInfo.SID,
				"error", err,
			)
			errorCount++
			continue
		}
		sentCount++
	}

	uc.logger.Infow("new user alert sent",
		"user_sid", userInfo.SID,
		"sent_count", sentCount,
		"error_count", errorCount,
	)

	return nil
}

// SendAlertWithInfo sends new user registration alert using pre-built user info
func (uc *SendNewUserAlertUseCase) SendAlertWithInfo(ctx context.Context, userInfo dto.NewUserInfo) error {
	if uc.botService == nil {
		uc.logger.Debugw("new user alert skipped: bot service not available")
		return nil
	}

	// Get bindings that want new user notifications
	bindings, err := uc.bindingRepo.FindBindingsForNewUserNotification(ctx)
	if err != nil {
		uc.logger.Errorw("failed to find bindings for new user notification", "error", err)
		return fmt.Errorf("failed to find bindings: %w", err)
	}

	if len(bindings) == 0 {
		uc.logger.Debugw("no bindings configured for new user notifications")
		return nil
	}

	message := BuildNewUserMessage(userInfo.SID, userInfo.Email, userInfo.Name, userInfo.Source, userInfo.CreatedAt)

	sentCount := 0
	errorCount := 0

	for _, binding := range bindings {
		if !binding.NotifyNewUser() {
			continue
		}

		if err := uc.botService.SendMessage(binding.TelegramUserID(), message); err != nil {
			uc.logger.Errorw("failed to send new user notification",
				"telegram_user_id", binding.TelegramUserID(),
				"user_sid", userInfo.SID,
				"error", err,
			)
			errorCount++
			continue
		}
		sentCount++
	}

	uc.logger.Infow("new user alert sent",
		"user_sid", userInfo.SID,
		"sent_count", sentCount,
		"error_count", errorCount,
	)

	return nil
}
