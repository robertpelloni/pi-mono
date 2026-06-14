package toolregistry

import (
	"sync"
)

// Handler is a generic function type for tool actions.
// Implementations can assert to concrete signatures as needed.
type Handler func(args ...interface{}) (interface{}, error)

// Registry holds a thread‑safe map of tool names to handlers.
type Registry struct {
	mu      sync.RWMutex
	handlers map[string]Handler
}

// New creates an empty registry.
func New() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

// Register adds a handler for a tool name. Overwrites any existing entry.
func (r *Registry) Register(name string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[name] = h
}

// Get retrieves a handler by name. Returns nil if not found.
func (r *Registry) Get(name string) Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.handlers[name]
}

// List returns a slice of all registered tool names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.handlers))
	for k := range r.handlers {
		keys = append(keys, k)
	}
	return keys
}
