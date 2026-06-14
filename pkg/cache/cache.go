package cache

import (
	"sync"
	"time"
)

type item struct {
	value      interface{}
	expiresAt time.Time
}

// Cache provides a simple thread‑safe in‑memory cache with TTL support.
type Cache struct {
	mu    sync.RWMutex
	store map[string]item
}

// New creates an empty Cache.
func New() *Cache { return &Cache{store: make(map[string]item)} }

// Set stores a value with the given TTL. ttl <=0 means no expiration.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.store[key] = item{value: value, expiresAt: exp}
}

// Get retrieves a value. The second return indicates whether the key was present and not expired.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	it, ok := c.store[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if it.expiresAt.IsZero() || it.expiresAt.After(time.Now()) {
		return it.value, true
	}
	// expired – delete
	c.mu.Lock()
	delete(c.store, key)
	c.mu.Unlock()
	return nil, false
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.store, key)
	c.mu.Unlock()
}
