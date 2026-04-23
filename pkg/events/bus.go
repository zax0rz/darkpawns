package events

import (
	"context"
	"sync"
)

// BusEvent is a structured message on the bus.
type BusEvent interface {
	Type() string
}

// Handler processes an event.
type Handler func(ctx context.Context, event BusEvent) error

// Bus is a publish/subscribe event bus.
type Bus interface {
	Publish(ctx context.Context, event BusEvent) error
	Subscribe(eventType string, handler Handler) (unsubscribe func())
}

// InProcessBus is a simple in-memory event bus implementation.
type InProcessBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewInProcessBus creates a new in-memory event bus.
func NewInProcessBus() *InProcessBus {
	return &InProcessBus{
		handlers: make(map[string][]Handler),
	}
}

// Publish sends an event to all subscribed handlers. Handlers run sequentially.
func (b *InProcessBus) Publish(ctx context.Context, event BusEvent) error {
	b.mu.RLock()
	handlers := b.handlers[event.Type()]
	b.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe registers a handler for the given event type.
// Returns an unsubscribe function.
func (b *InProcessBus) Subscribe(eventType string, handler Handler) func() {
	b.mu.Lock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.mu.Unlock()

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		handlers := b.handlers[eventType]
		for i, h := range handlers {
			//nolint:govet // comparing function pointers is intentional
			if &h == &handler {
				b.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}
}
