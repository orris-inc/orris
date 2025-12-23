package usecases

import (
	"context"
	"errors"

	"github.com/orris-inc/orris/internal/application/telegram/dto"
	"github.com/orris-inc/orris/internal/domain/telegram"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// VerifyCodeGenerator generates verification codes
type VerifyCodeGenerator interface {
	Generate(ctx context.Context, userID uint) (string, error)
}

// BotLinkProvider provides the Telegram bot link
type BotLinkProvider interface {
	GetBotLink() string
}

// GetBindingStatusUseCase retrieves the telegram binding status for a user
type GetBindingStatusUseCase struct {
	bindingRepo     telegram.TelegramBindingRepository
	verifyCodeGen   VerifyCodeGenerator
	botLinkProvider BotLinkProvider
	logger          logger.Interface
}

// NewGetBindingStatusUseCase creates a new GetBindingStatusUseCase
func NewGetBindingStatusUseCase(
	bindingRepo telegram.TelegramBindingRepository,
	verifyCodeGen VerifyCodeGenerator,
	botLinkProvider BotLinkProvider,
	logger logger.Interface,
) *GetBindingStatusUseCase {
	return &GetBindingStatusUseCase{
		bindingRepo:     bindingRepo,
		verifyCodeGen:   verifyCodeGen,
		botLinkProvider: botLinkProvider,
		logger:          logger,
	}
}

// Execute retrieves the binding status
func (uc *GetBindingStatusUseCase) Execute(ctx context.Context, userID uint) (*dto.BindingStatusResponse, error) {
	botLink := ""
	if uc.botLinkProvider != nil {
		botLink = uc.botLinkProvider.GetBotLink()
	}

	binding, err := uc.bindingRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, telegram.ErrBindingNotFound) {
			// User not bound, generate verify code
			code, err := uc.verifyCodeGen.Generate(ctx, userID)
			if err != nil {
				uc.logger.Errorw("failed to generate verify code", "user_id", userID, "error", err)
				return nil, err
			}
			return &dto.BindingStatusResponse{
				IsBound:    false,
				VerifyCode: code,
				BotLink:    botLink,
			}, nil
		}
		return nil, err
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
		BotLink: botLink,
	}, nil
}
