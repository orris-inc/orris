package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/application/telegram/dto"
	"github.com/orris-inc/orris/internal/domain/telegram"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdatePreferencesUseCase updates notification preferences
type UpdatePreferencesUseCase struct {
	bindingRepo telegram.TelegramBindingRepository
	logger      logger.Interface
}

// NewUpdatePreferencesUseCase creates a new UpdatePreferencesUseCase
func NewUpdatePreferencesUseCase(
	bindingRepo telegram.TelegramBindingRepository,
	logger logger.Interface,
) *UpdatePreferencesUseCase {
	return &UpdatePreferencesUseCase{
		bindingRepo: bindingRepo,
		logger:      logger,
	}
}

// Execute updates notification preferences
func (uc *UpdatePreferencesUseCase) Execute(
	ctx context.Context,
	userID uint,
	req dto.UpdatePreferencesRequest,
) (*dto.TelegramBindingResponse, error) {
	binding, err := uc.bindingRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates with current values as defaults
	notifyExpiring := binding.NotifyExpiring()
	notifyTraffic := binding.NotifyTraffic()
	expiringDays := binding.ExpiringDays()
	trafficThreshold := binding.TrafficThreshold()

	if req.NotifyExpiring != nil {
		notifyExpiring = *req.NotifyExpiring
	}
	if req.NotifyTraffic != nil {
		notifyTraffic = *req.NotifyTraffic
	}
	if req.ExpiringDays != nil {
		expiringDays = *req.ExpiringDays
	}
	if req.TrafficThreshold != nil {
		trafficThreshold = *req.TrafficThreshold
	}

	if err := binding.UpdatePreferences(notifyExpiring, notifyTraffic, expiringDays, trafficThreshold); err != nil {
		return nil, err
	}

	if err := uc.bindingRepo.Update(ctx, binding); err != nil {
		return nil, err
	}

	uc.logger.Infow("telegram preferences updated", "user_id", userID)

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
