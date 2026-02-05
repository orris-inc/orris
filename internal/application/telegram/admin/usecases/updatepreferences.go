package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
	"github.com/orris-inc/orris/internal/domain/telegram/admin"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdateAdminPreferencesUseCase updates admin notification preferences
type UpdateAdminPreferencesUseCase struct {
	bindingRepo AdminTelegramBindingRepository
	logger      logger.Interface
}

// NewUpdateAdminPreferencesUseCase creates a new UpdateAdminPreferencesUseCase
func NewUpdateAdminPreferencesUseCase(
	bindingRepo AdminTelegramBindingRepository,
	logger logger.Interface,
) *UpdateAdminPreferencesUseCase {
	return &UpdateAdminPreferencesUseCase{
		bindingRepo: bindingRepo,
		logger:      logger,
	}
}

// Execute updates admin notification preferences
func (uc *UpdateAdminPreferencesUseCase) Execute(
	ctx context.Context,
	userID uint,
	req dto.UpdateAdminPreferencesRequest,
) (*dto.AdminTelegramBindingResponse, error) {
	binding, err := uc.bindingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, admin.ErrBindingNotFound
	}

	// Apply updates using domain method
	if err := binding.UpdatePreferences(
		req.NotifyNodeOffline,
		req.NotifyAgentOffline,
		req.NotifyNewUser,
		req.NotifyPaymentSuccess,
		req.NotifyDailySummary,
		req.NotifyWeeklySummary,
		req.OfflineThresholdMinutes,
		req.NotifyResourceExpiring,
		req.ResourceExpiringDays,
		req.DailySummaryHour,
		req.WeeklySummaryHour,
		req.WeeklySummaryWeekday,
		req.OfflineCheckIntervalMinutes,
	); err != nil {
		return nil, err
	}

	if err := uc.bindingRepo.Update(ctx, binding); err != nil {
		return nil, err
	}

	uc.logger.Infow("admin telegram preferences updated", "user_id", userID)

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
