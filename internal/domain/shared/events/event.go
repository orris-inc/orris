package events

import (
	"time"
)

// DomainEvent represents a domain event interface
type DomainEvent interface {
	// GetAggregateID returns the ID of the aggregate that generated the event
	GetAggregateID() string
	
	// GetEventType returns the type/name of the event
	GetEventType() string
	
	// GetOccurredAt returns when the event occurred
	GetOccurredAt() time.Time
	
	// GetVersion returns the event version for schema evolution
	GetVersion() int
}

// BaseEvent provides common fields for all domain events
type BaseEvent struct {
	AggregateID string    `json:"aggregate_id"`
	EventType   string    `json:"event_type"`
	OccurredAt  time.Time `json:"occurred_at"`
	Version     int       `json:"version"`
}

// GetAggregateID returns the aggregate ID
func (e BaseEvent) GetAggregateID() string {
	return e.AggregateID
}

// GetEventType returns the event type
func (e BaseEvent) GetEventType() string {
	return e.EventType
}

// GetOccurredAt returns when the event occurred
func (e BaseEvent) GetOccurredAt() time.Time {
	return e.OccurredAt
}

// GetVersion returns the event version
func (e BaseEvent) GetVersion() int {
	return e.Version
}

// EventHandler represents a handler for domain events
type EventHandler interface {
	// Handle processes a domain event
	Handle(event DomainEvent) error
	
	// CanHandle checks if this handler can handle the given event type
	CanHandle(eventType string) bool
}

// EventPublisher publishes domain events
type EventPublisher interface {
	// Publish publishes a single event
	Publish(event DomainEvent) error
	
	// PublishAll publishes multiple events
	PublishAll(events []DomainEvent) error
}

// EventSubscriber subscribes to domain events
type EventSubscriber interface {
	// Subscribe registers a handler for specific event types
	Subscribe(eventType string, handler EventHandler) error
	
	// Unsubscribe removes a handler for specific event types
	Unsubscribe(eventType string, handler EventHandler) error
}

// EventDispatcher combines publisher and subscriber functionality
type EventDispatcher interface {
	EventPublisher
	EventSubscriber
	
	// Start starts the event dispatcher
	Start() error
	
	// Stop stops the event dispatcher
	Stop() error
}