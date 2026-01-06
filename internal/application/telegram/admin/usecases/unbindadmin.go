package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UnbindAdminUseCase handles admin telegram unbinding
type UnbindAdminUseCase struct {
	bindingRepo AdminTelegramBindingRepository
	logger      logger.Interface
}

// NewUnbindAdminUseCase creates a new UnbindAdminUseCase
func NewUnbindAdminUseCase(
	bindingRepo AdminTelegramBindingRepository,
	logger logger.Interface,
) *UnbindAdminUseCase {
	return &UnbindAdminUseCase{
		bindingRepo: bindingRepo,
		logger:      logger,
	}
}

// ExecuteByUserID unbinds admin telegram by user ID (called from web interface)
func (uc *UnbindAdminUseCase) ExecuteByUserID(ctx context.Context, userID uint) error {
	binding, err := uc.bindingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if binding == nil {
		return admin.ErrBindingNotFound
	}

	if err := uc.bindingRepo.Delete(ctx, binding.ID()); err != nil {
		return err
	}

	uc.logger.Infow("admin telegram binding removed by user", "user_id", userID)
	return nil
}

// ExecuteByTelegramID unbinds admin telegram by Telegram user ID (called from Telegram command)
func (uc *UnbindAdminUseCase) ExecuteByTelegramID(ctx context.Context, telegramUserID int64) error {
	binding, err := uc.bindingRepo.GetByTelegramUserID(ctx, telegramUserID)
	if err != nil {
		return err
	}
	if binding == nil {
		return admin.ErrBindingNotFound
	}

	if err := uc.bindingRepo.Delete(ctx, binding.ID()); err != nil {
		return err
	}

	uc.logger.Infow("admin telegram binding removed by telegram", "telegram_user_id", telegramUserID)
	return nil
}
