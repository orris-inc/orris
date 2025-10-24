package events

import (
	"fmt"
	"sync"
)

// InMemoryEventDispatcher is an in-memory implementation of EventDispatcher
type InMemoryEventDispatcher struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}
	eventCh  chan DomainEvent
	wg       sync.WaitGroup
}

// NewInMemoryEventDispatcher creates a new in-memory event dispatcher
func NewInMemoryEventDispatcher(bufferSize int) *InMemoryEventDispatcher {
	if bufferSize <= 0 {
		bufferSize = 100
	}

	return &InMemoryEventDispatcher{
		handlers: make(map[string][]EventHandler),
		stopCh:   make(chan struct{}),
		eventCh:  make(chan DomainEvent, bufferSize),
	}
}

// Publish publishes a single event
func (d *InMemoryEventDispatcher) Publish(event DomainEvent) error {
	if !d.running {
		return fmt.Errorf("event dispatcher is not running")
	}

	select {
	case d.eventCh <- event:
		return nil
	default:
		return fmt.Errorf("event channel is full")
	}
}

// PublishAll publishes multiple events
func (d *InMemoryEventDispatcher) PublishAll(events []DomainEvent) error {
	if !d.running {
		return fmt.Errorf("event dispatcher is not running")
	}

	for _, event := range events {
		if err := d.Publish(event); err != nil {
			return fmt.Errorf("failed to publish event %s: %w", event.GetEventType(), err)
		}
	}

	return nil
}

// Subscribe registers a handler for specific event types
func (d *InMemoryEventDispatcher) Subscribe(eventType string, handler EventHandler) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if eventType == "" {
		return fmt.Errorf("event type cannot be empty")
	}

	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	d.handlers[eventType] = append(d.handlers[eventType], handler)
	return nil
}

// Unsubscribe removes a handler for specific event types
func (d *InMemoryEventDispatcher) Unsubscribe(eventType string, handler EventHandler) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	handlers, exists := d.handlers[eventType]
	if !exists {
		return nil
	}

	newHandlers := make([]EventHandler, 0, len(handlers))
	for _, h := range handlers {
		if h != handler {
			newHandlers = append(newHandlers, h)
		}
	}

	if len(newHandlers) == 0 {
		delete(d.handlers, eventType)
	} else {
		d.handlers[eventType] = newHandlers
	}

	return nil
}

// Start starts the event dispatcher
func (d *InMemoryEventDispatcher) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return fmt.Errorf("event dispatcher is already running")
	}

	d.running = true
	d.wg.Add(1)

	go func() {
		defer d.wg.Done()
		d.processEvents()
	}()

	return nil
}

// Stop stops the event dispatcher
func (d *InMemoryEventDispatcher) Stop() error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return fmt.Errorf("event dispatcher is not running")
	}

	d.running = false
	d.mu.Unlock()

	close(d.stopCh)
	d.wg.Wait()

	return nil
}

// processEvents processes events from the event channel
func (d *InMemoryEventDispatcher) processEvents() {
	for {
		select {
		case <-d.stopCh:
			// Drain remaining events
			for {
				select {
				case event := <-d.eventCh:
					d.handleEvent(event)
				default:
					return
				}
			}
		case event := <-d.eventCh:
			d.handleEvent(event)
		}
	}
}

// handleEvent handles a single event
func (d *InMemoryEventDispatcher) handleEvent(event DomainEvent) {
	d.mu.RLock()
	handlers := d.handlers[event.GetEventType()]
	d.mu.RUnlock()

	for _, handler := range handlers {
		if handler.CanHandle(event.GetEventType()) {
			// Handle in goroutine to avoid blocking
			go func(h EventHandler, e DomainEvent) {
				if err := h.Handle(e); err != nil {
					// In production, this should be logged properly
					fmt.Printf("Error handling event %s: %v\n", e.GetEventType(), err)
				}
			}(handler, event)
		}
	}
}

// SimpleEventHandler is a simple implementation of EventHandler
type SimpleEventHandler struct {
	eventType string
	handler   func(DomainEvent) error
}

// NewSimpleEventHandler creates a new simple event handler
func NewSimpleEventHandler(eventType string, handler func(DomainEvent) error) *SimpleEventHandler {
	return &SimpleEventHandler{
		eventType: eventType,
		handler:   handler,
	}
}

// Handle processes a domain event
func (h *SimpleEventHandler) Handle(event DomainEvent) error {
	if h.handler != nil {
		return h.handler(event)
	}
	return nil
}

// CanHandle checks if this handler can handle the given event type
func (h *SimpleEventHandler) CanHandle(eventType string) bool {
	return h.eventType == eventType
}
