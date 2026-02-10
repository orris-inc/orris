package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	dto "github.com/orris-inc/orris/internal/shared/hubprotocol/forward"
	nodedto "github.com/orris-inc/orris/internal/shared/hubprotocol/node"

	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	hubAgentCommandChannel = "orris:hub:agent:command"
	hubNodeCommandChannel  = "orris:hub:node:command"
	hubStatusChannel       = "orris:hub:status"
)

// HubEventType represents the type of hub event.
type HubEventType string

const (
	HubEventAgentCommand HubEventType = "agent_command"
	HubEventNodeCommand  HubEventType = "node_command"
	HubEventAgentOnline  HubEventType = "agent_online"
	HubEventAgentOffline HubEventType = "agent_offline"
	HubEventNodeOnline   HubEventType = "node_online"
	HubEventNodeOffline  HubEventType = "node_offline"
)

// HubStatusEvent represents a status event for cross-instance relay.
type HubStatusEvent struct {
	Type       HubEventType `json:"type"`
	AgentID    uint         `json:"agent_id,omitempty"`
	NodeID     uint         `json:"node_id,omitempty"`
	AgentSID   string       `json:"agent_sid,omitempty"`
	AgentName  string       `json:"agent_name,omitempty"`
	NodeSID    string       `json:"node_sid,omitempty"`
	NodeName   string       `json:"node_name,omitempty"`
	Timestamp  int64        `json:"timestamp"`
	InstanceID string       `json:"instance_id,omitempty"` // Source instance ID to avoid self-delivery
}

// AgentCommandEvent wraps a command to be forwarded to an agent via Redis PubSub.
type AgentCommandEvent struct {
	AgentID uint             `json:"agent_id"`
	Command *dto.CommandData `json:"command"`
}

// NodeCommandEvent wraps a command to be forwarded to a node via Redis PubSub.
type NodeCommandEvent struct {
	NodeID  uint                    `json:"node_id"`
	Command *nodedto.NodeCommandData `json:"command"`
}

// HubCommandPublisher defines the interface for publishing hub commands across instances.
type HubCommandPublisher interface {
	PublishAgentCommand(ctx context.Context, agentID uint, cmd *dto.CommandData) error
	PublishNodeCommand(ctx context.Context, nodeID uint, cmd *nodedto.NodeCommandData) error
	PublishStatusEvent(ctx context.Context, event HubStatusEvent) error
}

// HubCommandSubscriber defines the interface for subscribing to hub commands.
type HubCommandSubscriber interface {
	SubscribeAgentCommands(ctx context.Context, handler func(agentID uint, cmd *dto.CommandData)) error
	SubscribeNodeCommands(ctx context.Context, handler func(nodeID uint, cmd *nodedto.NodeCommandData)) error
	SubscribeStatusEvents(ctx context.Context, handler func(event HubStatusEvent)) error
}

// HubEventBus combines publisher and subscriber interfaces.
type HubEventBus interface {
	HubCommandPublisher
	HubCommandSubscriber
}

// RedisHubEventBus implements HubEventBus using Redis Pub/Sub.
type RedisHubEventBus struct {
	client     *redis.Client
	logger     logger.Interface
	instanceID string // Unique ID for this instance to avoid self-delivery of status events
}

// NewRedisHubEventBus creates a new Redis-based hub event bus.
func NewRedisHubEventBus(client *redis.Client, logger logger.Interface) *RedisHubEventBus {
	return &RedisHubEventBus{
		client:     client,
		logger:     logger,
		instanceID: uuid.NewString(),
	}
}

// PublishAgentCommand publishes an agent command to Redis for cross-instance delivery.
func (b *RedisHubEventBus) PublishAgentCommand(ctx context.Context, agentID uint, cmd *dto.CommandData) error {
	event := AgentCommandEvent{
		AgentID: agentID,
		Command: cmd,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal agent command event: %w", err)
	}

	if err := b.client.Publish(ctx, hubAgentCommandChannel, data).Err(); err != nil {
		b.logger.Errorw("failed to publish agent command",
			"agent_id", agentID,
			"action", cmd.Action,
			"error", err,
		)
		return fmt.Errorf("failed to publish agent command: %w", err)
	}

	b.logger.Debugw("agent command published to Redis",
		"agent_id", agentID,
		"action", cmd.Action,
	)
	return nil
}

