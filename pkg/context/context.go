package context

import (
	"sync"
)

// Store holds a thread‑safe map of key/value pairs for a session.
type Store struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// New creates an empty Store.
func New() *Store { return &Store{data: make(map[string]interface{})} }

// Set stores a value under key.
func (s *Store) Set(key string, val interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
}

// Get retrieves a value. The bool reports presence.
func (s *Store) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

// Delete removes a key.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// Keys returns all keys currently stored.
func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}