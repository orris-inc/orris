package usecases

import (
	"context"
	"fmt"

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

// PaymentInfo contains information about a successful payment
type PaymentInfo struct {
	OrderSID       string  // Order/Payment SID
	UserSID        string  // User SID
	UserEmail      string  // User email
	PlanName       string  // Subscription plan name
	Amount         float64 // Payment amount
	Currency       string  // Currency code (e.g., "USD", "CNY")
	PaymentMethod  string  // Payment method (e.g., "stripe", "alipay")
	SubscriptionID string  // Subscription SID
	PaidAt         string  // Payment timestamp formatted
}

// SendAlert sends payment success alert to all subscribed admins
func (uc *SendPaymentSuccessUseCase) SendAlert(ctx context.Context, payment PaymentInfo) error {
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

	message := uc.buildPaymentSuccessMessage(payment)

	sentCount := 0
	errorCount := 0

	for _, binding := range bindings {
		if !binding.NotifyPaymentSuccess() {
			continue
		}

		if err := uc.botService.SendMessage(binding.TelegramUserID(), message); err != nil {
			uc.logger.Errorw("failed to send payment success notification",
				"telegram_user_id", binding.TelegramUserID(),
				"order_sid", payment.OrderSID,
				"error", err,
			)
			errorCount++
			continue
		}
		sentCount++
	}

	uc.logger.Infow("payment success alert sent",
		"order_sid", payment.OrderSID,
		"amount", payment.Amount,
		"currency", payment.Currency,
		"sent_count", sentCount,
		"error_count", errorCount,
	)

	return nil
}

func (uc *SendPaymentSuccessUseCase) buildPaymentSuccessMessage(payment PaymentInfo) string {
	// Format amount based on currency
	amountStr := formatAmount(payment.Amount, payment.Currency)

	return fmt.Sprintf(`💰 <b>Payment Success / 支付成功</b>

Order 订单: <code>%s</code>
User 用户: <code>%s</code>
Email 邮箱: <code>%s</code>

Plan 套餐: <b>%s</b>
Amount 金额: <b>%s</b>
Method 方式: %s
Subscription 订阅: <code>%s</code>
Time 时间: %s

🎉 New subscription activated!
新订阅已激活！`, payment.OrderSID, payment.UserSID, escapeHTML(payment.UserEmail), escapeHTML(payment.PlanName),
		amountStr, escapeHTML(payment.PaymentMethod), payment.SubscriptionID, payment.PaidAt)
}
