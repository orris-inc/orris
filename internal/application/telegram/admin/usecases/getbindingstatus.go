package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// verifyCodeTTL is the TTL for admin verify codes
	verifyCodeTTL = 10 * time.Minute
)

// AdminVerifyCodeGenerator generates verification codes for admin binding
type AdminVerifyCodeGenerator interface {
	Generate(ctx context.Context, userID uint) (string, error)
}

// AdminBotLinkProvider provides the Telegram bot link for admin notifications
type AdminBotLinkProvider interface {
	GetBotLink() string
}

// GetAdminBindingStatusUseCase retrieves the admin telegram binding status
type GetAdminBindingStatusUseCase struct {
	bindingRepo     AdminTelegramBindingRepository
	verifyCodeGen   AdminVerifyCodeGenerator
	botLinkProvider AdminBotLinkProvider
	logger          logger.Interface
}

// NewGetAdminBindingStatusUseCase creates a new GetAdminBindingStatusUseCase
func NewGetAdminBindingStatusUseCase(
	bindingRepo AdminTelegramBindingRepository,
	verifyCodeGen AdminVerifyCodeGenerator,
	botLinkProvider AdminBotLinkProvider,
	logger logger.Interface,
) *GetAdminBindingStatusUseCase {
	return &GetAdminBindingStatusUseCase{
		bindingRepo:     bindingRepo,
		verifyCodeGen:   verifyCodeGen,
		botLinkProvider: botLinkProvider,
		logger:          logger,
	}
}

// Execute retrieves the admin binding status
func (uc *GetAdminBindingStatusUseCase) Execute(ctx context.Context, userID uint) (*dto.AdminBindingStatusResponse, error) {
	botLink := ""
	if uc.botLinkProvider != nil {
		botLink = uc.botLinkProvider.GetBotLink()
	}

	binding, err := uc.bindingRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, admin.ErrBindingNotFound) {
			// Admin not bound, generate verify code
			code, err := uc.verifyCodeGen.Generate(ctx, userID)
			if err != nil {
				uc.logger.Errorw("failed to generate verify code for admin", "user_id", userID, "error", err)
				return nil, err
			}
			expiresAt := biztime.NowUTC().Add(verifyCodeTTL)
			return &dto.AdminBindingStatusResponse{
				IsBound:    false,
				VerifyCode: code,
				BotLink:    botLink,
				ExpiresAt:  &expiresAt,
			}, nil
		}
		return nil, err
	}

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
