package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/biztime"
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

// NewUserInfo contains information about a new user for alert
type NewUserInfo struct {
	SID       string
	Email     string
	Name      string
	CreatedAt string
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
	userInfo := NewUserInfo{
		SID:       newUser.SID(),
		Email:     newUser.Email().String(),
		Name:      newUser.Name().DisplayName(),
		CreatedAt: biztime.FormatInBizTimezone(newUser.CreatedAt(), "2006-01-02 15:04:05"),
	}

	message := uc.buildNewUserMessage(userInfo)

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
func (uc *SendNewUserAlertUseCase) SendAlertWithInfo(ctx context.Context, userInfo NewUserInfo) error {
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

	message := uc.buildNewUserMessage(userInfo)

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

func (uc *SendNewUserAlertUseCase) buildNewUserMessage(userInfo NewUserInfo) string {
	return fmt.Sprintf(`🆕 <b>New User Registration / 新用户注册</b>

User 用户: <code>%s</code>
Email 邮箱: <code>%s</code>
Name 名称: %s
Registered at 注册时间: %s

🎉 Welcome new user!
欢迎新用户！`, userInfo.SID, escapeHTML(userInfo.Email), escapeHTML(userInfo.Name), userInfo.CreatedAt)
}
