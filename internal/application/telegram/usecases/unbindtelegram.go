package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/telegram"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UnbindTelegramUseCase handles telegram unbinding
type UnbindTelegramUseCase struct {
	bindingRepo telegram.TelegramBindingRepository
	logger      logger.Interface
}

// NewUnbindTelegramUseCase creates a new UnbindTelegramUseCase
func NewUnbindTelegramUseCase(
	bindingRepo telegram.TelegramBindingRepository,
	logger logger.Interface,
) *UnbindTelegramUseCase {
	return &UnbindTelegramUseCase{
		bindingRepo: bindingRepo,
		logger:      logger,
	}
}

// ExecuteByUserID unbinds telegram by user ID
func (uc *UnbindTelegramUseCase) ExecuteByUserID(ctx context.Context, userID uint) error {
	binding, err := uc.bindingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if err := uc.bindingRepo.Delete(ctx, binding.ID()); err != nil {
		return err
	}

	uc.logger.Infow("telegram binding removed", "user_id", userID)
	return nil
}

// ExecuteByTelegramID unbinds telegram by Telegram user ID
func (uc *UnbindTelegramUseCase) ExecuteByTelegramID(ctx context.Context, telegramUserID int64) error {
	binding, err := uc.bindingRepo.GetByTelegramUserID(ctx, telegramUserID)
	if err != nil {
		return err
	}

	if err := uc.bindingRepo.Delete(ctx, binding.ID()); err != nil {
		return err
	}

	uc.logger.Infow("telegram binding removed", "telegram_user_id", telegramUserID)
	return nil
}
