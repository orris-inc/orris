package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orris-inc/orris/internal/shared/logger"
)

// SubscriptionChangeType represents the type of subscription change event
type SubscriptionChangeType string

const (
	// SubscriptionChangeActivation indicates a subscription was activated
	SubscriptionChangeActivation SubscriptionChangeType = "activation"
	// SubscriptionChangeDeactivation indicates a subscription was deactivated
	SubscriptionChangeDeactivation SubscriptionChangeType = "deactivation"
	// SubscriptionChangeUpdate indicates a subscription was updated
	SubscriptionChangeUpdate SubscriptionChangeType = "update"
)

// SubscriptionChangeEvent represents a subscription change event for cross-instance synchronization
type SubscriptionChangeEvent struct {
	SubscriptionID  uint                   `json:"subscription_id"`
	SubscriptionSID string                 `json:"subscription_sid"`
	ChangeType      SubscriptionChangeType `json:"change_type"`
	Timestamp       int64                  `json:"timestamp"`
}

// SubscriptionEventHandler is a callback function for handling subscription events
type SubscriptionEventHandler func(ctx context.Context, event SubscriptionChangeEvent)

// SubscriptionEventPublisher defines the interface for publishing subscription events
type SubscriptionEventPublisher interface {
	PublishActivation(ctx context.Context, subscriptionID uint, subscriptionSID string) error
	PublishDeactivation(ctx context.Context, subscriptionID uint, subscriptionSID string) error
	PublishUpdate(ctx context.Context, subscriptionID uint, subscriptionSID string) error
}

// SubscriptionEventSubscriber defines the interface for subscribing to subscription events
type SubscriptionEventSubscriber interface {
	Subscribe(ctx context.Context, handler SubscriptionEventHandler) error
}

const subscriptionChangeChannel = "orris:subscription:change"

// RedisSubscriptionEventBus implements both SubscriptionEventPublisher and SubscriptionEventSubscriber
// using Redis Pub/Sub for cross-instance event distribution
type RedisSubscriptionEventBus struct {
	client *redis.Client
	logger logger.Interface
}

// NewRedisSubscriptionEventBus creates a new Redis-based subscription event bus
func NewRedisSubscriptionEventBus(client *redis.Client, logger logger.Interface) *RedisSubscriptionEventBus {
	return &RedisSubscriptionEventBus{
		client: client,
		logger: logger,
	}
}

// PublishActivation publishes a subscription activation event
func (b *RedisSubscriptionEventBus) PublishActivation(ctx context.Context, subscriptionID uint, subscriptionSID string) error {
	return b.publish(ctx, SubscriptionChangeEvent{
		SubscriptionID:  subscriptionID,
		SubscriptionSID: subscriptionSID,
		ChangeType:      SubscriptionChangeActivation,
		Timestamp:       time.Now().Unix(),
	})
}

// PublishDeactivation publishes a subscription deactivation event
func (b *RedisSubscriptionEventBus) PublishDeactivation(ctx context.Context, subscriptionID uint, subscriptionSID string) error {
	return b.publish(ctx, SubscriptionChangeEvent{
		SubscriptionID:  subscriptionID,
		SubscriptionSID: subscriptionSID,
		ChangeType:      SubscriptionChangeDeactivation,
		Timestamp:       time.Now().Unix(),
	})
}

// PublishUpdate publishes a subscription update event
func (b *RedisSubscriptionEventBus) PublishUpdate(ctx context.Context, subscriptionID uint, subscriptionSID string) error {
	return b.publish(ctx, SubscriptionChangeEvent{
		SubscriptionID:  subscriptionID,
		SubscriptionSID: subscriptionSID,
		ChangeType:      SubscriptionChangeUpdate,
		Timestamp:       time.Now().Unix(),
	})
}

func (b *RedisSubscriptionEventBus) publish(ctx context.Context, event SubscriptionChangeEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := b.client.Publish(ctx, subscriptionChangeChannel, data).Err(); err != nil {
		b.logger.Errorw("failed to publish subscription change event",
			"subscription_id", event.SubscriptionID,
			"subscription_sid", event.SubscriptionSID,
			"change_type", event.ChangeType,
			"error", err,
		)
		return fmt.Errorf("failed to publish event: %w", err)
	}

	b.logger.Debugw("subscription change event published",
		"subscription_id", event.SubscriptionID,
		"subscription_sid", event.SubscriptionSID,
		"change_type", event.ChangeType,
	)
	return nil
}

// Subscribe subscribes to subscription change events and calls the handler for each event
func (b *RedisSubscriptionEventBus) Subscribe(ctx context.Context, handler SubscriptionEventHandler) error {
	pubsub := b.client.Subscribe(ctx, subscriptionChangeChannel)
	defer pubsub.Close()

	// Wait for subscription confirmation
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to channel: %w", err)
	}

	b.logger.Infow("subscribed to subscription change events",
		"channel", subscriptionChangeChannel,
	)

	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			b.logger.Infow("subscription event subscriber stopped",
				"reason", ctx.Err(),
			)
			return ctx.Err()

		case msg, ok := <-ch:
			if !ok {
				b.logger.Warnw("subscription event channel closed")
				return nil
			}

			var event SubscriptionChangeEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				b.logger.Warnw("failed to unmarshal subscription event",
					"payload", msg.Payload,
					"error", err,
				)
				continue
			}

			// Handle event in background goroutine to avoid blocking the event loop
			// Use Background context to decouple event handling from subscriber lifecycle
			go handler(context.Background(), event)
		}
	}
}
