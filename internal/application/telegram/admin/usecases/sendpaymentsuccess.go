package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SendPaymentSuccessUseCase handles sending payment success alerts to admins
type SendPaymentSuccessUseCase struct {
	bindingRepo admin.AdminTelegramBindingRepository
	botService  TelegramMessageSender
	logger      logger.Interface
}

// NewSendPaymentSuccessUseCase creates a new SendPaymentSuccessUseCase
func NewSendPaymentSuccessUseCase(
	bindingRepo admin.AdminTelegramBindingRepository,
	botService TelegramMessageSender,
	logger logger.Interface,
) *SendPaymentSuccessUseCase {
	return &SendPaymentSuccessUseCase{
		bindingRepo: bindingRepo,
		botService:  botService,
		logger:      logger,
	}
}

// SendAlert sends payment success alert to all subscribed admins
func (uc *SendPaymentSuccessUseCase) SendAlert(ctx context.Context, payment dto.PaymentInfo) error {
	if uc.botService == nil {
		uc.logger.Debugw("payment success alert skipped: bot service not available")
		return nil
	}

	// Get bindings that want payment success notifications
	bindings, err := uc.bindingRepo.FindBindingsForPaymentSuccessNotification(ctx)
	if err != nil {
		uc.logger.Errorw("failed to find bindings for payment success notification", "error", err)
		return fmt.Errorf("failed to find bindings: %w", err)
	}

	if len(bindings) == 0 {
		uc.logger.Debugw("no bindings configured for payment success notifications")
		return nil
	}

	message := BuildPaymentSuccessMessage(
		payment.PaymentSID,
		payment.UserSID,
		payment.UserEmail,
		payment.PlanName,
		payment.Amount,
		payment.Currency,
		payment.PaymentMethod,
		payment.TransactionID,
		payment.PaidAt,
	)

	sentCount := 0
	errorCount := 0

	for _, binding := range bindings {
		if !binding.NotifyPaymentSuccess() {
			continue
		}

		if err := uc.botService.SendMessage(binding.TelegramUserID(), message); err != nil {
			uc.logger.Errorw("failed to send payment success notification",
				"telegram_user_id", binding.TelegramUserID(),
				"payment_sid", payment.PaymentSID,
				"error", err,
			)
			errorCount++
			continue
		}
		sentCount++
	}

	uc.logger.Infow("payment success alert sent",
		"payment_sid", payment.PaymentSID,
		"amount", payment.Amount,
		"currency", payment.Currency,
		"sent_count", sentCount,
		"error_count", errorCount,
	)

	return nil
}