// PublishNodeCommand publishes a node command to Redis for cross-instance delivery.
func (b *RedisHubEventBus) PublishNodeCommand(ctx context.Context, nodeID uint, cmd *nodedto.NodeCommandData) error {
	event := NodeCommandEvent{
		NodeID:  nodeID,
		Command: cmd,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal node command event: %w", err)
	}

	if err := b.client.Publish(ctx, hubNodeCommandChannel, data).Err(); err != nil {
		b.logger.Errorw("failed to publish node command",
			"node_id", nodeID,
			"action", cmd.Action,
			"error", err,
		)
		return fmt.Errorf("failed to publish node command: %w", err)
	}

	b.logger.Debugw("node command published to Redis",
		"node_id", nodeID,
		"action", cmd.Action,
	)
	return nil
}

// PublishStatusEvent publishes a status event (online/offline) to Redis.
// The instance ID is automatically set to avoid self-delivery.
func (b *RedisHubEventBus) PublishStatusEvent(ctx context.Context, event HubStatusEvent) error {
	if event.Timestamp == 0 {
		event.Timestamp = biztime.NowUTC().Unix()
	}
	event.InstanceID = b.instanceID

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal status event: %w", err)
	}

	if err := b.client.Publish(ctx, hubStatusChannel, data).Err(); err != nil {
		b.logger.Errorw("failed to publish hub status event",
			"event_type", event.Type,
			"error", err,
		)
		return fmt.Errorf("failed to publish status event: %w", err)
	}

	b.logger.Debugw("hub status event published to Redis",
		"event_type", event.Type,
	)
	return nil
}

// SubscribeAgentCommands subscribes to agent command events from Redis.
func (b *RedisHubEventBus) SubscribeAgentCommands(ctx context.Context, handler func(agentID uint, cmd *dto.CommandData)) error {
	return b.subscribeWithReconnect(ctx, hubAgentCommandChannel, func(payload string) {
		var event AgentCommandEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			b.logger.Warnw("failed to unmarshal agent command event",
				"payload", payload,
				"error", err,
			)
			return
		}
		handler(event.AgentID, event.Command)
	})
}

// SubscribeNodeCommands subscribes to node command events from Redis.
func (b *RedisHubEventBus) SubscribeNodeCommands(ctx context.Context, handler func(nodeID uint, cmd *nodedto.NodeCommandData)) error {
	return b.subscribeWithReconnect(ctx, hubNodeCommandChannel, func(payload string) {
		var event NodeCommandEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			b.logger.Warnw("failed to unmarshal node command event",
				"payload", payload,
				"error", err,
			)
			return
		}
		handler(event.NodeID, event.Command)
	})
}

// SubscribeStatusEvents subscribes to status events from Redis.
// Events published by this instance are automatically filtered out.
func (b *RedisHubEventBus) SubscribeStatusEvents(ctx context.Context, handler func(event HubStatusEvent)) error {
	return b.subscribeWithReconnect(ctx, hubStatusChannel, func(payload string) {
		var event HubStatusEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			b.logger.Warnw("failed to unmarshal hub status event",
				"payload", payload,
				"error", err,
			)
			return
		}

		// Skip events from own instance to avoid duplicate local broadcasts
		if event.InstanceID == b.instanceID {
			return
		}

		handler(event)
	})
}

// subscribeWithReconnect wraps subscribe with automatic reconnection and exponential backoff.
func (b *RedisHubEventBus) subscribeWithReconnect(ctx context.Context, channel string, handler func(payload string)) error {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		err := b.subscribe(ctx, channel, handler)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		b.logger.Warnw("hub subscription disconnected, reconnecting",
			"channel", channel,
			"error", err,
			"backoff", backoff,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, maxBackoff)
	}
}

// subscribe is a generic Redis Pub/Sub subscriber.
func (b *RedisHubEventBus) subscribe(ctx context.Context, channel string, handler func(payload string)) error {
	pubsub := b.client.Subscribe(ctx, channel)
	defer pubsub.Close()

	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to channel %s: %w", channel, err)
	}

	b.logger.Infow("subscribed to hub event channel",
		"channel", channel,
	)

	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			b.logger.Infow("hub event subscriber stopped",
				"channel", channel,
				"reason", ctx.Err(),
			)
			return ctx.Err()

		case msg, ok := <-ch:
			if !ok {
				b.logger.Warnw("hub event channel closed",
					"channel", channel,
				)
				return nil
			}

			goroutine.SafeGo(b.logger, "hub-event-handler-"+channel, func() {
				handler(msg.Payload)
			})
		}
	}
}
