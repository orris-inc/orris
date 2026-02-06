package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/orris-inc/orris/internal/application/telegram/dto"
	"github.com/orris-inc/orris/internal/domain/telegram"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// VerifyCodeVerifier verifies codes and returns user ID
type VerifyCodeVerifier interface {
	Verify(ctx context.Context, code string) (uint, error)
	Delete(ctx context.Context, code string) error
}

// BindTelegramUseCase handles telegram binding via verify code
type BindTelegramUseCase struct {
	bindingRepo telegram.TelegramBindingRepository
	verifyCode  VerifyCodeVerifier
	logger      logger.Interface
}

// NewBindTelegramUseCase creates a new BindTelegramUseCase
func NewBindTelegramUseCase(
	bindingRepo telegram.TelegramBindingRepository,
	verifyCode VerifyCodeVerifier,
	logger logger.Interface,
) *BindTelegramUseCase {
	return &BindTelegramUseCase{
		bindingRepo: bindingRepo,
		verifyCode:  verifyCode,
		logger:      logger,
	}
}

// Execute binds a telegram account via verify code
func (uc *BindTelegramUseCase) Execute(
	ctx context.Context,
	telegramUserID int64,
	telegramUsername string,
	verifyCode string,
) (*dto.TelegramBindingResponse, error) {
	// Verify the code and get userID
	userID, err := uc.verifyCode.Verify(ctx, verifyCode)
	if err != nil {
		uc.logger.Warnw("invalid verify code", "code", verifyCode, "error", err)
		return nil, telegram.ErrInvalidVerifyCode
	}

	// Check if user already has binding
	existing, err := uc.bindingRepo.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, telegram.ErrBindingNotFound) {
		return nil, fmt.Errorf("failed to check existing binding: %w", err)
	}
	if existing != nil {
		return nil, telegram.ErrAlreadyBound
	}

	// Check if telegram account is already used
	existingTg, err := uc.bindingRepo.GetByTelegramUserID(ctx, telegramUserID)
	if err != nil && !errors.Is(err, telegram.ErrBindingNotFound) {
		return nil, fmt.Errorf("failed to check telegram binding: %w", err)
	}
	if existingTg != nil {
		return nil, telegram.ErrTelegramAlreadyUsed
	}

	// Create binding
	binding, err := telegram.NewTelegramBinding(userID, telegramUserID, telegramUsername, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create binding: %w", err)
	}

	if err := uc.bindingRepo.Create(ctx, binding); err != nil {
		return nil, fmt.Errorf("failed to save binding: %w", err)
	}

	// Delete the verify code (cleanup)
	_ = uc.verifyCode.Delete(ctx, verifyCode)

	uc.logger.Infow("telegram binding created",
		"user_id", userID,
		"telegram_user_id", telegramUserID,
	)

	return &dto.TelegramBindingResponse{
		SID:              binding.SID(),
		TelegramUserID:   binding.TelegramUserID(),
		TelegramUsername: binding.TelegramUsername(),
		NotifyExpiring:   binding.NotifyExpiring(),
		NotifyTraffic:    binding.NotifyTraffic(),
		ExpiringDays:     binding.ExpiringDays(),
		TrafficThreshold: binding.TrafficThreshold(),
		CreatedAt:        binding.CreatedAt(),
	}, nil
}
