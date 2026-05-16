package eventbus

import "sync"

// Listener wraps a callback with an ID for comparison.
type listener struct {
	id       int
	callback func(any)
}

// EventBus provides a simple publish/subscribe event system.
type EventBus struct {
	mu        sync.RWMutex
	listeners map[string][]listener
	nextID    int
}

// New creates a new EventBus.
func New() *EventBus {
	return &EventBus{
		listeners: make(map[string][]listener),
	}
}

// On subscribes to an event. Returns an unsubscribe function.
func On[T any](bus *EventBus, eventType string, fn func(T)) func() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	id := bus.nextID
	bus.nextID++

	wrapper := listener{
		id: id,
		callback: func(v any) {
			if typed, ok := v.(T); ok {
				fn(typed)
			}
		},
	}
	bus.listeners[eventType] = append(bus.listeners[eventType], wrapper)

	return func() {
		bus.mu.Lock()
		defer bus.mu.Unlock()
		listeners := bus.listeners[eventType]
		for i, l := range listeners {
			if l.id == id {
				bus.listeners[eventType] = append(listeners[:i], listeners[i+1:]...)
				return
			}
		}
	}
}

// Emit publishes an event to all subscribers.
func Emit[T any](bus *EventBus, eventType string, event T) {
	bus.mu.RLock()
	listeners := make([]listener, len(bus.listeners[eventType]))
	copy(listeners, bus.listeners[eventType])
	bus.mu.RUnlock()

	for _, l := range listeners {
		l.callback(event)
	}
}

// Once subscribes to an event for a single invocation.
func Once[T any](bus *EventBus, eventType string, fn func(T)) func() {
	var unsubscribe func()
	unsubscribe = On(bus, eventType, func(e T) {
		unsubscribe()
		fn(e)
	})
	return unsubscribe
}

// Clear removes all listeners.
func (bus *EventBus) Clear() {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.listeners = make(map[string][]listener)
}

// ListenerCount returns the number of listeners for an event type.
func (bus *EventBus) ListenerCount(eventType string) int {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	return len(bus.listeners[eventType])
}
