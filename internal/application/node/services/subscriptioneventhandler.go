package services

import (
	"context"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/pubsub"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SubscriptionEventHandler handles subscription change events from Redis Pub/Sub
// for cross-instance synchronization
type SubscriptionEventHandler struct {
	subscriptionRepo subscription.SubscriptionRepository
	syncService      *SubscriptionSyncService
	logger           logger.Interface
}

// NewSubscriptionEventHandler creates a new SubscriptionEventHandler
func NewSubscriptionEventHandler(
	subscriptionRepo subscription.SubscriptionRepository,
	syncService *SubscriptionSyncService,
	logger logger.Interface,
) *SubscriptionEventHandler {
	return &SubscriptionEventHandler{
		subscriptionRepo: subscriptionRepo,
		syncService:      syncService,
		logger:           logger,
	}
}

// HandleEvent processes a subscription change event
// This is called for events from other instances via Redis Pub/Sub
func (h *SubscriptionEventHandler) HandleEvent(ctx context.Context, event pubsub.SubscriptionChangeEvent) {
	h.logger.Debugw("handling subscription change event",
		"subscription_id", event.SubscriptionID,
		"subscription_sid", event.SubscriptionSID,
		"change_type", event.ChangeType,
	)

	// Get subscription from database
	sub, err := h.subscriptionRepo.GetByID(ctx, event.SubscriptionID)
	if err != nil {
		h.logger.Warnw("failed to get subscription for event handling",
			"subscription_id", event.SubscriptionID,
			"error", err,
		)
		return
	}

	if sub == nil {
		h.logger.Warnw("subscription not found for event handling",
			"subscription_id", event.SubscriptionID,
		)
		return
	}

	// Convert event type to change type
	var changeType string
	switch event.ChangeType {
	case pubsub.SubscriptionChangeActivation:
		changeType = dto.SubscriptionChangeAdded
	case pubsub.SubscriptionChangeDeactivation:
		changeType = dto.SubscriptionChangeRemoved
	case pubsub.SubscriptionChangeUpdate:
		changeType = dto.SubscriptionChangeUpdated
	default:
		h.logger.Warnw("unknown subscription change type",
			"change_type", event.ChangeType,
		)
		return
	}

	// Notify local nodes about the subscription change
	// This does NOT re-publish the event, avoiding loops
	if err := h.syncService.NotifySubscriptionChange(ctx, sub, changeType); err != nil {
		h.logger.Warnw("failed to notify local nodes of subscription change",
			"subscription_id", event.SubscriptionID,
			"change_type", changeType,
			"error", err,
		)
	}
}

// StartSubscriber starts the subscription event subscriber in a background goroutine
func (h *SubscriptionEventHandler) StartSubscriber(ctx context.Context, subscriber pubsub.SubscriptionEventSubscriber) {
	goroutine.SafeGo(h.logger, "subscription-event-subscriber", func() {
		h.logger.Infow("starting subscription event subscriber")

		err := subscriber.Subscribe(ctx, h.HandleEvent)
		if err != nil && ctx.Err() == nil {
			h.logger.Errorw("subscription event subscriber stopped unexpectedly",
				"error", err,
			)
		}
	})
}
