package usecases

import (
	"context"

	subscriptionApp "orris/internal/application/subscription"
	"orris/internal/shared/logger"
)

type SyncSubscriptionPermissionsCommand struct {
	SubscriptionID uint
}

type SyncSubscriptionPermissionsUseCase struct {
	service *subscriptionApp.Service
	logger  logger.Interface
}

func NewSyncSubscriptionPermissionsUseCase(
	service *subscriptionApp.Service,
	logger logger.Interface,
) *SyncSubscriptionPermissionsUseCase {
	return &SyncSubscriptionPermissionsUseCase{
		service: service,
		logger:  logger,
	}
}

func (uc *SyncSubscriptionPermissionsUseCase) Execute(ctx context.Context, cmd SyncSubscriptionPermissionsCommand) error {
	uc.logger.Infow("executing sync subscription permissions use case",
		"subscription_id", cmd.SubscriptionID)

	if err := uc.service.SyncPermissions(ctx, cmd.SubscriptionID); err != nil {
		uc.logger.Errorw("failed to sync subscription permissions",
			"error", err,
			"subscription_id", cmd.SubscriptionID)
		return err
	}

	uc.logger.Infow("subscription permissions synced successfully",
		"subscription_id", cmd.SubscriptionID)

	return nil
}
