package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AdminVerifyCodeVerifier verifies codes and returns user ID for admin binding
type AdminVerifyCodeVerifier interface {
	Verify(ctx context.Context, code string) (uint, error)
	Delete(ctx context.Context, code string) error
}

// BindAdminUseCase handles admin telegram binding via verify code
type BindAdminUseCase struct {
	bindingRepo AdminTelegramBindingRepository
	userRepo    user.Repository
	verifyCode  AdminVerifyCodeVerifier
	logger      logger.Interface
}

// AdminTelegramBindingRepository is a local alias for the domain repository interface
type AdminTelegramBindingRepository = admin.AdminTelegramBindingRepository

// NewBindAdminUseCase creates a new BindAdminUseCase
func NewBindAdminUseCase(
	bindingRepo AdminTelegramBindingRepository,
	userRepo user.Repository,
	verifyCode AdminVerifyCodeVerifier,
	logger logger.Interface,
) *BindAdminUseCase {
	return &BindAdminUseCase{
		bindingRepo: bindingRepo,
		userRepo:    userRepo,
		verifyCode:  verifyCode,
		logger:      logger,
	}
}

// Execute binds an admin telegram account via verify code
func (uc *BindAdminUseCase) Execute(
	ctx context.Context,
	telegramUserID int64,
	telegramUsername string,
	verifyCode string,
) (*dto.AdminTelegramBindingResponse, error) {
	// Verify the code and get userID
	userID, err := uc.verifyCode.Verify(ctx, verifyCode)
	if err != nil {
		uc.logger.Warnw("invalid verify code for admin binding", "error", err)
		return nil, admin.ErrInvalidVerifyCode
	}

	// Check if user is admin
	u, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		uc.logger.Errorw("failed to get user for admin binding", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if u == nil || !u.IsAdmin() {
		uc.logger.Warnw("non-admin user attempted admin binding", "user_id", userID)
		return nil, admin.ErrNotAdmin
	}

	// Check if admin already has binding
	existing, err := uc.bindingRepo.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, admin.ErrBindingNotFound) {
		return nil, fmt.Errorf("failed to check existing binding: %w", err)
	}
	if existing != nil {
		return nil, admin.ErrAlreadyBound
	}

	// Check if telegram account is already used by another admin
	existingTg, err := uc.bindingRepo.GetByTelegramUserID(ctx, telegramUserID)
	if err != nil && !errors.Is(err, admin.ErrBindingNotFound) {
		return nil, fmt.Errorf("failed to check telegram binding: %w", err)
	}
	if existingTg != nil {
		return nil, admin.ErrTelegramAlreadyUsed
	}

	// Create binding
	binding, err := admin.NewAdminTelegramBinding(userID, telegramUserID, telegramUsername, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create admin binding: %w", err)
	}

	if err := uc.bindingRepo.Create(ctx, binding); err != nil {
		return nil, fmt.Errorf("failed to save admin binding: %w", err)
	}

	// Delete the verify code (cleanup)
	_ = uc.verifyCode.Delete(ctx, verifyCode)

	uc.logger.Infow("admin telegram binding created",
		"user_id", userID,
		"telegram_user_id", telegramUserID,
	)

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
