package events

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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
	Subscribe(eventType string, handler Handler) (id string, unsubscribe func())
}

// subscriber wraps a handler with a unique ID for reliable unsubscribe.
type subscriber struct {
	id      string
	handler Handler
}

// InProcessBus is a simple in-memory event bus implementation.
type InProcessBus struct {
	mu        sync.RWMutex
	handlers  map[string][]subscriber
	nextID    atomic.Uint64
}

// NewInProcessBus creates a new in-memory event bus.
func NewInProcessBus() *InProcessBus {
	return &InProcessBus{
		handlers: make(map[string][]subscriber),
	}
}

// Publish sends an event to all subscribed handlers. Handlers run sequentially.
func (b *InProcessBus) Publish(ctx context.Context, event BusEvent) error {
	b.mu.RLock()
	subs := b.handlers[event.Type()]
	b.mu.RUnlock()

	for _, sub := range subs {
		if err := sub.handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe registers a handler for the given event type.
// Returns a unique subscriber ID and an unsubscribe function.
func (b *InProcessBus) Subscribe(eventType string, handler Handler) (string, func()) {
	id := fmt.Sprintf("sub-%d", b.nextID.Add(1))
	sub := subscriber{id: id, handler: handler}

	b.mu.Lock()
	b.handlers[eventType] = append(b.handlers[eventType], sub)
	b.mu.Unlock()

	return id, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		subs := b.handlers[eventType]
		for i, s := range subs {
			if s.id == id {
				b.handlers[eventType] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}
}
